package keeper

import (
	"context"

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
	if err := ms.k.VideoRenderingTasks.Set(ctx, "1", videoRendering.VideoRenderingTask{Requester: msg.Creator, Cid: msg.Cid, StartFrame: msg.StartFrame, EndFrame: msg.EndFrame, InProgress: false, ThreadAmount: msg.Threads}); err != nil {
		return nil, err
	}
	return &videoRendering.MsgCreateVideoRenderingTaskResponse{TaskId: "1"}, nil
}
