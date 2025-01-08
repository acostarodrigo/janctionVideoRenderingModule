package keeper

import (
	"context"
	"log"
	"strconv"

	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

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
	// TODO had validations about the parameters
	taskInfo, err := ms.k.VideoRenderingTaskInfo.Get(ctx)
	if err != nil {
		return nil, err
	}

	var nextId = taskInfo.NextId
	// we get the taskId in string
	taskId := strconv.FormatUint(nextId, 10)

	// and increase the task id counter for next task
	nextId++
	ms.k.VideoRenderingTaskInfo.Set(ctx, videoRendering.VideoRenderingTaskInfo{NextId: nextId})

	videoTask := videoRendering.VideoRenderingTask{TaskId: taskId, Requester: msg.Creator, Cid: msg.Cid, StartFrame: msg.StartFrame, EndFrame: msg.EndFrame, InProgress: true, ThreadAmount: msg.Threads, Reward: msg.Reward}
	threads := videoTask.GenerateThreads()
	videoTask.Threads = threads

	if err := ms.k.VideoRenderingTasks.Set(ctx, taskId, videoTask); err != nil {
		return nil, err
	}
	return &videoRendering.MsgCreateVideoRenderingTaskResponse{TaskId: taskId}, nil
}

func (ms msgServer) AddWorker(ctx context.Context, msg *videoRendering.MsgAddWorker) (*videoRendering.MsgAddWorkerResponse, error) {
	found, err := ms.k.Workers.Has(ctx, msg.Address)
	if err != nil {
		return nil, err
	}

	if found {
		log.Printf("Worker %v already exists.", msg.Address)
		return nil, sdkerrors.ErrAppConfig.Wrapf(videoRendering.ErrWorkerAlreadyRegistered.Error(), "worker (%s) is already registered", msg.Address)
	}

	// worker is not previously registered, so we move on
	// TODO I'm facking a stacked value of 100 for future use
	reputation := videoRendering.Worker_Reputation{Points: 0, Stacked: 100}
	worker := videoRendering.Worker{Address: msg.Address, Reputation: &reputation, Status: videoRendering.Worker_WORKER_STATUS_IDLE, Enabled: true}

	ms.k.Workers.Set(ctx, msg.Address, worker)
	return &videoRendering.MsgAddWorkerResponse{}, nil
}

func (ms msgServer) SubscribeWorkerToTask(ctx context.Context, msg *videoRendering.MsgSubscribeWorkerToTask) (*videoRendering.MsgSubscribeWorkerToTaskResponse, error) {
	worker, err := ms.k.Workers.Get(ctx, msg.Address)
	if err != nil {
		return nil, err
	}

	if !worker.Enabled {
		return nil, sdkerrors.ErrAppConfig.Wrapf(videoRendering.ErrWorkerNotAvailable.Error(), "worker (%s) it nos enabled or doesn't exists", msg.Address)
	}
	task, err := ms.k.VideoRenderingTasks.Get(ctx, msg.TaskId)
	if err != nil {
		return nil, err
	}
	if !task.InProgress {
		return nil, sdkerrors.ErrAppConfig.Wrapf(videoRendering.ErrWorkerTaskNotAvailable.Error(), "task (%s) is already completed. Can't subscribe worker", msg.TaskId)
	}

	for _, v := range task.Threads {
		if len(v.Workers) < 10 {
			v.Workers = append(v.Workers, msg.Address)
			ms.k.VideoRenderingTasks.Set(ctx, task.TaskId, task)
			return &videoRendering.MsgSubscribeWorkerToTaskResponse{ThreadId: v.ThreadId}, nil
		}
	}
	return nil, nil
}
