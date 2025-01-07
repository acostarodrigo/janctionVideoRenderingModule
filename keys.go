package videoRendering

import "cosmossdk.io/collections"

const ModuleName = "videoRendering"

var (
	ParamsKey                     = collections.NewPrefix("Params")
	VideoRenderingTaskKey         = collections.NewPrefix("videoRenderingTaskList/value/")
	WorkerKey                     = collections.NewPrefix("Worker")
	TaskInfoKey                   = collections.NewPrefix(0)
	PendingVideoRenderingTasksKey = collections.NewPrefix(1)
)
