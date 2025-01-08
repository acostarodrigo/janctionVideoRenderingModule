package module

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strconv"

	"cosmossdk.io/core/appmodule"
	gwruntime "github.com/grpc-ecosystem/grpc-gateway/runtime"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"

	"github.com/janction/videoRendering"
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
		if task.InProgress && task.Reward >= uint64(am.keeper.Configuration.MinReward) {
			return true, task
		}
	}
	return false, videoRendering.VideoRenderingTask{}
}

func (am AppModule) BeginBlock(ctx context.Context) error {
	// TODO get task I'm susbcribed to, and start working on it
	return nil
}

// EndBlock contains the logic that is automatically triggered at the end of each block.
// The end block implementation is optional.
func (am AppModule) EndBlock(ctx context.Context) error {
	k := am.keeper

	// we validate if this node is enabled to perform work
	if k.Configuration.Enabled {
		// we get the address of the worker's alias and set it on the keeper's configuration
		conf := &am.keeper.Configuration
		address := keeper.GetWorkerAddress(conf.WorkerName)
		am.keeper.Configuration.WorkerAddress = address

		// we validate if the worker is idle
		worker, _ := k.Workers.Get(ctx, k.Configuration.WorkerAddress)
		if worker.Enabled {
			if worker.Status == videoRendering.Worker_WORKER_STATUS_IDLE {
				// we find any task in progress that has enought reward
				log.Printf(" worker %v is idle ", worker.Address)
				found, task := am.getPendingVideoRenderingTask(ctx)
				if found {
					task.SubscribeWorkerToTask(ctx, worker.Address)
				}
			} else {
				// TODO validate the node is actually doing some work.
				log.Printf(" worker %v is doing work ", worker.Address)
			}
		}
	}

	return nil
}
