package videoRendering

func (VideoRenderingTask) Validate() error {
	return nil
}

func GetEmptyVideoRenderingTaskList() []IndexedVideoRenderingTask {
	return []IndexedVideoRenderingTask{}
}
