package vm

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"reflect"
	"runtime"
	"testing"

	"bou.ke/monkey"
	"github.com/janction/videoRendering/db"
	"github.com/stretchr/testify/require"
)

// --- Test for IsContainerRunning ---
func TestIsContainerRunningKo(t *testing.T) {
	// 1. Setup
	ctx := context.Background()

	// 2. Monkey patching
	patch1 := monkey.Patch(exec.CommandContext, func(ctx context.Context, name string, arg ...string) *exec.Cmd {
		return nil
	})
	defer patch1.Unpatch()

	patch2 := monkey.PatchInstanceMethod(reflect.TypeOf(&exec.Cmd{}), "Output", func(cmd *exec.Cmd) ([]byte, error) {
		return nil, fmt.Errorf("Output error")
	})
	defer patch2.Unpatch()

	// 3. Execute method under test
	b := IsContainerRunning(ctx, "1234")

	// 4. Verification
	require.False(t, b)
}

func TestIsContainerRunningOk(t *testing.T) {
	// 1. Setup
	ctx := context.Background()

	// 2. Monkey patching
	patch1 := monkey.Patch(exec.CommandContext, func(ctx context.Context, name string, arg ...string) *exec.Cmd {
		return nil
	})
	defer patch1.Unpatch()

	patch2 := monkey.PatchInstanceMethod(reflect.TypeOf(&exec.Cmd{}), "Output", func(cmd *exec.Cmd) ([]byte, error) {
		return []byte("myBlender1234"), nil
	})
	defer patch2.Unpatch()

	// 3. Execute method under test
	b := IsContainerRunning(ctx, "1234")

	// 4. Verification
	require.True(t, b)
}

// --- Test for RenderVideo ---
func TestRenderVideoNoReverse(t *testing.T) {
	// 1. Setup
	mockDB := db.NewDB(nil)
	ctx := context.Background()
	cid := "bafybeigdyrztxx3b7d5qzq2ujay5g4qxxuj5f6x3h6lgv7d4ttrddn3cxa"
	id := "thread123"
	path := "/tmp/rendering/thread123/frame_42"
	start := int64(1)
	end := int64(10)
	reverse := false
	function_calls := make([]int64, 0, 10) // Empty slice with a capacity of 10

	// 2. Monkey patch the renderVideoFrame function to not actually call it, just save the call to a variable
	patch1 := monkey.Patch(renderVideoFrame, func(ctx context.Context, cid string, frameNumber int64, id string, path string, db *db.DB) error {
		function_calls = append(function_calls, frameNumber)
		return nil
	})
	defer patch1.Unpatch()

	// 3. Execute method under test
	RenderVideo(ctx, cid, start, end, id, path, reverse, &mockDB)

	// 5. Verification
	expected := []int64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	require.Equal(t, function_calls, expected)
}

func TestRenderVideoReverse(t *testing.T) {
	// 1. Setup
	mockDB := db.NewDB(nil)
	ctx := context.Background()
	cid := "bafybeigdyrztxx3b7d5qzq2ujay5g4qxxuj5f6x3h6lgv7d4ttrddn3cxa"
	id := "thread123"
	path := "/tmp/rendering/thread123/frame_42"
	start := int64(1)
	end := int64(10)
	reverse := true
	function_calls := make([]int64, 0, 10) // Empty slice with a capacity of 10

	// 2. Monkey patch the renderVideoFrame function to not actually call it, just count the number of times it is called
	patch1 := monkey.Patch(renderVideoFrame, func(ctx context.Context, cid string, frameNumber int64, id string, path string, db *db.DB) error {
		function_calls = append(function_calls, frameNumber)
		return nil
	})
	defer patch1.Unpatch()

	// 3. Execute method under test
	RenderVideo(ctx, cid, start, end, id, path, reverse, &mockDB)

	// 4. Verification
	expected := []int64{10, 9, 8, 7, 6, 5, 4, 3, 2, 1}
	require.Equal(t, function_calls, expected)
}

// --- Test for renderVideoFrame ---
func TestRenderVideoFrame_ContainerVerificationError(t *testing.T) {
	// 1. Setup
	mockDB := db.NewDB(nil)
	ctx := context.Background()
	cid := "bafybeigdyrztxx3b7d5qzq2ujay5g4qxxuj5f6x3h6lgv7d4ttrddn3cxa"
	frameNumber := int64(42)
	id := "thread123"
	path := "/tmp/rendering/thread123/frame_42"

	// 2. Mock DB methods
	patch3 := monkey.Patch((*db.DB).AddLogEntry, func(_ *db.DB, threadId, log string, timestamp, severity int64) error {
		return nil
	})
	defer patch3.Unpatch()

	// 3. Monkey patching
	patch1 := monkey.Patch(exec.CommandContext, func(ctx context.Context, name string, arg ...string) *exec.Cmd {
		return &exec.Cmd{}
	})
	defer patch1.Unpatch()

	patch2 := monkey.PatchInstanceMethod(reflect.TypeOf(&exec.Cmd{}), "Output", func(cmd *exec.Cmd) ([]byte, error) {
		return nil, fmt.Errorf("Error verifying if container already exists")
	})
	defer patch2.Unpatch()

	// 4. Execute method under test
	err := renderVideoFrame(ctx, cid, frameNumber, id, path, &mockDB)

	// 5. Verification
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to check container existence: Error verifying if container already exists")
}

func TestRenderVideoFrame_ContainerAlreadyExist(t *testing.T) {
	// 1. Setup
	mockDB := db.NewDB(nil)
	ctx := context.Background()
	cid := "bafybeigdyrztxx3b7d5qzq2ujay5g4qxxuj5f6x3h6lgv7d4ttrddn3cxa"
	frameNumber := int64(42)
	id := "thread123"
	path := "/tmp/rendering/thread123/frame_42"

	// 2. Mock DB methods
	patch3 := monkey.Patch((*db.DB).AddLogEntry, func(_ *db.DB, threadId, log string, timestamp, severity int64) error {
		return nil
	})
	defer patch3.Unpatch()

	// 3. Monkey patching
	patch1 := monkey.Patch(exec.CommandContext, func(ctx context.Context, name string, arg ...string) *exec.Cmd {
		return &exec.Cmd{}
	})
	defer patch1.Unpatch()

	patch2 := monkey.PatchInstanceMethod(reflect.TypeOf(&exec.Cmd{}), "Output", func(cmd *exec.Cmd) ([]byte, error) {
		return []byte("myBlender1234"), nil
	})
	defer patch2.Unpatch()

	// 4. Execute method under test
	err := renderVideoFrame(ctx, cid, frameNumber, id, path, &mockDB)

	// 5. Verification
	require.NoError(t, err)
}

func TestRenderVideoFrame_CreatingContainerKo(t *testing.T) {
	// 1. Setup
	mockDB := db.NewDB(nil)
	ctx := context.Background()
	cid := "bafybeigdyrztxx3b7d5qzq2ujay5g4qxxuj5f6x3h6lgv7d4ttrddn3cxa"
	frameNumber := int64(42)
	id := "thread123"
	path := "/tmp/rendering/thread123/frame_42"

	// 2. Mock DB methods
	patch4 := monkey.Patch((*db.DB).AddLogEntry, func(_ *db.DB, threadId, log string, timestamp, severity int64) error {
		return nil
	})
	defer patch4.Unpatch()

	// 3. Monkey patch CommandContext to return an *exec.Cmd with visible arguments
	patch1 := monkey.Patch(exec.CommandContext, func(ctx context.Context, name string, arg ...string) *exec.Cmd {
		return &exec.Cmd{
			Path: name,
			Args: append([]string{name}, arg...),
		}
	})
	defer patch1.Unpatch()

	// 4. Patch Output to simulate that container doesn't exist
	patch2 := monkey.PatchInstanceMethod(reflect.TypeOf(&exec.Cmd{}), "Output", func(cmd *exec.Cmd) ([]byte, error) {
		return []byte(""), nil
	})
	defer patch2.Unpatch()

	// 5. Patch Run to simulate failure when creating the container
	patch3 := monkey.PatchInstanceMethod(reflect.TypeOf(&exec.Cmd{}), "Run", func(cmd *exec.Cmd) error {
		return fmt.Errorf("Error creating container")
	})
	defer patch3.Unpatch()

	// 6. Execute the function under test
	err := renderVideoFrame(ctx, cid, frameNumber, id, path, &mockDB)

	// 7. Assert the error
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to create and start container: Error creating container")
}

func TestRenderVideoFrame_CreatingContainerOk_WaitingContainerKo(t *testing.T) {
	// 1. Setup
	mockDB := db.NewDB(nil)
	ctx := context.Background()
	cid := "bafybeigdyrztxx3b7d5qzq2ujay5g4qxxuj5f6x3h6lgv7d4ttrddn3cxa"
	frameNumber := int64(42)
	id := "thread123"
	path := "/tmp/rendering/thread123/frame_42"

	// 2. Mock DB methods
	patch4 := monkey.Patch((*db.DB).AddLogEntry, func(_ *db.DB, threadId, log string, timestamp, severity int64) error {
		return nil
	})
	defer patch4.Unpatch()

	// 3. Monkey patch CommandContext to return an *exec.Cmd with visible arguments
	patch1 := monkey.Patch(exec.CommandContext, func(ctx context.Context, name string, arg ...string) *exec.Cmd {
		return &exec.Cmd{
			Path: name,
			Args: append([]string{name}, arg...),
		}
	})
	defer patch1.Unpatch()

	// 4. Patch Output to simulate that container doesn't exist
	patch2 := monkey.PatchInstanceMethod(reflect.TypeOf(&exec.Cmd{}), "Output", func(cmd *exec.Cmd) ([]byte, error) {
		return []byte(""), nil
	})
	defer patch2.Unpatch()

	// 5. Patch Run to simulate success when creating the container and failure when waiting for it
	patch3 := monkey.PatchInstanceMethod(reflect.TypeOf(&exec.Cmd{}), "Run", func(cmd *exec.Cmd) error {
		if len(cmd.Args) > 1 {
			switch cmd.Args[1] {
			case "run":
				return nil
			case "wait":
				return fmt.Errorf("failed here")
			}
		}
		return fmt.Errorf("unexpected command")
	})
	defer patch3.Unpatch()

	// 6. Execute the function under test
	err := renderVideoFrame(ctx, cid, frameNumber, id, path, &mockDB)

	// 7. Assert the error
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to wait for container: failed here")
}

func TestRenderVideoFrame_CreatingContainerOk_WaitingContainerOk_RetrieveLogsKo(t *testing.T) {
	// 1. Setup
	mockDB := db.NewDB(nil)
	ctx := context.Background()
	cid := "bafybeigdyrztxx3b7d5qzq2ujay5g4qxxuj5f6x3h6lgv7d4ttrddn3cxa"
	frameNumber := int64(42)
	id := "thread123"
	path := "/tmp/rendering/thread123/frame_42"

	// 2. Mock DB methods
	patch4 := monkey.Patch((*db.DB).AddLogEntry, func(_ *db.DB, threadId, log string, timestamp, severity int64) error {
		return nil
	})
	defer patch4.Unpatch()

	// 3. Monkey patch CommandContext to return an *exec.Cmd with visible arguments
	patch1 := monkey.Patch(exec.CommandContext, func(ctx context.Context, name string, arg ...string) *exec.Cmd {
		return &exec.Cmd{
			Path: name,
			Args: append([]string{name}, arg...),
		}
	})
	defer patch1.Unpatch()

	// 4. Patch Output to simulate that container doesn't exist and retrieving logs fail
	patch2 := monkey.PatchInstanceMethod(reflect.TypeOf(&exec.Cmd{}), "Output", func(cmd *exec.Cmd) ([]byte, error) {
		switch cmd.Args[1] {
		case "ps":
			return []byte(""), nil
		case "logs":
			return nil, fmt.Errorf("failed here")
		}
		return nil, fmt.Errorf("unexpected command")
	})
	defer patch2.Unpatch()

	// 5. Patch Run to simulate success when creating and waiting for the container and failure when retrieving
	patch3 := monkey.PatchInstanceMethod(reflect.TypeOf(&exec.Cmd{}), "Run", func(cmd *exec.Cmd) error {
		if len(cmd.Args) > 1 {
			switch cmd.Args[1] {
			case "run":
				return nil
			case "wait":
				return nil
			}
		}
		return fmt.Errorf("unexpected command")
	})
	defer patch3.Unpatch()

	// 6. Execute the function under test
	err := renderVideoFrame(ctx, cid, frameNumber, id, path, mockDB)

	// 7. Assert the error
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to retrieve container logs: failed here")
}

func TestRenderVideoFrame_CreatingContainerOk_WaitingContainerOk_RetrieveLogsOk_VerifyFileOk(t *testing.T) {
	// 1. Setup
	mockDB := db.NewDB(nil)
	ctx := context.Background()
	cid := "bafybeigdyrztxx3b7d5qzq2ujay5g4qxxuj5f6x3h6lgv7d4ttrddn3cxa"
	frameNumber := int64(42)
	id := "thread123"
	path := "/tmp/rendering/thread123/frame_42"

	// 2. Mock DB methods
	patch5 := monkey.Patch((*db.DB).AddLogEntry, func(_ *db.DB, threadId, log string, timestamp, severity int64) error {
		return nil
	})
	defer patch5.Unpatch()

	// 3. Monkey patch CommandContext to return an *exec.Cmd with visible arguments
	patch1 := monkey.Patch(exec.CommandContext, func(ctx context.Context, name string, arg ...string) *exec.Cmd {
		return &exec.Cmd{
			Path: name,
			Args: append([]string{name}, arg...),
		}
	})
	defer patch1.Unpatch()

	// 4. Patch Output to simulate that container doesn't exist and retrieving logs succeeds
	patch2 := monkey.PatchInstanceMethod(reflect.TypeOf(&exec.Cmd{}), "Output", func(cmd *exec.Cmd) ([]byte, error) {
		switch cmd.Args[1] {
		case "ps":
			return []byte(""), nil
		case "logs":
			return []byte(""), nil
		}
		return nil, fmt.Errorf("unexpected command")
	})
	defer patch2.Unpatch()

	// 5. Patch Run to simulate success when creating and waiting for the container
	patch3 := monkey.PatchInstanceMethod(reflect.TypeOf(&exec.Cmd{}), "Run", func(cmd *exec.Cmd) error {
		if len(cmd.Args) > 1 {
			switch cmd.Args[1] {
			case "run":
				return nil
			case "wait":
				return nil
			}
		}
		return fmt.Errorf("unexpected command")
	})
	defer patch3.Unpatch()

	// 6. Patch Os.Stat to simulate success when verifying that file exists
	patch4 := monkey.Patch(os.Stat, func(name string) (os.FileInfo, error) {
		return nil, nil
	})
	defer patch4.Unpatch()

	// 7. Execute the function under test
	err := renderVideoFrame(ctx, cid, frameNumber, id, path, &mockDB)

	// 8. Assert no error
	require.NoError(t, err)
}

// --- Test for RemoveContainer ---
func TestRemoveContainerKo(t *testing.T) {
	// 1. Setup
	ctx := context.Background()
	name := "container123"

	// 2. Monkey patch CommandContext to return an *exec.Cmd with visible arguments
	patch1 := monkey.Patch(exec.CommandContext, func(ctx context.Context, name string, arg ...string) *exec.Cmd {
		return &exec.Cmd{
			Path: name,
			Args: append([]string{name}, arg...),
		}
	})
	defer patch1.Unpatch()

	// 3. Patch Run to simulate that container was removed successfully
	patch2 := monkey.PatchInstanceMethod(reflect.TypeOf(&exec.Cmd{}), "Run", func(cmd *exec.Cmd) error {
		return fmt.Errorf("failed removing container")
	})
	defer patch2.Unpatch()

	// 4. Execute the function under test
	err := RemoveContainer(ctx, name)

	// 5. Assert the error
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed removing container")
}

func TestRemoveContainerOk(t *testing.T) {
	// 1. Setup
	ctx := context.Background()
	name := "container123"

	// 2. Monkey patch CommandContext to return an *exec.Cmd with visible arguments
	patch1 := monkey.Patch(exec.CommandContext, func(ctx context.Context, name string, arg ...string) *exec.Cmd {
		return &exec.Cmd{
			Path: name,
			Args: append([]string{name}, arg...),
		}
	})
	defer patch1.Unpatch()

	// 3. Patch Run to simulate that container was removed successfully
	patch2 := monkey.PatchInstanceMethod(reflect.TypeOf(&exec.Cmd{}), "Run", func(cmd *exec.Cmd) error {
		return nil
	})
	defer patch2.Unpatch()

	// 4. Execute the function under test
	err := RemoveContainer(ctx, name)

	// 5. Assert no error
	require.NoError(t, err)
}

// --- Test for CountFilesInDirectory ---
func TestCountFilesInDirectoryKo(t *testing.T) {
	// 1. Setup
	path := "path123"

	// 2. Patch Os.ReadDir to simulate failure when reading the directory
	patch4 := monkey.Patch(os.ReadDir, func(name string) ([]os.DirEntry, error) {
		return nil, fmt.Errorf("failed to read directory")
	})
	defer patch4.Unpatch()

	// 3. Execute the function under test
	count := CountFilesInDirectory(path)

	// 4. Assert that the count is 0
	require.Equal(t, count, 0)
}

func TestCountFilesInDirectoryOk(t *testing.T) {
	// 1. Setup
	path := "path123"

	// 2. Patch Os.ReadDir to simulate success when reading the directory
	patch4 := monkey.Patch(os.ReadDir, func(name string) ([]os.DirEntry, error) {
		return nil, nil
	})
	defer patch4.Unpatch()

	// 3. Execute the function under test
	count := CountFilesInDirectory(path)

	// 4. Assert that the count is 0
	require.Equal(t, count, 0)
}

// --- Test for FormatFrameFilename ---
func TestFormatFrameFilename(t *testing.T) {
	// 1. Setup
	frame := 42

	// 2. Execute the function under test
	filename := FormatFrameFilename(frame)

	// 3. Assert
	require.Equal(t, filename, "frame_000042.png")
}

// --- Test for IsARM64 ---
func TestIsARM64(t *testing.T) {
	// 1. Execute the function under test
	is_arm := isARM64()

	// 2. Assert
	require.Equal(t, is_arm, runtime.GOARCH == "arm64")
}

// --- Test for IsContainerExited ---
func TestIsContainerExitedKo(t *testing.T) {
	// 1. Setup
	id := "thread123"

	// 2. Monkey patch CommandContext to return an *exec.Cmd with visible arguments
	patch1 := monkey.Patch(exec.Command, func(name string, arg ...string) *exec.Cmd {
		return &exec.Cmd{
			Path: name,
			Args: append([]string{name}, arg...),
		}
	})
	defer patch1.Unpatch()

	// 3. Patch Run in order to simulate docker ps failure
	patch2 := monkey.PatchInstanceMethod(reflect.TypeOf(&exec.Cmd{}), "Run", func(cmd *exec.Cmd) error {
		switch cmd.Args[1] {
		case "ps":
			return fmt.Errorf("Error creating container")
		}
		return fmt.Errorf("unexpected command")
	})
	defer patch2.Unpatch()

	// 4. Execute the function under test
	result, err := IsContainerExited(id)

	// 5. Assert
	require.Equal(t, result, false)
	require.Error(t, err)
	require.Contains(t, err.Error(), "Error creating container")
}

func TestIsContainerExitedOk(t *testing.T) {
	// 1. Setup
	threadId := "thread123"
	containerName := "myBlender" + threadId

	// 2. Monkey patch CommandContext to return an *exec.Cmd with visible arguments
	patch1 := monkey.Patch(exec.Command, func(name string, arg ...string) *exec.Cmd {
		return &exec.Cmd{
			Path: name,
			Args: append([]string{name}, arg...),
		}
	})
	defer patch1.Unpatch()

	// 3. Patch Run in order to simulate docker ps failure
	patch2 := monkey.PatchInstanceMethod(reflect.TypeOf(&exec.Cmd{}), "Run", func(cmd *exec.Cmd) error {
		switch cmd.Args[1] {
		case "ps":
			cmd.Stdout.Write([]byte(containerName))
			return nil
		}
		return fmt.Errorf("unexpected command")
	})
	defer patch2.Unpatch()

	// 4. Execute the function under test
	result, err := IsContainerExited(threadId)

	// 5. Assert
	require.Equal(t, result, true)
	require.NoError(t, err)
}
