package keeper

import (
	"context"
	"errors"

	"cosmossdk.io/collections"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/janction/videoRendering"
)

var _ videoRendering.QueryServer = queryServer{}

// NewQueryServerImpl returns an implementation of the module QueryServer.
func NewQueryServerImpl(k Keeper) videoRendering.QueryServer {
	return queryServer{k}
}

type queryServer struct {
	k Keeper
}

// GetGame defines the handler for the Query/GetGame RPC method.
func (qs queryServer) GetVideoRenderingTask(ctx context.Context, req *videoRendering.QueryGetVideoRenderingTaskRequest) (*videoRendering.QueryGetVideoRenderingTaskResponse, error) {
	videoRenderingTask, err := qs.k.VideoRenderingTasks.Get(ctx, req.Index)
	if err == nil {
		return &videoRendering.QueryGetVideoRenderingTaskResponse{VideoRenderingTask: &videoRenderingTask}, nil
	}
	if errors.Is(err, collections.ErrNotFound) {
		return &videoRendering.QueryGetVideoRenderingTaskResponse{VideoRenderingTask: nil}, nil
	}

	return nil, status.Error(codes.Internal, err.Error())
}
