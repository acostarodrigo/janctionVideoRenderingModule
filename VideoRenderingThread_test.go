package videoRendering

import (
	"context"
	fmt "fmt"
	"os"
	"testing"
	"time"

	"bou.ke/monkey"
	"github.com/cosmos/cosmos-sdk/codec"
	types "github.com/cosmos/cosmos-sdk/codec/types"
	secp256k1 "github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	videoRenderingCrypto "github.com/janction/videoRendering/crypto"
	"github.com/janction/videoRendering/db"
	"github.com/janction/videoRendering/ipfs"
	"github.com/janction/videoRendering/mocks"
	"github.com/janction/videoRendering/vm"
	"github.com/stretchr/testify/mock"

	"github.com/stretchr/testify/require"
)

// --- Test for StartWork ---
func TestStartWork_ContainerRunning(t *testing.T) {
	// 1. Setup
	mockDB := new(mocks.DB)
	thread := &VideoRenderingThread{
		ThreadId:   "thread123",
		StartFrame: 0,
		EndFrame:   1,
	}

	// 2. Mock DB methods
	mockDB.On("UpdateThread", "thread123", false, false, true, false, false, false, false, false).Return(nil).Once()

	mockDB.On("AddLogEntry", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()

	// 3.1 Monkey patch vm methods
	patch1 := monkey.Patch(vm.IsContainerRunning, func(ctx context.Context, threadId string) bool {
		return true // Simulate that the container is running
	})
	defer patch1.Unpatch()

	// 4. Prepare test context and input values
	cid := "fakeCID"
	path := t.TempDir()
	ctx := context.Background()

	// 5. Execute method under test
	err := thread.StartWork(ctx, "worker1", cid, path, mockDB)
	require.NoError(t, err)

	// 6. Verify mock expectations
	mockDB.AssertExpectations(t)
}

func TestStartWork_ContainerNotRunning_IPFSRunning_IPFSGetKo(t *testing.T) {
	// 1. Setup
	mockDB := new(mocks.DB)
	thread := &VideoRenderingThread{
		ThreadId:   "thread123",
		StartFrame: 0,
		EndFrame:   1,
	}

	// 2. Mock DB methods
	mockDB.On("UpdateThread", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything,
		mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Times(3)

	mockDB.On("AddLogEntry", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()

	// 3.1 Monkey patch vm methods
	patch1 := monkey.Patch(vm.IsContainerRunning, func(ctx context.Context, threadId string) bool {
		return false // Simulate container is not running
	})
	defer patch1.Unpatch()

	patch2 := monkey.Patch(vm.RemoveContainer, func(ctx context.Context, name string) error {
		return nil // Simulate successful removal of container
	})
	defer patch2.Unpatch()

	// 3.2 Monkey patch ipfs methods
	patch3 := monkey.Patch(ipfs.EnsureIPFSRunning, func() {
		// no-op, simulate IPFS is not running
	})
	defer patch3.Unpatch()

	patch4 := monkey.Patch(ipfs.IPFSGet, func(cid string, outputPath string) error {
		// Simulate IPFS file is not available
		return fmt.Errorf("IPFS file is not available")
	})
	defer patch4.Unpatch()

	// 4. Prepare test context and input values
	cid := "fakeCID"
	path := t.TempDir()
	ctx := context.Background()

	// 5. Execute method under test
	err := thread.StartWork(ctx, "worker1", cid, path, mockDB)

	// 6. Verify that we got the expected error (IPFS file not available)
	require.Error(t, err)
	require.Contains(t, err.Error(), "IPFS file is not available")

	// 7. Verify mock expectations
	mockDB.AssertExpectations(t)
}

func TestStartWork_ContainerNotRunning_IPFSRunning_IPFSGetOk_RenderVideoKo(t *testing.T) {
	// 1. Setup
	mockDB := new(mocks.DB)
	thread := &VideoRenderingThread{
		ThreadId:   "thread123",
		StartFrame: 0,
		EndFrame:   1,
	}
	expected := false

	// 2. Mock DB methods
	mockDB.
		On("UpdateThread", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything,
			mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			// thread expected status validation
			if args.Bool(1) == true &&
				args.Bool(2) == true &&
				args.Bool(3) == false &&
				args.Bool(4) == false &&
				args.Bool(5) == false &&
				args.Bool(6) == false &&
				args.Bool(7) == false &&
				args.Bool(8) == false {
				expected = true
			}
		}).
		Return(nil).
		Times(4)

	mockDB.On("AddLogEntry", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()

	// 3.1 Monkey patch vm methods
	patch1 := monkey.Patch(vm.IsContainerRunning, func(ctx context.Context, threadId string) bool {
		return false // Simulate container is not running
	})
	defer patch1.Unpatch()

	patch2 := monkey.Patch(vm.RemoveContainer, func(ctx context.Context, name string) error {
		return nil // Simulate successful removal of container
	})
	defer patch2.Unpatch()

	patch3 := monkey.Patch(vm.RenderVideo, func(ctx context.Context, cid string, start int64, end int64, id string, path string, reverse bool, db db.Database) {
		// no-op, simulate video rendering
	})
	defer patch3.Unpatch()

	// 3.2 Monkey patch ipfs methods
	patch4 := monkey.Patch(ipfs.EnsureIPFSRunning, func() {
		// no-op, simulate IPFS is running
	})
	defer patch4.Unpatch()

	patch5 := monkey.Patch(ipfs.IPFSGet, func(cid string, outputPath string) error {
		// Simulate IPFS file is available
		return nil
	})
	defer patch5.Unpatch()

	// 3.3 Monkey patch os methods
	patch6 := monkey.Patch(os.Stat, func(name string) (os.FileInfo, error) {
		return nil, fmt.Errorf("Video rendering error") // Simulate video rendering failure
	})
	defer patch6.Unpatch()

	// 4. Prepare test context and input values
	cid := "fakeCID"
	path := t.TempDir()
	ctx := context.Background()

	// 5. Execute method under test
	err := thread.StartWork(ctx, "worker1", cid, path, mockDB)

	// 6. Verify that we got the expected thread status (video rendering error)
	require.NoError(t, err)
	require.True(t, expected, "Expected thread status (video rendering error)")

	// 7. Verify mock expectations
	mockDB.AssertExpectations(t)
}

func TestStartWork_ContainerNotRunning_IPFSRunning_IPFSGetOk_RenderVideoOk_FilesKo(t *testing.T) {
	// 1. Setup
	mockDB := new(mocks.DB)
	thread := &VideoRenderingThread{
		ThreadId:   "thread123",
		StartFrame: 0,
		EndFrame:   1,
	}
	file := mocks.MockFileInfo{
		Filename: "video1.mp4",
		Filesize: 2048,
		Filemode: 0644,
		ModTime_: time.Date(2025, 4, 11, 10, 0, 0, 0, time.UTC),
		IsDir_:   false,
	}
	files := []os.DirEntry{
		mocks.MockDirEntry{Filename: "video1.mp4", IsDir_: false},
	}
	expected := false

	// 2. Mock DB methods
	mockDB.
		On("UpdateThread", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything,
			mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			// thread expected status validation
			if args.Bool(1) == true &&
				args.Bool(2) == true &&
				args.Bool(3) == false &&
				args.Bool(4) == false &&
				args.Bool(5) == false &&
				args.Bool(6) == false &&
				args.Bool(7) == false &&
				args.Bool(8) == false {
				expected = true
			}
		}).
		Return(nil).
		Times(4)

	mockDB.On("AddLogEntry", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()

	// 3.1 Monkey patch vm methods
	patch1 := monkey.Patch(vm.IsContainerRunning, func(ctx context.Context, threadId string) bool {
		return false // Simulate container is not running
	})
	defer patch1.Unpatch()

	patch2 := monkey.Patch(vm.RemoveContainer, func(ctx context.Context, name string) error {
		return nil // Simulate successful removal of container
	})
	defer patch2.Unpatch()

	patch3 := monkey.Patch(vm.RenderVideo, func(ctx context.Context, cid string, start int64, end int64, id string, path string, reverse bool, db db.Database) {
		// no-op, simulate video rendering
	})
	defer patch3.Unpatch()

	// 3.2 Monkey patch ipfs methods
	patch4 := monkey.Patch(ipfs.EnsureIPFSRunning, func() {
		// no-op, simulate IPFS is running
	})
	defer patch4.Unpatch()

	patch5 := monkey.Patch(ipfs.IPFSGet, func(cid string, outputPath string) error {
		// Simulate IPFS file is available
		return nil
	})
	defer patch5.Unpatch()

	// 3.3 Monkey patch os methods
	patch6 := monkey.Patch(os.Stat, func(name string) (os.FileInfo, error) {
		return file, nil
	})
	defer patch6.Unpatch()

	patch7 := monkey.Patch(os.ReadDir, func(name string) ([]os.DirEntry, error) {
		return files, nil
	})
	defer patch7.Unpatch()

	// 4. Prepare test context and input values
	cid := "fakeCID"
	path := t.TempDir()
	ctx := context.Background()

	// 5. Execute method under test
	err := thread.StartWork(ctx, "worker1", cid, path, mockDB)

	// 6. Verify that we got the expected thread status (incorrect amount of files)
	require.NoError(t, err)
	require.True(t, expected, "Expected thread status (incorrect amount of files)")

	// 7. Verify mock expectations
	mockDB.AssertExpectations(t)
}

func TestStartWork_ContainerNotRunning_IPFSRunning_IPFSGetOk_RenderVideoOk_FilesOk(t *testing.T) {
	// 1. Setup
	mockDB := new(mocks.DB)
	thread := &VideoRenderingThread{
		ThreadId:   "thread123",
		StartFrame: 0,
		EndFrame:   1,
	}
	file := mocks.MockFileInfo{
		Filename: "video1.mp4",
		Filesize: 2048,
		Filemode: 0644,
		ModTime_: time.Date(2025, 4, 11, 10, 0, 0, 0, time.UTC),
		IsDir_:   false,
	}
	files := []os.DirEntry{
		mocks.MockDirEntry{Filename: "video1.mp4", IsDir_: false},
		mocks.MockDirEntry{Filename: "video2.mp4", IsDir_: false},
	}
	expected := false

	// 2. Mock DB methods
	mockDB.
		On("UpdateThread", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything,
			mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			// thread expected status validation
			if args.Bool(1) == true &&
				args.Bool(2) == true &&
				args.Bool(3) == true &&
				args.Bool(4) == true &&
				args.Bool(5) == false &&
				args.Bool(6) == false &&
				args.Bool(7) == false &&
				args.Bool(8) == false {
				expected = true
			}
		}).
		Return(nil).
		Times(4)

	mockDB.On("AddLogEntry", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()

	// 3.1 Monkey patch vm methods
	patch1 := monkey.Patch(vm.IsContainerRunning, func(ctx context.Context, threadId string) bool {
		return false // Simulate container is not running
	})
	defer patch1.Unpatch()

	patch2 := monkey.Patch(vm.RemoveContainer, func(ctx context.Context, name string) error {
		return nil // Simulate successful removal of container
	})
	defer patch2.Unpatch()

	patch3 := monkey.Patch(vm.RenderVideo, func(ctx context.Context, cid string, start int64, end int64, id string, path string, reverse bool, db db.Database) {
		// no-op, simulate video rendering
	})
	defer patch3.Unpatch()

	// 3.2 Monkey patch ipfs methods
	patch4 := monkey.Patch(ipfs.EnsureIPFSRunning, func() {
		// no-op, simulate IPFS is running
	})
	defer patch4.Unpatch()

	patch5 := monkey.Patch(ipfs.IPFSGet, func(cid string, outputPath string) error {
		// Simulate IPFS file is available
		return nil
	})
	defer patch5.Unpatch()

	// 3.3 Monkey patch os methods
	patch6 := monkey.Patch(os.Stat, func(name string) (os.FileInfo, error) {
		return file, nil
	})
	defer patch6.Unpatch()

	patch7 := monkey.Patch(os.ReadDir, func(name string) ([]os.DirEntry, error) {
		return files, nil
	})
	defer patch7.Unpatch()

	// 4. Prepare test context and input values
	cid := "fakeCID"
	path := t.TempDir()
	ctx := context.Background()

	// 5. Execute method under test
	err := thread.StartWork(ctx, "worker1", cid, path, mockDB)

	// 6. Verify that we got the expected thread status (video rendering error)
	require.NoError(t, err)
	require.True(t, expected, "Expected thread status (no error)")

	// 7. Verify mock expectations
	mockDB.AssertExpectations(t)
}

// --- Test for ProposeSolution ---
func TestProposeSolution_FrameAmountKo(t *testing.T) {
	// 1. Setup
	mockDB := new(mocks.DB)
	thread := &VideoRenderingThread{
		ThreadId:   "thread123",
		StartFrame: 0,
		EndFrame:   1,
	}
	cdc := codec.NewProtoCodec(types.NewInterfaceRegistry())
	expected := false

	// 2. Mock DB methods
	mockDB.
		On("UpdateThread", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything,
			mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			// thread expected status validation
			if args.Bool(1) == true &&
				args.Bool(2) == true &&
				args.Bool(3) == true &&
				args.Bool(4) == true &&
				args.Bool(5) == false &&
				args.Bool(6) == false &&
				args.Bool(7) == false &&
				args.Bool(8) == false {
				expected = true
			}
		}).
		Return(nil).
		Twice()

	// 3. Monkey patching
	patch1 := monkey.Patch(vm.CountFilesInDirectory, func(directoryPath string) int {
		return 1
	})
	defer patch1.Unpatch()

	err := thread.ProposeSolution(cdc, "worker-alias-001", "cosmos1abcdefg1234567", "/tmp/test-rendering", mockDB)

	// 6. Verify that we got the expected thread status (frame amount error)
	require.NoError(t, err)
	require.True(t, expected, "Expected thread status (frame amount error)")

	// 7. Verify mock expectations
	mockDB.AssertExpectations(t)
}

func TestProposeSolution_FrameAmountOk_GenerateHashesKo(t *testing.T) {
	// 1. Setup
	mockDB := new(mocks.DB)
	thread := &VideoRenderingThread{
		ThreadId:   "thread123",
		StartFrame: 0,
		EndFrame:   1,
	}
	cdc := codec.NewProtoCodec(types.NewInterfaceRegistry())

	// 2. Mock DB methods
	mockDB.
		On("UpdateThread", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything,
			mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(nil).
		Twice()

	// 3. Monkey patching
	patch1 := monkey.Patch(vm.CountFilesInDirectory, func(directoryPath string) int {
		return 2
	})
	defer patch1.Unpatch()

	patch2 := monkey.Patch(GenerateDirectoryFileHashes, func(directoryPath string) (map[string]string, error) {
		return nil, fmt.Errorf("Generate hash error")
	})
	defer patch2.Unpatch()

	err := thread.ProposeSolution(cdc, "worker-alias-001", "cosmos1abcdefg1234567", "/tmp/test-rendering", mockDB)

	// 6. Verify that we got the expected error
	require.Error(t, err)
	require.Contains(t, err.Error(), "Generate hash error")

	// 7. Verify mock expectations
	mockDB.AssertExpectations(t)
}

func TestProposeSolution_FrameAmountOk_GenerateHashesOk_ExtractPublicKeyKo(t *testing.T) {
	// 1. Setup
	mockDB := new(mocks.DB)
	thread := &VideoRenderingThread{
		ThreadId:   "thread123",
		StartFrame: 0,
		EndFrame:   1,
	}
	cdc := codec.NewProtoCodec(types.NewInterfaceRegistry())

	// 2. Mock DB methods
	mockDB.
		On("UpdateThread", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything,
			mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(nil).
		Twice()

	// 3. Monkey patching
	patch1 := monkey.Patch(vm.CountFilesInDirectory, func(directoryPath string) int {
		return 2
	})
	defer patch1.Unpatch()

	patch2 := monkey.Patch(GenerateDirectoryFileHashes, func(directoryPath string) (map[string]string, error) {
		return map[string]string{
			"video1.mp4": "a1b2c3d4e5f6g7h8i9j0",
			"video2.mp4": "1234567890abcdef1234",
		}, nil
	})
	defer patch2.Unpatch()

	patch3 := monkey.Patch(videoRenderingCrypto.ExtractPublicKey, func(rootDir, alias string, codec codec.Codec) (cryptotypes.PubKey, error) {
		return secp256k1.GenPrivKey().PubKey(), fmt.Errorf("Extracting public key error")
	})
	defer patch3.Unpatch()

	err := thread.ProposeSolution(cdc, "worker-alias-001", "cosmos1abcdefg1234567", "/tmp/test-rendering", mockDB)

	// 6. Verify that we got the expected error
	require.Error(t, err)
	require.Contains(t, err.Error(), "Extracting public key error")

	// 7. Verify mock expectations
	mockDB.AssertExpectations(t)
}

func TestProposeSolution_FrameAmountOk_GenerateHashesOk_ExtractPublicKeyOk_GenerateMessageKo(t *testing.T) {
	// 1. Setup
	mockDB := new(mocks.DB)
	thread := &VideoRenderingThread{
		ThreadId:   "thread123",
		StartFrame: 0,
		EndFrame:   1,
	}
	cdc := codec.NewProtoCodec(types.NewInterfaceRegistry())

	// 2. Mock DB methods
	mockDB.
		On("UpdateThread", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything,
			mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(nil).
		Twice()

	// 3. Monkey patching
	patch1 := monkey.Patch(vm.CountFilesInDirectory, func(directoryPath string) int {
		return 2
	})
	defer patch1.Unpatch()

	patch2 := monkey.Patch(GenerateDirectoryFileHashes, func(directoryPath string) (map[string]string, error) {
		return map[string]string{
			"video1.mp4": "a1b2c3d4e5f6g7h8i9j0",
			"video2.mp4": "1234567890abcdef1234",
		}, nil
	})
	defer patch2.Unpatch()

	patch3 := monkey.Patch(videoRenderingCrypto.ExtractPublicKey, func(rootDir, alias string, codec codec.Codec) (cryptotypes.PubKey, error) {
		return secp256k1.GenPrivKey().PubKey(), nil
	})
	defer patch3.Unpatch()

	patch4 := monkey.Patch(videoRenderingCrypto.GenerateSignableMessage, func(hash, workerAddr string) ([]byte, error) {
		return []byte("fake-signable-message"), fmt.Errorf("Generating message error")
	})
	defer patch4.Unpatch()

	err := thread.ProposeSolution(cdc, "worker-alias-001", "cosmos1abcdefg1234567", "/tmp/test-rendering", mockDB)

	// 6. Verify that we got the expected error
	require.Error(t, err)
	require.Contains(t, err.Error(), "Generating message error")

	// 7. Verify mock expectations
	mockDB.AssertExpectations(t)
}

func TestProposeSolution_FrameAmountOk_GenerateHashesOk_ExtractPublicKeyOk_GenerateMessageOk_SignMessageKo(t *testing.T) {
	// 1. Setup
	mockDB := new(mocks.DB)
	thread := &VideoRenderingThread{
		ThreadId:   "thread123",
		StartFrame: 0,
		EndFrame:   1,
	}
	cdc := codec.NewProtoCodec(types.NewInterfaceRegistry())

	// 2. Mock DB methods
	mockDB.
		On("UpdateThread", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything,
			mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(nil).
		Twice()

	// 3. Monkey patching
	patch1 := monkey.Patch(vm.CountFilesInDirectory, func(directoryPath string) int {
		return 2
	})
	defer patch1.Unpatch()

	patch2 := monkey.Patch(GenerateDirectoryFileHashes, func(directoryPath string) (map[string]string, error) {
		return map[string]string{
			"video1.mp4": "a1b2c3d4e5f6g7h8i9j0",
			"video2.mp4": "1234567890abcdef1234",
		}, nil
	})
	defer patch2.Unpatch()

	patch3 := monkey.Patch(videoRenderingCrypto.ExtractPublicKey, func(rootDir, alias string, codec codec.Codec) (cryptotypes.PubKey, error) {
		return secp256k1.GenPrivKey().PubKey(), nil
	})
	defer patch3.Unpatch()

	patch4 := monkey.Patch(videoRenderingCrypto.GenerateSignableMessage, func(hash, workerAddr string) ([]byte, error) {
		return []byte("fake-signable-message"), nil
	})
	defer patch4.Unpatch()

	patch5 := monkey.Patch(videoRenderingCrypto.SignMessage, func(rootDir, alias string, message []byte, codec codec.Codec) ([]byte, cryptotypes.PubKey, error) {
		return []byte("fake-signable-message"), secp256k1.GenPrivKey().PubKey(), fmt.Errorf("Signing message error")
	})
	defer patch5.Unpatch()

	err := thread.ProposeSolution(cdc, "worker-alias-001", "cosmos1abcdefg1234567", "/tmp/test-rendering", mockDB)

	// 6. Verify that we got the expected error
	require.Error(t, err)
	require.Contains(t, err.Error(), "Signing message error")

	// 7. Verify mock expectations
	mockDB.AssertExpectations(t)
}

func TestProposeSolution_FrameAmountOk_GenerateHashesOk_ExtractPublicKeyOk_GenerateMessageOk_SignMessageOk_SolutionKo(t *testing.T) {
	// 1. Setup
	mockDB := new(mocks.DB)
	thread := &VideoRenderingThread{
		ThreadId:   "thread123",
		StartFrame: 0,
		EndFrame:   1,
	}
	cdc := codec.NewProtoCodec(types.NewInterfaceRegistry())

	// 2. Mock DB methods
	mockDB.
		On("UpdateThread", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything,
			mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(nil).
		Once()

	// 3. Monkey patching
	patch1 := monkey.Patch(vm.CountFilesInDirectory, func(directoryPath string) int {
		return 2
	})
	defer patch1.Unpatch()

	patch2 := monkey.Patch(GenerateDirectoryFileHashes, func(directoryPath string) (map[string]string, error) {
		return map[string]string{
			"video1.mp4": "a1b2c3d4e5f6g7h8i9j0",
			"video2.mp4": "1234567890abcdef1234",
		}, nil
	})
	defer patch2.Unpatch()

	patch3 := monkey.Patch(videoRenderingCrypto.ExtractPublicKey, func(rootDir, alias string, codec codec.Codec) (cryptotypes.PubKey, error) {
		return secp256k1.GenPrivKey().PubKey(), nil
	})
	defer patch3.Unpatch()

	patch4 := monkey.Patch(videoRenderingCrypto.GenerateSignableMessage, func(hash, workerAddr string) ([]byte, error) {
		return []byte("fake-signable-message"), nil
	})
	defer patch4.Unpatch()

	patch5 := monkey.Patch(videoRenderingCrypto.SignMessage, func(rootDir, alias string, message []byte, codec codec.Codec) ([]byte, cryptotypes.PubKey, error) {
		return []byte("fake-signable-message"), secp256k1.GenPrivKey().PubKey(), nil
	})
	defer patch5.Unpatch()

	patch6 := monkey.Patch(ExecuteCli, func(args []string) error {
		return fmt.Errorf("Solution error")
	})
	defer patch6.Unpatch()

	err := thread.ProposeSolution(cdc, "worker-alias-001", "cosmos1abcdefg1234567", "/tmp/test-rendering", mockDB)

	// 6. Verify that we got the expected error
	require.Error(t, err)
	require.Contains(t, err.Error(), "Solution error")

	// 7. Verify mock expectations
	mockDB.AssertExpectations(t)
}

func TestProposeSolution_FrameAmountOk_GenerateHashesOk_ExtractPublicKeyOk_GenerateMessageOk_SignMessageOk_SolutionOk(t *testing.T) {
	// 1. Setup
	mockDB := new(mocks.DB)
	thread := &VideoRenderingThread{
		ThreadId:   "thread123",
		StartFrame: 0,
		EndFrame:   1,
	}
	cdc := codec.NewProtoCodec(types.NewInterfaceRegistry())

	// 2. Mock DB methods
	mockDB.
		On("UpdateThread", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything,
			mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(nil).
		Once()

	mockDB.On("AddLogEntry", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()

	// 3. Monkey patching
	patch1 := monkey.Patch(vm.CountFilesInDirectory, func(directoryPath string) int {
		return 2
	})
	defer patch1.Unpatch()

	patch2 := monkey.Patch(GenerateDirectoryFileHashes, func(directoryPath string) (map[string]string, error) {
		return map[string]string{
			"video1.mp4": "a1b2c3d4e5f6g7h8i9j0",
			"video2.mp4": "1234567890abcdef1234",
		}, nil
	})
	defer patch2.Unpatch()

	patch3 := monkey.Patch(videoRenderingCrypto.ExtractPublicKey, func(rootDir, alias string, codec codec.Codec) (cryptotypes.PubKey, error) {
		return secp256k1.GenPrivKey().PubKey(), nil
	})
	defer patch3.Unpatch()

	patch4 := monkey.Patch(videoRenderingCrypto.GenerateSignableMessage, func(hash, workerAddr string) ([]byte, error) {
		return []byte("fake-signable-message"), nil
	})
	defer patch4.Unpatch()

	patch5 := monkey.Patch(videoRenderingCrypto.SignMessage, func(rootDir, alias string, message []byte, codec codec.Codec) ([]byte, cryptotypes.PubKey, error) {
		return []byte("fake-signable-message"), secp256k1.GenPrivKey().PubKey(), nil
	})
	defer patch5.Unpatch()

	patch6 := monkey.Patch(ExecuteCli, func(args []string) error {
		return nil
	})
	defer patch6.Unpatch()

	err := thread.ProposeSolution(cdc, "worker-alias-001", "cosmos1abcdefg1234567", "/tmp/test-rendering", mockDB)

	// 6. Verify that we got no error
	require.NoError(t, err)

	// 7. Verify mock expectations
	mockDB.AssertExpectations(t)
}

func TestSubmitVerification_FileCountKo(t *testing.T) {
	// 1. Setup
	mockDB := new(mocks.DB)
	thread := &VideoRenderingThread{
		ThreadId:   "thread123",
		StartFrame: 0,
		EndFrame:   1,
	}
	cdc := codec.NewProtoCodec(types.NewInterfaceRegistry())
	expected := false

	// 2. Mock DB methods
	mockDB.
		On("UpdateThread", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything,
			mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			// thread expected status validation
			if args.Bool(1) == true &&
				args.Bool(2) == true &&
				args.Bool(3) == true &&
				args.Bool(4) == true &&
				args.Bool(5) == true &&
				args.Bool(6) == false &&
				args.Bool(7) == false &&
				args.Bool(8) == false {
				expected = true
			}
		}).
		Return(nil).
		Twice()

	// 3. Monkey patching
	patch1 := monkey.Patch(vm.CountFilesInDirectory, func(directoryPath string) int {
		return 0
	})
	defer patch1.Unpatch()

	err := thread.SubmitVerification(cdc, "worker-alias-001", "cosmos1abcdefg1234567", "/tmp/test-rendering", mockDB)

	// 6. Verify that we got the expected thread status (file amount error)
	require.NoError(t, err)
	require.True(t, expected, "Expected thread status (file amount error)")

	// 7. Verify mock expectations
	mockDB.AssertExpectations(t)
}

func TestSubmitVerification_FileCountOk_FileThresholdKo(t *testing.T) {
	// 1. Setup
	mockDB := new(mocks.DB)
	thread := &VideoRenderingThread{
		ThreadId:   "thread123",
		StartFrame: 0,
		EndFrame:   10,
	}
	cdc := codec.NewProtoCodec(types.NewInterfaceRegistry())
	expected := false

	// 2. Mock DB methods
	mockDB.
		On("UpdateThread", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything,
			mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			// thread expected status validation
			if args.Bool(1) == true &&
				args.Bool(2) == true &&
				args.Bool(3) == true &&
				args.Bool(4) == true &&
				args.Bool(5) == true &&
				args.Bool(6) == false &&
				args.Bool(7) == false &&
				args.Bool(8) == false {
				expected = true
			}
		}).
		Return(nil).
		Twice()

	// 3. Monkey patching
	patch1 := monkey.Patch(vm.CountFilesInDirectory, func(directoryPath string) int {
		return 1
	})
	defer patch1.Unpatch()

	err := thread.SubmitVerification(cdc, "worker-alias-001", "cosmos1abcdefg1234567", "/tmp/test-rendering", mockDB)

	// 6. Verify that we got the expected error
	require.NoError(t, err)
	require.True(t, expected, "Expected thread status (file threshold error)")

	// 7. Verify mock expectations
	mockDB.AssertExpectations(t)
}

func TestSubmitVerification_FileCountOk_FileThresholdOk_GenerateFileHashesKo(t *testing.T) {
	// 1. Setup
	mockDB := new(mocks.DB)
	thread := &VideoRenderingThread{
		ThreadId:   "thread123",
		StartFrame: 0,
		EndFrame:   1,
	}
	cdc := codec.NewProtoCodec(types.NewInterfaceRegistry())

	// 2. Mock DB methods
	mockDB.
		On("UpdateThread", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything,
			mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(nil).
		Twice()

	// 3. Monkey patching
	patch1 := monkey.Patch(vm.CountFilesInDirectory, func(directoryPath string) int {
		return 1
	})
	defer patch1.Unpatch()

	patch2 := monkey.Patch(GenerateDirectoryFileHashes, func(directoryPath string) (map[string]string, error) {
		return nil, fmt.Errorf("Generate hash error")
	})
	defer patch2.Unpatch()

	err := thread.SubmitVerification(cdc, "worker-alias-001", "cosmos1abcdefg1234567", "/tmp/test-rendering", mockDB)

	// 6. Verify that we got the expected error
	require.Error(t, err)
	require.Contains(t, err.Error(), "Generate hash error")

	// 7. Verify mock expectations
	mockDB.AssertExpectations(t)
}

func TestSubmitVerification_FileCountOk_FileThresholdOk_GenerateFileHashesOk_GetPublicKeyKo(t *testing.T) {
	// 1. Setup
	mockDB := new(mocks.DB)
	thread := &VideoRenderingThread{
		ThreadId:   "thread123",
		StartFrame: 0,
		EndFrame:   1,
	}
	cdc := codec.NewProtoCodec(types.NewInterfaceRegistry())

	// 2. Mock DB methods
	mockDB.
		On("UpdateThread", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything,
			mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(nil).
		Twice()

	// 3. Monkey patching
	patch1 := monkey.Patch(vm.CountFilesInDirectory, func(directoryPath string) int {
		return 1
	})
	defer patch1.Unpatch()

	patch2 := monkey.Patch(GenerateDirectoryFileHashes, func(directoryPath string) (map[string]string, error) {
		return map[string]string{
			"video1.mp4": "a1b2c3d4e5f6g7h8i9j0",
		}, nil
	})
	defer patch2.Unpatch()

	patch3 := monkey.Patch(videoRenderingCrypto.GetPublicKey, func(rootDir, alias string, codec codec.Codec) (cryptotypes.PubKey, error) {
		return secp256k1.GenPrivKey().PubKey(), fmt.Errorf("Get public key error")
	})
	defer patch3.Unpatch()

	err := thread.SubmitVerification(cdc, "worker-alias-001", "cosmos1abcdefg1234567", "/tmp/test-rendering", mockDB)

	// 6. Verify that we got the expected error
	require.Error(t, err)
	require.Contains(t, err.Error(), "Get public key error")

	// 7. Verify mock expectations
	mockDB.AssertExpectations(t)
}

func TestSubmitVerification_FileCountOk_FileThresholdOk_GenerateFileHashesOk_GetPublicKeyOk_GenerateSignableMessageKo(t *testing.T) {
	// 1. Setup
	mockDB := new(mocks.DB)
	thread := &VideoRenderingThread{
		ThreadId:   "thread123",
		StartFrame: 0,
		EndFrame:   1,
	}
	cdc := codec.NewProtoCodec(types.NewInterfaceRegistry())

	// 2. Mock DB methods
	mockDB.
		On("UpdateThread", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything,
			mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(nil).
		Twice()

	// 3. Monkey patching
	patch1 := monkey.Patch(vm.CountFilesInDirectory, func(directoryPath string) int {
		return 1
	})
	defer patch1.Unpatch()

	patch2 := monkey.Patch(GenerateDirectoryFileHashes, func(directoryPath string) (map[string]string, error) {
		return map[string]string{
			"video1.mp4": "a1b2c3d4e5f6g7h8i9j0",
		}, nil
	})
	defer patch2.Unpatch()

	patch3 := monkey.Patch(videoRenderingCrypto.GetPublicKey, func(rootDir, alias string, codec codec.Codec) (cryptotypes.PubKey, error) {
		return secp256k1.GenPrivKey().PubKey(), nil
	})
	defer patch3.Unpatch()

	patch4 := monkey.Patch(videoRenderingCrypto.GenerateSignableMessage, func(hash, workerAddr string) ([]byte, error) {
		return []byte("fake-signable-message"), fmt.Errorf("Generating message error")
	})
	defer patch4.Unpatch()

	err := thread.SubmitVerification(cdc, "worker-alias-001", "cosmos1abcdefg1234567", "/tmp/test-rendering", mockDB)

	// 6. Verify that we got the expected error
	require.Error(t, err)
	require.Contains(t, err.Error(), "Generating message error")

	// 7. Verify mock expectations
	mockDB.AssertExpectations(t)
}

func TestSubmitVerification_FileCountOk_FileThresholdOk_GenerateFileHashesOk_GetPublicKeyOk_GenerateSignableMessageOk_SignMessageKo(t *testing.T) {
	// 1. Setup
	mockDB := new(mocks.DB)
	thread := &VideoRenderingThread{
		ThreadId:   "thread123",
		StartFrame: 0,
		EndFrame:   1,
	}
	cdc := codec.NewProtoCodec(types.NewInterfaceRegistry())

	// 2. Mock DB methods
	mockDB.
		On("UpdateThread", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything,
			mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(nil).
		Twice()

	// 3. Monkey patching
	patch1 := monkey.Patch(vm.CountFilesInDirectory, func(directoryPath string) int {
		return 1
	})
	defer patch1.Unpatch()

	patch2 := monkey.Patch(GenerateDirectoryFileHashes, func(directoryPath string) (map[string]string, error) {
		return map[string]string{
			"video1.mp4": "a1b2c3d4e5f6g7h8i9j0",
		}, nil
	})
	defer patch2.Unpatch()

	patch3 := monkey.Patch(videoRenderingCrypto.GetPublicKey, func(rootDir, alias string, codec codec.Codec) (cryptotypes.PubKey, error) {
		return secp256k1.GenPrivKey().PubKey(), nil
	})
	defer patch3.Unpatch()

	patch4 := monkey.Patch(videoRenderingCrypto.GenerateSignableMessage, func(hash, workerAddr string) ([]byte, error) {
		return []byte("fake-signable-message"), nil
	})
	defer patch4.Unpatch()

	patch5 := monkey.Patch(videoRenderingCrypto.SignMessage, func(rootDir, alias string, message []byte, codec codec.Codec) ([]byte, cryptotypes.PubKey, error) {
		return []byte("fake-signable-message"), secp256k1.GenPrivKey().PubKey(), fmt.Errorf("Signing message error")
	})
	defer patch5.Unpatch()

	err := thread.SubmitVerification(cdc, "worker-alias-001", "cosmos1abcdefg1234567", "/tmp/test-rendering", mockDB)

	// 6. Verify that we got the expected error
	require.Error(t, err)
	require.Contains(t, err.Error(), "Signing message error")

	// 7. Verify mock expectations
	mockDB.AssertExpectations(t)
}

func TestSubmitVerification_FileCountOk_FileThresholdOk_GenerateFileHashesOk_GetPublicKeyOk_GenerateSignableMessageOk_SignMessageOk_SubmitValidationKo(t *testing.T) {
	// 1. Setup
	mockDB := new(mocks.DB)
	thread := &VideoRenderingThread{
		ThreadId:   "thread123",
		StartFrame: 0,
		EndFrame:   1,
	}
	cdc := codec.NewProtoCodec(types.NewInterfaceRegistry())

	// 2. Mock DB methods
	mockDB.
		On("UpdateThread", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything,
			mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(nil).
		Twice()

	mockDB.On("AddLogEntry", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()

	// 3. Monkey patching
	patch1 := monkey.Patch(vm.CountFilesInDirectory, func(directoryPath string) int {
		return 1
	})
	defer patch1.Unpatch()

	patch2 := monkey.Patch(GenerateDirectoryFileHashes, func(directoryPath string) (map[string]string, error) {
		return map[string]string{
			"video1.mp4": "a1b2c3d4e5f6g7h8i9j0",
		}, nil
	})
	defer patch2.Unpatch()

	patch3 := monkey.Patch(videoRenderingCrypto.GetPublicKey, func(rootDir, alias string, codec codec.Codec) (cryptotypes.PubKey, error) {
		return secp256k1.GenPrivKey().PubKey(), nil
	})
	defer patch3.Unpatch()

	patch4 := monkey.Patch(videoRenderingCrypto.GenerateSignableMessage, func(hash, workerAddr string) ([]byte, error) {
		return []byte("fake-signable-message"), nil
	})
	defer patch4.Unpatch()

	patch5 := monkey.Patch(videoRenderingCrypto.SignMessage, func(rootDir, alias string, message []byte, codec codec.Codec) ([]byte, cryptotypes.PubKey, error) {
		return []byte("fake-signable-message"), secp256k1.GenPrivKey().PubKey(), nil
	})
	defer patch5.Unpatch()

	patch6 := monkey.Patch(submitValidation, func(validator string, taskId, threadId, publicKey string, signatures []string) error {
		return fmt.Errorf("Submit validation error")
	})
	defer patch6.Unpatch()

	err := thread.SubmitVerification(cdc, "worker-alias-001", "cosmos1abcdefg1234567", "/tmp/test-rendering", mockDB)

	// 6. Verify that we got the expected error
	require.Error(t, err)
	require.Contains(t, err.Error(), "Submit validation error")

	// 7. Verify mock expectations
	mockDB.AssertExpectations(t)
}

func TestSubmitVerification_FileCountOk_FileThresholdOk_GenerateFileHashesOk_GetPublicKeyOk_GenerateSignableMessageOk_SignMessageOk_SubmitValidationOk(t *testing.T) {
	// 1. Setup
	mockDB := new(mocks.DB)
	thread := &VideoRenderingThread{
		ThreadId:   "thread123",
		StartFrame: 0,
		EndFrame:   1,
	}
	cdc := codec.NewProtoCodec(types.NewInterfaceRegistry())

	// 2. Mock DB methods
	mockDB.
		On("UpdateThread", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything,
			mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(nil).
		Once()

	mockDB.On("AddLogEntry", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()

	// 3. Monkey patching
	patch1 := monkey.Patch(vm.CountFilesInDirectory, func(directoryPath string) int {
		return 1
	})
	defer patch1.Unpatch()

	patch2 := monkey.Patch(GenerateDirectoryFileHashes, func(directoryPath string) (map[string]string, error) {
		return map[string]string{
			"video1.mp4": "a1b2c3d4e5f6g7h8i9j0",
		}, nil
	})
	defer patch2.Unpatch()

	patch3 := monkey.Patch(videoRenderingCrypto.GetPublicKey, func(rootDir, alias string, codec codec.Codec) (cryptotypes.PubKey, error) {
		return secp256k1.GenPrivKey().PubKey(), nil
	})
	defer patch3.Unpatch()

	patch4 := monkey.Patch(videoRenderingCrypto.GenerateSignableMessage, func(hash, workerAddr string) ([]byte, error) {
		return []byte("fake-signable-message"), nil
	})
	defer patch4.Unpatch()

	patch5 := monkey.Patch(videoRenderingCrypto.SignMessage, func(rootDir, alias string, message []byte, codec codec.Codec) ([]byte, cryptotypes.PubKey, error) {
		return []byte("fake-signable-message"), secp256k1.GenPrivKey().PubKey(), nil
	})
	defer patch5.Unpatch()

	patch6 := monkey.Patch(submitValidation, func(validator string, taskId, threadId, publicKey string, signatures []string) error {
		return nil
	})
	defer patch6.Unpatch()

	err := thread.SubmitVerification(cdc, "worker-alias-001", "cosmos1abcdefg1234567", "/tmp/test-rendering", mockDB)

	// 6. Verify that we got no error
	require.NoError(t, err)

	// 7. Verify mock expectations
	mockDB.AssertExpectations(t)
}
