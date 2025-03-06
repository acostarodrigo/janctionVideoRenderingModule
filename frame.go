package videoRendering

// finds an specific Frame from the Frames slice
func GetFrame(frames []*VideoRenderingThread_Frame, filename string) *VideoRenderingThread_Frame {
	for _, frame := range frames {
		if frame.Filename == filename {
			return frame
		}
	}
	return nil
}
