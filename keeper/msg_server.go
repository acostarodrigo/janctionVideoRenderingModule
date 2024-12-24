package keeper

import (
	"context"
	"log"
	"strconv"

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
	log.Println("Started creation")
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

	if err := ms.k.VideoRenderingTasks.Set(ctx, taskId, videoRendering.VideoRenderingTask{Requester: msg.Creator, Cid: msg.Cid, StartFrame: msg.StartFrame, EndFrame: msg.EndFrame, InProgress: false, ThreadAmount: msg.Threads}); err != nil {
		return nil, err
	}
	return &videoRendering.MsgCreateVideoRenderingTaskResponse{TaskId: taskId}, nil
}
