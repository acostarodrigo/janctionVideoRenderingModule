package videoRendering

import "strconv"

func (VideoRenderingTask) Validate() error {
	return nil
}

func GetEmptyVideoRenderingTaskList() []IndexedVideoRenderingTask {
	return []IndexedVideoRenderingTask{}
}

func (t *VideoRenderingTask) GenerateThreads() (res []*VideoRenderingThread) {
	// Split frames among the threads
	frameRanges := splitFrames(int(t.StartFrame), int(t.EndFrame), int(t.ThreadAmount))

	// Print the result

	for i, r := range frameRanges {
		thread := VideoRenderingThread{ThreadId: t.TaskId + strconv.FormatInt(int64(i), 10), StartFrame: uint32(r.StartFrame), EndFrame: uint32(r.EndFrame)}
		res = append(res, &thread)
	}

	return res
}

// SplitFrames divides the total frames into chunks based on the number of threads
// FrameRange represents the range of frames assigned to a thread
type frameRange struct {
	StartFrame int
	EndFrame   int
}

func splitFrames(startFrame, endFrame, threads int) []frameRange {
	totalFrames := endFrame - startFrame + 1 // Total number of frames
	framesPerThread := totalFrames / threads // Base number of frames per thread
	remainder := totalFrames % threads       // Remaining frames to distribute

	result := make([]frameRange, threads)

	currentStart := startFrame
	for i := 0; i < threads; i++ {
		extra := 0
		if remainder > 0 {
			extra = 1
			remainder--
		}

		end := currentStart + framesPerThread + extra - 1
		result[i] = frameRange{StartFrame: currentStart, EndFrame: end}
		currentStart = end + 1
	}

	return result
}
