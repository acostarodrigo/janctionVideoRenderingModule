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
				{
					RpcMethod: "GetPendingVideoRenderingTasks",
					Use:       "get-pending-video-rendering-tasks",
					Short:     "Gets the pending video rendering tasks",
				},
			},
		},
		Tx: &autocliv1.ServiceCommandDescriptor{
			Service: videoRenderingv1.Msg_ServiceDesc.ServiceName,
			RpcCommandOptions: []*autocliv1.RpcCommandOptions{
				{
					RpcMethod: "CreateVideoRenderingTask",
					Use:       "create-video-rendering-task [cid] [startFrame] [endFrame] [threads] [reward]",
					Short:     "Creates a new video Rendering task",
					Long:      "", // TODO Add long
					Example:   "", // TODO add exampe
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "cid"},
						{ProtoField: "startFrame"},
						{ProtoField: "endFrame"},
						{ProtoField: "threads"},
						{ProtoField: "reward"},
					},
				},
				{
					RpcMethod: "AddWorker",
					Use:       "add-worker [public_ip] [ipfs_id] [stake]--from [workerAddress]",
					Short:     "Registers a new worker that will perform video rendering tasks",
					Long:      "", // TODO Add long
					Example:   "", // TODO add exampe
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "public_ip"},
						{ProtoField: "ipfs_id"},
						{ProtoField: "stake"},
					},
				},
				{
					RpcMethod: "SubscribeWorkerToTask",
					Use:       "subscribe-worker-to-task [address] [taskId] --from [workerAddress]",
					Short:     "Subscribes an existing enabled worker to perform work in the specified task",
					Long:      "", // TODO Add long
					Example:   "", // TODO add exampe
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "address"},
						{ProtoField: "taskId"},
					},
				},
				{
					RpcMethod: "ProposeSolution",
					Use:       "propose-solution [taskId] [threadId] [solution] --from [workerAddress]",
					Short:     "Proposes a solution to a thread.",
					Long:      "", // TODO Add long
					Example:   "", // TODO add exampe
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "taskId"},
						{ProtoField: "threadId"},
						{ProtoField: "solution", Varargs: true},
					},
				},
				{
					RpcMethod: "SubmitSolution",
					Use:       "submit-solution [taskId] [threadId] [cid] --from [workerAddress]",
					Short:     "Submits the cid of the directory with all the uploaded frames.",
					Long:      "", // TODO Add long
					Example:   "", // TODO add exampe
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "taskId"},
						{ProtoField: "threadId"},
						{ProtoField: "cid", Varargs: false},
					},
				},
				{
					RpcMethod: "SubmitValidation",
					Use:       "submit-validation [taskId] [threadId] [filesAmount] [valid] --from [workerAddress]",
					Short:     "Submit a validation to a proposed solution",
					Long:      "", // TODO Add long
					Example:   "", // TODO add exampe
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "taskId"},
						{ProtoField: "threadId"},
						{ProtoField: "filesAmount"},
						{ProtoField: "valid"},
					},
				},
			},
		},
	}
}
