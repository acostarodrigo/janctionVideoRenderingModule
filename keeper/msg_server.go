package keeper

import (
	"context"
	"slices"
	"strconv"
	"strings"

	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/ipfs/go-cid"

	"github.com/janction/videoRendering"
)

type msgServer struct {
	k Keeper
}

var _ videoRendering.MsgServer = msgServer{}

// NewMsgServerImpl returns an implementation of the module MsgServer interface.
func NewMsgServerImpl(keeper Keeper) videoRendering.MsgServer {
	return &msgServer{k: keeper}
}

// CreateGame defines the handler for the MsgCreateVideoRenderingTask message.
func (ms msgServer) CreateVideoRenderingTask(ctx context.Context, msg *videoRendering.MsgCreateVideoRenderingTask) (*videoRendering.MsgCreateVideoRenderingTaskResponse, error) {
	ms.k.logger.Info("CreateVideoRenderingTask -  creator: %s, cid: %s, startFrame: %v, endFrame: %v, threads: %v, reward: %s", msg.Creator, msg.Cid, msg.StartFrame, msg.EndFrame, msg.Threads, msg.Reward)

	// TODO had validations about the parameters
	taskInfo, err := ms.k.VideoRenderingTaskInfo.Get(ctx)
	if err != nil {
		ms.k.logger.Error("Getting task: %s", err.Error())
		return nil, err
	}

	// TODO reward must be valid

	// we make sure the provided CID is valid
	_, err = cid.Decode(msg.Cid)
	if err != nil {
		ms.k.logger.Error("provided cid is invalid: (%s)", msg.Cid)
		return nil, sdkerrors.ErrAppConfig.Wrapf(videoRendering.ErrInvalidVideoRenderingTask.Error(), "cid %s is invalid", msg.Cid)
	}

	var nextId = taskInfo.NextId
	// we get the taskId in string
	taskId := strconv.FormatInt(nextId, 10)

	// and increase the task id counter for next task
	nextId++
	ms.k.VideoRenderingTaskInfo.Set(ctx, videoRendering.VideoRenderingTaskInfo{NextId: nextId})

	videoTask := videoRendering.VideoRenderingTask{TaskId: taskId, Requester: msg.Creator, Cid: msg.Cid, StartFrame: msg.StartFrame, EndFrame: msg.EndFrame, Completed: false, ThreadAmount: msg.Threads, Reward: msg.Reward}
	threads := videoTask.GenerateThreads(taskId)
	videoTask.Threads = threads

	// the module will keep the reward to be distributed later
	// TODO Add validations for msg.Creator
	addr, _ := types.AccAddressFromBech32(msg.Creator)
	ms.k.BankKeeper.SendCoinsFromAccountToModule(ctx, addr, videoRendering.ModuleName, types.NewCoins(*msg.Reward))

	// we create the task
	if err := ms.k.VideoRenderingTasks.Set(ctx, taskId, videoTask); err != nil {
		return nil, err
	}
	return &videoRendering.MsgCreateVideoRenderingTaskResponse{TaskId: taskId}, nil
}

func (ms msgServer) AddWorker(ctx context.Context, msg *videoRendering.MsgAddWorker) (*videoRendering.MsgAddWorkerResponse, error) {
	ms.k.logger.Info("AddWorker - creator: %s, ipfsId: %s, publicIp: %s, stake: %s", msg.Creator, msg.IpfsId, msg.PublicIp, msg.Stake)

	found, err := ms.k.Workers.Has(ctx, msg.Creator)
	if err != nil {
		ms.k.logger.Error("Worker exists?: %s", err.Error())
		return &videoRendering.MsgAddWorkerResponse{Ok: false, Message: err.Error()}, err
	}

	if found {
		ms.k.logger.Error("Worker %v already exists.", msg.Creator)
		error := sdkerrors.ErrAppConfig.Wrapf(videoRendering.ErrWorkerAlreadyRegistered.Error(), "worker (%s) is already registered", msg.Creator)
		return &videoRendering.MsgAddWorkerResponse{Ok: false, Message: error.Error()}, error
	}

	// we verify the staking amount if valid and at least equeal the min value
	params, _ := ms.k.Params.Get(ctx)
	if msg.Stake.Denom != params.MinWorkerStaking.Denom {
		error := sdkerrors.ErrAppConfig.Wrapf(videoRendering.ErrWorkerIncorrectStake.Error(), "staked coin denom %s is not accepted", msg.Stake.Denom)
		ms.k.logger.Error(error.Error())
		return &videoRendering.MsgAddWorkerResponse{Ok: false, Message: error.Error()}, error
	}

	if msg.Stake.Amount.LT(params.MinWorkerStaking.Amount) {
		error := sdkerrors.ErrAppConfig.Wrapf(videoRendering.ErrWorkerIncorrectStake.Error(), "staked coin is not enought. Min value is %v", params.MinWorkerStaking.Amount)
		ms.k.logger.Error(error.Error())
		return &videoRendering.MsgAddWorkerResponse{Ok: false, Message: error.Error()}, error
	}

	// we verify the account has enought balance to stack
	addr, _ := types.AccAddressFromBech32(msg.Creator)
	balance := ms.k.BankKeeper.GetBalance(ctx, addr, params.MinWorkerStaking.Denom)
	ms.k.logger.Debug("balance of %s [%s]: %s", msg.Creator, addr, balance)

	if balance.Amount.LT(params.MinWorkerStaking.Amount) {
		error := sdkerrors.ErrAppConfig.Wrapf(videoRendering.ErrWorkerIncorrectStake.Error(), "not enought balance to stack. Min value is %v", params.MinWorkerStaking.Amount)
		ms.k.logger.Error(error.Error())
		return &videoRendering.MsgAddWorkerResponse{Ok: false, Message: error.Error()}, error
	}

	// worker is not previously registered, so we move on
	reputation := videoRendering.Worker_Reputation{Points: 0, Staked: &msg.Stake, Validations: 0, Solutions: 0, Winnings: types.NewCoin(params.MinWorkerStaking.Denom, math.NewInt(0))}
	worker := videoRendering.Worker{Address: msg.Creator, Reputation: &reputation, Enabled: true, PublicIp: msg.PublicIp, IpfsId: msg.IpfsId}

	err = ms.k.Workers.Set(ctx, msg.Creator, worker)
	if err != nil {
		ms.k.logger.Error("Getting worker: %s", err.Error())
		return &videoRendering.MsgAddWorkerResponse{Ok: false, Message: err.Error()}, err
	}

	// // we stack the coins in the module
	err = ms.k.BankKeeper.SendCoinsFromAccountToModule(ctx, addr, videoRendering.ModuleName, types.NewCoins(msg.Stake))
	if err != nil {
		ms.k.logger.Error("Stacking worker's coins: %s", err.Error())
		return &videoRendering.MsgAddWorkerResponse{Ok: false, Message: err.Error()}, err
	}

	return &videoRendering.MsgAddWorkerResponse{Ok: true, Message: "Worker added correctly"}, nil
}

func (ms msgServer) SubscribeWorkerToTask(ctx context.Context, msg *videoRendering.MsgSubscribeWorkerToTask) (*videoRendering.MsgSubscribeWorkerToTaskResponse, error) {
	ms.k.logger.Info("SubscribeWorkerToTask - address: %s, taskId: %s", msg.Address, msg.TaskId)

	worker, err := ms.k.Workers.Get(ctx, msg.Address)
	if err != nil {
		ms.k.logger.Error("Getting Worker: %s", err.Error())
		return nil, err
	}

	if !worker.Enabled {
		ms.k.logger.Debug("Worker not enabled: %s", worker.String())
		return nil, sdkerrors.ErrAppConfig.Wrapf(videoRendering.ErrWorkerNotAvailable.Error(), "worker (%s) it nos enabled or doesn't exists", msg.Address)
	}
	task, err := ms.k.VideoRenderingTasks.Get(ctx, msg.TaskId)
	if err != nil {
		ms.k.logger.Error("Getting task: %s", err.Error())
		return nil, err
	}
	if task.Completed {
		ms.k.logger.Debug("Task is completed: %s", task.String())
		return nil, sdkerrors.ErrAppConfig.Wrapf(videoRendering.ErrWorkerTaskNotAvailable.Error(), "task (%s) is already completed. Can't subscribe worker", msg.TaskId)
	}

	// we get the params to get the MaxWorkersPerThread value
	params, _ := ms.k.Params.Get(ctx)
	for i, v := range task.Threads {
		if len(v.Workers) < int(params.MaxWorkersPerThread) && !v.Completed {
			v.Workers = append(v.Workers, msg.Address)

			ms.k.VideoRenderingTasks.Set(ctx, task.TaskId, task)
			worker.CurrentTaskId = task.TaskId
			worker.CurrentThreadIndex = int32(i)
			ms.k.Workers.Set(ctx, msg.Address, worker)

			err := ms.k.VideoRenderingTasks.Set(ctx, task.TaskId, task)
			if err != nil {
				ms.k.logger.Error("error trying to update thread %s to in progress", v.ThreadId)
			}
			return &videoRendering.MsgSubscribeWorkerToTaskResponse{ThreadId: v.ThreadId}, nil
		}
	}
	return nil, nil
}

func (ms msgServer) ProposeSolution(ctx context.Context, msg *videoRendering.MsgProposeSolution) (*videoRendering.MsgProposeSolutionResponse, error) {
	ms.k.logger.Info("ProposeSolution - creator: %s, taskId: %s, threadId: %s, ZKPs: %s", msg.Creator, msg.TaskId, msg.ThreadId, msg.Zkps)

	// creator of the solution must be a valid worker
	worker, err := ms.k.Workers.Get(ctx, msg.Creator)
	if err != nil {
		ms.k.logger.Error("Getting Worker: %s", err.Error())
		return nil, err
	}

	if !worker.Enabled {
		ms.k.logger.Error("workers %s is not enabled to propose a solution", msg.Creator)
		return nil, sdkerrors.ErrAppConfig.Wrapf(videoRendering.ErrInvalidSolution.Error(), "workers %s is not enabled to propose a solution", msg.Creator)
	}

	task, err := ms.k.VideoRenderingTasks.Get(ctx, msg.TaskId)
	if err != nil {
		ms.k.logger.Error("Getting Task: %s", err.Error())
		return nil, err
	}

	// task must exists and be in progress
	if task.Completed {
		ms.k.logger.Error("Task %s is not valid to accept solutions", msg.TaskId)
		return nil, sdkerrors.ErrAppConfig.Wrapf(videoRendering.ErrInvalidSolution.Error(), "Task %s is not valid to accept solutions", msg.TaskId)
	}

	for i, v := range task.Threads {
		// TODO threads might be better as map instead of slice
		if v.ThreadId == msg.ThreadId {
			if v.Solution != nil {
				ms.k.logger.Error("thread %s already has a solution", msg.ThreadId)
				return nil, sdkerrors.ErrAppConfig.Wrapf(videoRendering.ErrInvalidSolution.Error(), "thread %s already has a solution", msg.ThreadId)
			}
			// worker must be a valid registered worker in the thread with a solution
			if !slices.Contains(v.Workers, msg.Creator) {
				ms.k.logger.Error("Worker %s is not valid at thread %s", msg.Creator, msg.ThreadId)
				return nil, sdkerrors.ErrAppConfig.Wrapf(videoRendering.ErrInvalidSolution.Error(), "Worker %s is not valid at thread %s", msg.Creator, msg.ThreadId)
			}

			// solution len must be equal to the frames generated
			if len(msg.Zkps) != (int(v.EndFrame) - int(v.StartFrame) + 1) {
				ms.k.logger.Error("amount of files in solution is incorrect, %v ", len(msg.Zkps))
				return nil, sdkerrors.ErrAppConfig.Wrapf(videoRendering.ErrInvalidSolution.Error(), "amount of files in solution is incorrect, %v ", len(msg.Zkps))
			}

			// we have passed all validations, lets add the solution to the thread
			var frames []*videoRendering.VideoRenderingThread_Frame
			for _, val := range msg.Zkps {
				parts := strings.SplitN(val, "=", 2)
				frame := videoRendering.VideoRenderingThread_Frame{Filename: parts[0], Zkp: parts[1]}
				frames = append(frames, &frame)
			}

			task.Threads[i].Solution = &videoRendering.VideoRenderingThread_Solution{ProposedBy: msg.Creator, Frames: frames}
			err = ms.k.VideoRenderingTasks.Set(ctx, msg.TaskId, task)

			if err != nil {
				ms.k.logger.Error("unable to propose solution %s", err.Error())
				return nil, err
			}
			ms.k.logger.Info("Proposing solution %s", msg.Zkps)
		}
	}

	return &videoRendering.MsgProposeSolutionResponse{}, nil
}

// TODO Implement
func (ms msgServer) RevealSolution(ctx context.Context, msg *videoRendering.MsgRevealSolution) (*videoRendering.MsgRevealSolutionResponse, error) {
	ms.k.logger.Info("RevealSolution - creator: %s, taskId: %s, threadId: %s, CIDs: %s", msg.Creator, msg.TaskId, msg.ThreadId, msg.Cids)

	// Solution must be from a worker on the thread
	task, err := ms.k.VideoRenderingTasks.Get(ctx, msg.TaskId)

	if err != nil {
		ms.k.logger.Error("Getting task: %s", err.Error())
		return nil, err
	}

	worker, err := ms.k.Workers.Get(ctx, msg.Creator)

	if err != nil {
		ms.k.logger.Error("Getting Worker: %s", err.Error())
		return nil, err
	}

	if worker.CurrentTaskId != msg.TaskId {
		ms.k.logger.Error("worker is not working on task")
		return nil, sdkerrors.ErrAppConfig.Wrapf(videoRendering.ErrInvalidVerification.Error(), "worker is not working on task")
	}

	if task.Completed {
		ms.k.logger.Error("task is already completed. No more validations accepted")
		return nil, sdkerrors.ErrAppConfig.Wrapf(videoRendering.ErrInvalidVerification.Error(), "task is already completed. No more validations accepted")
	}

	thread := task.Threads[worker.CurrentThreadIndex]

	if thread.Solution.Accepted {
		ms.k.logger.Error("solution has already been accepted")
		return nil, sdkerrors.ErrAppConfig.Wrapf(videoRendering.ErrInvalidVerification.Error(), "solution has already been accepted.")
	}

	if thread.Solution.ProposedBy != msg.Creator {
		ms.k.logger.Error("creator is not the winner.")
		return nil, sdkerrors.ErrAppConfig.Wrapf(videoRendering.ErrInvalidVerification.Error(), "creator is not the winner.")
	}

	if thread.ThreadId != msg.ThreadId {
		ms.k.logger.Error("worker is not working on thread")
		return nil, sdkerrors.ErrAppConfig.Wrapf(videoRendering.ErrInvalidVerification.Error(), "worker is not working on thread")
	}

	// this shouldn't happen.
	if !slices.Contains(thread.Workers, msg.Creator) {
		ms.k.logger.Error("worker is not working on thread")
		return nil, sdkerrors.ErrAppConfig.Wrapf(videoRendering.ErrInvalidVerification.Error(), "worker is not working on thread")
	}

	// we make sure we have enought validations
	params, err := ms.k.Params.Get(ctx)
	if err != nil {
		ms.k.logger.Error("Getting Params: %s", err.Error())
		return nil, err
	}
	if len(thread.Validations) < int(params.MinValidators) {
		ms.k.logger.Error("not enought validators to perform verification")
		return nil, sdkerrors.ErrAppConfig.Wrapf(videoRendering.ErrInvalidVerification.Error(), "not enought validators to perform verification")
	}

	// cids amount must be equal to the amount of frames
	if len(msg.Cids) != len(thread.Solution.Frames) {
		ms.k.logger.Error("invalid amount of cids for the solution")
		return nil, sdkerrors.ErrAppConfig.Wrapf(videoRendering.ErrInvalidVerification.Error(), "invalid amount of cids for the solution")
	}

	for _, cids := range msg.Cids {
		parts := strings.SplitN(cids, "=", 2)
		idx := slices.IndexFunc(thread.Solution.Frames, func(f *videoRendering.VideoRenderingThread_Frame) bool { return f.Filename == parts[0] })
		if idx == -1 {
			return nil, sdkerrors.ErrAppConfig.Wrapf(videoRendering.ErrInvalidVerification.Error(), "frame %s not found in solution", parts[0])
		}
		thread.Solution.Frames[idx].Cid = parts[1]
	}

	task.Threads[worker.CurrentThreadIndex] = thread
	ms.k.VideoRenderingTasks.Set(ctx, msg.TaskId, task)
	return nil, nil
}

func (ms msgServer) SubmitValidation(ctx context.Context, msg *videoRendering.MsgSubmitValidation) (*videoRendering.MsgSubmitValidationResponse, error) {
	ms.k.logger.Info("SubmitValidation - creator: %s, taskId: %s, threadId: %s, ZKPs: %s", msg.Creator, msg.TaskId, msg.ThreadId, msg.Zkps)

	// validation must be from a worker on the thread
	task, err := ms.k.VideoRenderingTasks.Get(ctx, msg.TaskId)

	if err != nil {
		ms.k.logger.Error("Getting Task: %s", err.Error())
		return nil, err
	}

	worker, err := ms.k.Workers.Get(ctx, msg.Creator)

	if err != nil {
		ms.k.logger.Error("Getting Worker: %s", err.Error())
		return nil, err
	}

	if !worker.Enabled {
		ms.k.logger.Error("worker is not allowed to validate solutions")
		return nil, sdkerrors.ErrAppConfig.Wrapf(videoRendering.ErrInvalidVerification.Error(), "worker is not allowed to validate solutions")
	}

	if worker.CurrentTaskId != msg.TaskId {
		ms.k.logger.Error("worker is not working on task")
		return nil, sdkerrors.ErrAppConfig.Wrapf(videoRendering.ErrInvalidVerification.Error(), "worker is not working on task")
	}

	if task.Completed {
		ms.k.logger.Error("task is already completed. No more validations accepted")
		return nil, sdkerrors.ErrAppConfig.Wrapf(videoRendering.ErrInvalidVerification.Error(), "task is already completed. No more validations accepted")
	}

	thread := task.Threads[worker.CurrentThreadIndex]
	if thread.ThreadId != msg.ThreadId {
		ms.k.logger.Error("worker is not working on thread")
		return nil, sdkerrors.ErrAppConfig.Wrapf(videoRendering.ErrInvalidVerification.Error(), "worker is not working on thread")
	}

	// this shouldn't happen.
	if !slices.Contains(thread.Workers, msg.Creator) {
		ms.k.logger.Error("worker is not working on thread")
		return nil, sdkerrors.ErrAppConfig.Wrapf(videoRendering.ErrInvalidVerification.Error(), "worker is not working on thread")
	}

	var frames []*videoRendering.VideoRenderingThread_Frame
	for _, zkps := range msg.Zkps {
		parts := strings.SplitN(zkps, "=", 2)
		frame := videoRendering.VideoRenderingThread_Frame{Filename: parts[0], Zkp: parts[1]}
		frames = append(frames, &frame)
	}
	validation := videoRendering.VideoRenderingThread_Validation{Validator: msg.Creator, IsReverse: thread.IsReverse(worker.Address), Frames: frames}
	task.Threads[worker.CurrentThreadIndex].Validations = append(thread.Validations, &validation)
	ms.k.VideoRenderingTasks.Set(ctx, msg.TaskId, task)

	return &videoRendering.MsgSubmitValidationResponse{}, nil
}

func (ms msgServer) SubmitSolution(ctx context.Context, msg *videoRendering.MsgSubmitSolution) (*videoRendering.MsgSubmitSolutionResponse, error) {
	ms.k.logger.Info("SubmitSolution - creator: %s, taskId: %s, threadId: %s, Dir: %s", msg.Creator, msg.TaskId, msg.ThreadId, msg.Dir)

	task, err := ms.k.VideoRenderingTasks.Get(ctx, msg.TaskId)
	if err != nil {
		ms.k.logger.Error("Getting Task: %s", err.Error())
		return nil, sdkerrors.ErrAppConfig.Wrapf(videoRendering.ErrInvalidSolution.Error(), "provided task doesn't exists")
	}
	for i, thread := range task.Threads {
		if thread.ThreadId == msg.ThreadId {

			if thread.Solution.ProposedBy != msg.Creator {
				error := sdkerrors.ErrAppConfig.Wrapf(videoRendering.ErrInvalidSolution.Error(), "only the provider of the solution can upload it")
				ms.k.logger.Error(error.Error())
				return nil, error
			}

			if !thread.Completed {
				error := sdkerrors.ErrAppConfig.Wrapf(videoRendering.ErrInvalidSolution.Error(), "thread is not yet completed")
				ms.k.logger.Error(error.Error())
				return nil, error
			}

			// TODO Implement
			// we make sure ipfs is running
			// ipfs.EnsureIPFSRunning()

			// we verify the solution
			// err := thread.VerifySubmittedSolution(msg.Cid)
			// if err != nil {
			// 	return nil, sdkerrors.ErrAppConfig.Wrapf(videoRendering.ErrInvalidSolution.Error(), "submited solution is incorrect")
			// }

			// solution is verified so we pay the winner
			addr, _ := types.AccAddressFromBech32(msg.Creator)
			payment := task.GetWinnerReward()
			ms.k.BankKeeper.SendCoinsFromModuleToAccount(ctx, videoRendering.ModuleName, addr, types.NewCoins(payment))
			task.Threads[i].Solution.Dir = msg.Dir
			ms.k.VideoRenderingTasks.Set(ctx, msg.TaskId, task)

			// we increase the reputation of the winner
			worker, _ := ms.k.Workers.Get(ctx, msg.Creator)
			worker.DeclareWinner(payment)
			ms.k.Workers.Set(ctx, msg.Creator, worker)
			break
		}
	}
	return &videoRendering.MsgSubmitSolutionResponse{}, nil
}
