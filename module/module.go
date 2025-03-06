package module

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"path/filepath"
	"strconv"

	"cosmossdk.io/core/appmodule"
	"cosmossdk.io/math"
	gwruntime "github.com/grpc-ecosystem/grpc-gateway/runtime"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"

	"github.com/janction/videoRendering"
	"github.com/janction/videoRendering/ipfs"
	"github.com/janction/videoRendering/keeper"
)

var (
	_ module.AppModuleBasic = AppModule{}
	_ module.HasGenesis     = AppModule{}
	_ appmodule.AppModule   = AppModule{}
)

// ConsensusVersion defines the current module consensus version.
const ConsensusVersion = 1

type AppModule struct {
	cdc    codec.Codec
	keeper keeper.Keeper
}

// NewAppModule creates a new AppModule object
func NewAppModule(cdc codec.Codec, keeper keeper.Keeper) AppModule {
	return AppModule{
		cdc:    cdc,
		keeper: keeper,
	}
}

func NewAppModuleBasic(m AppModule) module.AppModuleBasic {
	return module.CoreAppModuleBasicAdaptor(m.Name(), m)
}

// Name returns the videoRendering module's name.
func (AppModule) Name() string { return videoRendering.ModuleName }

// RegisterLegacyAminoCodec registers the videoRendering module's types on the LegacyAmino codec.
// New modules do not need to support Amino.
func (AppModule) RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {}

// RegisterGRPCGatewayRoutes registers the gRPC Gateway routes for the videoRendering module.
func (AppModule) RegisterGRPCGatewayRoutes(clientCtx client.Context, mux *gwruntime.ServeMux) {
	if err := videoRendering.RegisterQueryHandlerClient(context.Background(), mux, videoRendering.NewQueryClient(clientCtx)); err != nil {
		panic(err)
	}
}

// RegisterInterfaces registers interfaces and implementations of the videoRendering module.
func (AppModule) RegisterInterfaces(registry codectypes.InterfaceRegistry) {
	videoRendering.RegisterInterfaces(registry)
}

// ConsensusVersion implements AppModule/ConsensusVersion.
func (AppModule) ConsensusVersion() uint64 { return ConsensusVersion }

// RegisterServices registers a gRPC query service to respond to the module-specific gRPC queries.
func (am AppModule) RegisterServices(cfg module.Configurator) {
	// Register servers
	videoRendering.RegisterMsgServer(cfg.MsgServer(), keeper.NewMsgServerImpl(am.keeper))
	videoRendering.RegisterQueryServer(cfg.QueryServer(), keeper.NewQueryServerImpl(am.keeper))

	// Register in place module state migration migrations
	// m := keeper.NewMigrator(am.keeper)
	// if err := cfg.RegisterMigration(videoRendering.ModuleName, 1, m.Migrate1to2); err != nil {
	//     panic(fmt.Sprintf("failed to migrate x/%s from version 1 to 2: %v", videoRendering.ModuleName, err))
	// }
}

// DefaultGenesis returns default genesis state as raw bytes for the module.
func (AppModule) DefaultGenesis(cdc codec.JSONCodec) json.RawMessage {
	return cdc.MustMarshalJSON(videoRendering.NewGenesisState())
}

// ValidateGenesis performs genesis state validation for the circuit module.
func (AppModule) ValidateGenesis(cdc codec.JSONCodec, _ client.TxEncodingConfig, bz json.RawMessage) error {
	var data videoRendering.GenesisState
	if err := cdc.UnmarshalJSON(bz, &data); err != nil {
		return fmt.Errorf("failed to unmarshal %s genesis state: %w", videoRendering.ModuleName, err)
	}
	return data.Validate()
}

// InitGenesis performs genesis initialization for the videoRendering module.
// It returns no validator updates.
func (am AppModule) InitGenesis(ctx sdk.Context, cdc codec.JSONCodec, data json.RawMessage) {
	var genesisState videoRendering.GenesisState
	cdc.MustUnmarshalJSON(data, &genesisState)

	if err := am.keeper.InitGenesis(ctx, &genesisState); err != nil {
		panic(fmt.Sprintf("failed to initialize %s genesis state: %v", videoRendering.ModuleName, err))
	}
}

// ExportGenesis returns the exported genesis state as raw bytes for the circuit
// module.
func (am AppModule) ExportGenesis(ctx sdk.Context, cdc codec.JSONCodec) json.RawMessage {
	gs, err := am.keeper.ExportGenesis(ctx)
	if err != nil {
		panic(fmt.Sprintf("failed to export %s genesis state: %v", videoRendering.ModuleName, err))
	}

	return cdc.MustMarshalJSON(gs)
}

func (am AppModule) getPendingVideoRenderingTask(ctx context.Context) (bool, videoRendering.VideoRenderingTask) {
	// TODO move this to global parameters
	params, _ := am.keeper.Params.Get(ctx)
	ti, err := am.keeper.VideoRenderingTaskInfo.Get(ctx)

	if err != nil {
		panic(err)
	}
	nextId := int(ti.NextId)
	for i := 0; i < nextId; i++ {
		task, err := am.keeper.VideoRenderingTasks.Get(ctx, strconv.Itoa(i))
		if err != nil {
			continue
		}

		// we only search for in progress and with the reward this node will accept
		if !task.Completed && task.Reward.Amount.GTE(math.NewInt(am.keeper.Configuration.MinReward)) {
			for _, value := range task.Threads {
				if !value.Completed && len(value.Workers) < int(params.MaxWorkersPerThread) {
					return true, task
				}
			}
		}
	}
	return false, videoRendering.VideoRenderingTask{}
}

func (am AppModule) BeginBlock(ctx context.Context) error {
	k := am.keeper

	// we adjust the amount of minimum validators per thread based on the amount of
	// registered workers
	params, _ := k.Params.Get(ctx)
	count := 0
	iterator, _ := k.Workers.Iterate(ctx, nil)
	for iterator.Valid() {
		count++
		iterator.Next()
	}
	// we adjust the min validators dinamycally. Max 7
	if count > 1 && count < 7 {
		params.MinValidators = int64(count)
		k.Params.Set(ctx, params)
	}

	if k.Configuration.Enabled && k.Configuration.WorkerAddress != "" {
		worker, _ := k.Workers.Get(ctx, k.Configuration.WorkerAddress)
		if worker.Enabled && worker.CurrentTaskId != "" {
			// we have to start some work!
			task, err := k.VideoRenderingTasks.Get(ctx, worker.CurrentTaskId)
			if err != nil {
				log.Printf("error processing task %v. %v", task.TaskId, err.Error())
				return nil
			}
			thread := *task.Threads[worker.CurrentThreadIndex]
			dbThread, _ := k.DB.ReadThread(thread.ThreadId)
			log.Printf("local thread is %s, %v, %v, %v, %v", dbThread.ID, dbThread.WorkStarted, dbThread.WorkCompleted, dbThread.SolutionProposed, dbThread.VerificationStarted)

			workPath := filepath.Join(k.Configuration.RootPath, "renders", thread.ThreadId)

			if thread.Solution == nil && !dbThread.WorkStarted {
				log.Printf("thread %v of task %v started", thread.ThreadId, task.TaskId)
				go thread.StartWork(worker.Address, task.Cid, workPath, &k.DB)
			}

			if thread.Solution == nil && dbThread.WorkCompleted && !dbThread.SolutionProposed {
				log.Printf("thread %v of task %v started", thread.ThreadId, task.TaskId)
				go thread.ProposeSolution(ctx, worker.Address, workPath, &k.DB, k.ProvingKeyPath)
			}

			if thread.Solution != nil && !dbThread.VerificationStarted {
				// start verification
				go thread.Verify(ctx, worker.Address, workPath, &k.DB, k.ProvingKeyPath)
			}
		}
	}

	return nil
}

// EndBlock contains the logic that is automatically triggered at the end of each block.
// The end block implementation is optional.
func (am AppModule) EndBlock(ctx context.Context) error {
	k := am.keeper

	// we validate if this node is enabled to perform work
	if k.Configuration.Enabled && k.Configuration.WorkerAddress != "" {
		// we validate if the worker is idle
		worker, _ := k.Workers.Get(ctx, k.Configuration.WorkerAddress)

		if worker.Address == "" {
			isRegistered, _ := k.DB.IsWorkerRegistered(k.Configuration.WorkerAddress)
			if !isRegistered {
				// the worker is not registered, so we do it with the stake
				params, _ := am.keeper.Params.Get(ctx)
				go worker.RegisterWorker(k.Configuration.WorkerAddress, *params.MinWorkerStaking, &k.DB)
			}
		}

		if worker.Enabled && worker.CurrentTaskId == "" {
			// we find any task in progress that has enought reward
			log.Printf(" worker %v is idle ", worker.Address)
			found, task := am.getPendingVideoRenderingTask(ctx)
			if found {
				log.Printf(" registering worker %v in task %v ", worker.Address, task.TaskId)
				go task.SubscribeWorkerToTask(ctx, worker.Address)

			}
		}
	}

	params, _ := am.keeper.Params.Get(ctx)
	maxId, _ := k.VideoRenderingTaskInfo.Get(ctx)
	for i := 0; i < int(maxId.NextId); i++ {
		task, _ := k.VideoRenderingTasks.Get(ctx, strconv.Itoa(i))
		if !task.Completed {
			for _, thread := range task.Threads {
				if len(thread.Validations) >= int(params.MinValidators) && !thread.Completed && thread.Solution.ProposedBy == am.keeper.Configuration.WorkerAddress {

					db, _ := k.DB.ReadThread(thread.ThreadId)
					if !db.SolutionRevealed {
						// We have reached enought validations, if we are the winning node, is time to reveal the solution
						log.Printf("Time to reveal solution!!!!!!")
						go thread.RevealSolution(am.keeper.Configuration.RootPath, &k.DB)
					}
					// log.Println("we are ready to validate", thread.ThreadId)
					// am.EvaluateCompletedThread(ctx, &task, k)

					// // PAY to validators

					// // if we are the node that proposed the solution, then we upload it
					// if thread.Solution.ProposedBy == am.keeper.Configuration.WorkerAddress {
					// 	localThread, _ := am.keeper.DB.ReadThread(thread.ThreadId)
					// 	if !localThread.SubmitionStarted {
					// 		go thread.SubmitSolution(ctx, am.keeper.Configuration.WorkerAddress, am.keeper.Configuration.RootPath, &am.keeper.DB)
					// 	}
					// }
				}
			}
		}
	}

	for i := 0; i < int(maxId.NextId); i++ {
		task, _ := k.VideoRenderingTasks.Get(ctx, strconv.Itoa(i))
		if !task.Completed {
			completed := true
			for _, thread := range task.Threads {
				if !thread.Completed {
					// we found at least one thread not completed, so task isn't complete
					completed = false
					break
				}
			}
			if completed {
				// all threads are over, we mark the task as completed
				task.Completed = true
				k.VideoRenderingTasks.Set(ctx, task.TaskId, task)
			}
		}
	}

	// we now will connect to the IPFS nodes of new workers
	k.Workers.Walk(ctx, nil, func(address string, worker videoRendering.Worker) (stop bool, err error) {
		isAdded, _ := k.DB.IsIPFSWorkerAdded(address)
		if worker.IpfsId != "" && worker.PublicIp != "" && !isAdded {
			log.Printf("Connecting to IPFS node %s at %s", worker.IpfsId, worker.PublicIp)
			ipfs.EnsureIPFSRunning()
			go ipfs.ConnectToIPFSNode(worker.PublicIp, worker.IpfsId)

			// ✅ Mark worker as processed
			am.keeper.DB.AddIPFSWorker(address)
			return true, nil
		}
		return false, nil // Continue iterating
	})

	return nil
}

func (am AppModule) EvaluateCompletedThread(ctx context.Context, task *videoRendering.VideoRenderingTask, index int) error {
	//TODO  implement validations. What happens if a validation is false?
	thread := task.Threads[index]

	for _, worker := range thread.Workers {
		// we reset all workers
		worker, _ := am.keeper.Workers.Get(ctx, worker)
		worker.CurrentTaskId = ""
		worker.CurrentThreadIndex = 0
		// we increase reputation
		worker.Reputation.Points = worker.Reputation.Points + 1
		worker.Reputation.Validations = worker.Reputation.Validations + 1
		// we pay for the validation
		winning := thread.GetValidatorReward(worker.Address, task.GetValidatorsReward())
		worker.Reputation.Winnings = worker.Reputation.Winnings.Add(winning)
		am.keeper.Workers.Set(ctx, worker.Address, worker)
	}

	task.Threads[index].Completed = true
	am.keeper.VideoRenderingTasks.Set(ctx, task.TaskId, *task)

	return nil
}
