package videoRendering

import "cosmossdk.io/collections"

const ModuleName = "videoRendering"

var (
	ParamsKey             = collections.NewPrefix("Params")
	VideoRenderingTaskKey = collections.NewPrefix("VideoRenderingFiles/value/")
)
