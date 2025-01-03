package module

import (
	autocliv1 "cosmossdk.io/api/cosmos/autocli/v1"
	videoRenderingv1 "github.com/janction/videoRendering/api/v1"
)

// AutoCLIOptions implements the autocli.HasAutoCLIConfig interface.
func (am AppModule) AutoCLIOptions() *autocliv1.ModuleOptions {
	return &autocliv1.ModuleOptions{
		Query: &autocliv1.ServiceCommandDescriptor{
			Service: videoRenderingv1.Query_ServiceDesc.ServiceName,
			RpcCommandOptions: []*autocliv1.RpcCommandOptions{
				{
					RpcMethod: "GetVideoRenderingTask",
					Use:       "get-video-rendering-task index",
					Short:     "Get the current value of the Video Rendering task at index",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "index"},
					},
				},
			},
		},
		Tx: &autocliv1.ServiceCommandDescriptor{
			Service: videoRenderingv1.Msg_ServiceDesc.ServiceName,
			RpcCommandOptions: []*autocliv1.RpcCommandOptions{
				{
					RpcMethod: "CreateVideoRenderingTask",
					Use:       "create-video-rendering-task [cid] [startFrame] [endFrame] [threads]",
					Short:     "Creates a new video Rendering task",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "cid"},
						{ProtoField: "startFrame"},
						{ProtoField: "endFrame"},
						{ProtoField: "threads"},
					},
				},
			},
		},
	}
}
