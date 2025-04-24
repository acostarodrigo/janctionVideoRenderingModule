package vm

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"reflect"
	"testing"

	"bou.ke/monkey"
	"github.com/janction/videoRendering/mocks"
	"github.com/stretchr/testify/mock"
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

// --- Test for renderVideoFrame ---
func TestRenderVideoFrame_ContainerVerificationError(t *testing.T) {
	// 1. Setup
	mockDB := new(mocks.DB)
	ctx := context.Background()
	cid := "bafybeigdyrztxx3b7d5qzq2ujay5g4qxxuj5f6x3h6lgv7d4ttrddn3cxa"
	frameNumber := int64(42)
	id := "thread123"
	path := "/tmp/rendering/thread123/frame_42"

	// 2. Mock DB methods
	mockDB.On("AddLogEntry", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()

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
	err := renderVideoFrame(ctx, cid, frameNumber, id, path, mockDB)

	// 5. Verification
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to check container existence: Error verifying if container already exists")

	// 6. Verify mock expectations
	mockDB.AssertExpectations(t)
}

func TestRenderVideoFrame_ContainerAlreadyExist(t *testing.T) {
	// 1. Setup
	mockDB := new(mocks.DB)
	ctx := context.Background()
	cid := "bafybeigdyrztxx3b7d5qzq2ujay5g4qxxuj5f6x3h6lgv7d4ttrddn3cxa"
	frameNumber := int64(42)
	id := "thread123"
	path := "/tmp/rendering/thread123/frame_42"

	// 2. Mock DB methods
	mockDB.On("AddLogEntry", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()

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
	err := renderVideoFrame(ctx, cid, frameNumber, id, path, mockDB)

	// 5. Verification
	require.NoError(t, err)

	// 6. Verify mock expectations
	mockDB.AssertExpectations(t)
}

func TestRenderVideoFrame_CreatingContainerKo(t *testing.T) {
	// 1. Setup
	mockDB := new(mocks.DB)
	ctx := context.Background()
	cid := "bafybeigdyrztxx3b7d5qzq2ujay5g4qxxuj5f6x3h6lgv7d4ttrddn3cxa"
	frameNumber := int64(42)
	id := "thread123"
	path := "/tmp/rendering/thread123/frame_42"

	// 2. Mock DB methods
	mockDB.On("AddLogEntry", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()

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
	err := renderVideoFrame(ctx, cid, frameNumber, id, path, mockDB)

	// 7. Assert the error
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to create and start container: Error creating container")

	// 8. Verify mock expectations
	mockDB.AssertExpectations(t)
}

func TestRenderVideoFrame_CreatingContainerOk_WaitingContainerKo(t *testing.T) {
	// 1. Setup
	mockDB := new(mocks.DB)
	ctx := context.Background()
	cid := "bafybeigdyrztxx3b7d5qzq2ujay5g4qxxuj5f6x3h6lgv7d4ttrddn3cxa"
	frameNumber := int64(42)
	id := "thread123"
	path := "/tmp/rendering/thread123/frame_42"

	// 2. Mock DB methods
	mockDB.On("AddLogEntry", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()

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
	err := renderVideoFrame(ctx, cid, frameNumber, id, path, mockDB)

	// 7. Assert the error
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to wait for container: failed here")

	// 8. Verify mock expectations
	mockDB.AssertExpectations(t)
}

func TestRenderVideoFrame_CreatingContainerOk_WaitingContainerOk_RetrieveLogsKo(t *testing.T) {
	// 1. Setup
	mockDB := new(mocks.DB)
	ctx := context.Background()
	cid := "bafybeigdyrztxx3b7d5qzq2ujay5g4qxxuj5f6x3h6lgv7d4ttrddn3cxa"
	frameNumber := int64(42)
	id := "thread123"
	path := "/tmp/rendering/thread123/frame_42"

	// 2. Mock DB methods
	mockDB.On("AddLogEntry", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()

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

	// 8. Verify mock expectations
	mockDB.AssertExpectations(t)
}

func TestRenderVideoFrame_CreatingContainerOk_WaitingContainerOk_RetrieveLogsOk_VerifyFileOk(t *testing.T) {
	// 1. Setup
	mockDB := new(mocks.DB)
	ctx := context.Background()
	cid := "bafybeigdyrztxx3b7d5qzq2ujay5g4qxxuj5f6x3h6lgv7d4ttrddn3cxa"
	frameNumber := int64(42)
	id := "thread123"
	path := "/tmp/rendering/thread123/frame_42"

	// 2. Mock DB methods
	mockDB.On("AddLogEntry", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()

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
	err := renderVideoFrame(ctx, cid, frameNumber, id, path, mockDB)

	// 8. Assert no error
	require.NoError(t, err)

	// 9. Verify mock expectations
	mockDB.AssertExpectations(t)
}

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
