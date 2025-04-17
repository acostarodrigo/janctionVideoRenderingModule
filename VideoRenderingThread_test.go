package videoRendering

import (
	"context"
	fmt "fmt"
	"os"
	"testing"
	"time"

	"bou.ke/monkey"
	sdkmath "cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/codec"
	types "github.com/cosmos/cosmos-sdk/codec/types"
	secp256k1 "github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	c_types "github.com/cosmos/cosmos-sdk/types"
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
	// Setup
	mockDB := new(mocks.DB)
	thread := &VideoRenderingThread{
		ThreadId:   "thread123",
		StartFrame: 0,
		EndFrame:   1,
	}

	// Mock DB methods
	mockDB.On("UpdateThread", "thread123", false, false, true, false, false, false, false, false).Return(nil).Once()

	mockDB.On("AddLogEntry", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()

	// Monkey patching
	patch1 := monkey.Patch(vm.IsContainerRunning, func(ctx context.Context, threadId string) bool {
		return true // Simulate that the container is running
	})
	defer patch1.Unpatch()

	// Prepare test context and input values
	cid := "fakeCID"
	path := t.TempDir()
	ctx := context.Background()

	// Execute method under test
	err := thread.StartWork(ctx, "worker1", cid, path, mockDB)
	require.NoError(t, err)

	// Verify mock expectations
	mockDB.AssertExpectations(t)
}

func TestStartWork_ContainerNotRunning_IPFSRunning_IPFSGetKo(t *testing.T) {
	// Setup
	mockDB := new(mocks.DB)
	thread := &VideoRenderingThread{
		ThreadId:   "thread123",
		StartFrame: 0,
		EndFrame:   1,
	}

	// Mock DB methods
	mockDB.On("UpdateThread", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything,
		mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Times(3)

	mockDB.On("AddLogEntry", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()

	// Monkey patching
	patch1 := monkey.Patch(vm.IsContainerRunning, func(ctx context.Context, threadId string) bool {
		return false // Simulate container is not running
	})
	defer patch1.Unpatch()

	patch2 := monkey.Patch(vm.RemoveContainer, func(ctx context.Context, name string) error {
		return nil // Simulate successful removal of container
	})
	defer patch2.Unpatch()

	patch3 := monkey.Patch(ipfs.EnsureIPFSRunning, func() {
		// no-op, simulate IPFS is not running
	})
	defer patch3.Unpatch()

	patch4 := monkey.Patch(ipfs.IPFSGet, func(cid string, outputPath string) error {
		// Simulate IPFS file is not available
		return fmt.Errorf("IPFS file is not available")
	})
	defer patch4.Unpatch()

	// Prepare test context and input values
	cid := "fakeCID"
	path := t.TempDir()
	ctx := context.Background()

	// Execute method under test
	err := thread.StartWork(ctx, "worker1", cid, path, mockDB)

	// Verify that we got the expected error (IPFS file not available)
	require.Error(t, err)
	require.Contains(t, err.Error(), "IPFS file is not available")

	// Verify mock expectations
	mockDB.AssertExpectations(t)
}

func TestStartWork_ContainerNotRunning_IPFSRunning_IPFSGetOk_RenderVideoKo(t *testing.T) {
	// Setup
	mockDB := new(mocks.DB)
	thread := &VideoRenderingThread{
		ThreadId:   "thread123",
		StartFrame: 0,
		EndFrame:   1,
	}
	expected := false

	// Mock DB methods
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

	// Monkey patching
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

	patch4 := monkey.Patch(ipfs.EnsureIPFSRunning, func() {
		// no-op, simulate IPFS is running
	})
	defer patch4.Unpatch()

	patch5 := monkey.Patch(ipfs.IPFSGet, func(cid string, outputPath string) error {
		// Simulate IPFS file is available
		return nil
	})
	defer patch5.Unpatch()

	patch6 := monkey.Patch(os.Stat, func(name string) (os.FileInfo, error) {
		return nil, fmt.Errorf("Video rendering error") // Simulate video rendering failure
	})
	defer patch6.Unpatch()

	// Prepare test context and input values
	cid := "fakeCID"
	path := t.TempDir()
	ctx := context.Background()

	// Execute method under test
	err := thread.StartWork(ctx, "worker1", cid, path, mockDB)

	// Verify that we got the expected thread status (video rendering error)
	require.NoError(t, err)
	require.True(t, expected, "Expected thread status (video rendering error)")

	// Verify mock expectations
	mockDB.AssertExpectations(t)
}

func TestStartWork_ContainerNotRunning_IPFSRunning_IPFSGetOk_RenderVideoOk_FilesKo(t *testing.T) {
	// Setup
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

	// Mock DB methods
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

	// Monkey patching
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

	patch4 := monkey.Patch(ipfs.EnsureIPFSRunning, func() {
		// no-op, simulate IPFS is running
	})
	defer patch4.Unpatch()

	patch5 := monkey.Patch(ipfs.IPFSGet, func(cid string, outputPath string) error {
		// Simulate IPFS file is available
		return nil
	})
	defer patch5.Unpatch()

	patch6 := monkey.Patch(os.Stat, func(name string) (os.FileInfo, error) {
		return file, nil
	})
	defer patch6.Unpatch()

	patch7 := monkey.Patch(os.ReadDir, func(name string) ([]os.DirEntry, error) {
		return files, nil
	})
	defer patch7.Unpatch()

	// Prepare test context and input values
	cid := "fakeCID"
	path := t.TempDir()
	ctx := context.Background()

	// Execute method under test
	err := thread.StartWork(ctx, "worker1", cid, path, mockDB)

	// Verify that we got the expected thread status (incorrect amount of files)
	require.NoError(t, err)
	require.True(t, expected, "Expected thread status (incorrect amount of files)")

	// Verify mock expectations
	mockDB.AssertExpectations(t)
}

func TestStartWork_ContainerNotRunning_IPFSRunning_IPFSGetOk_RenderVideoOk_FilesOk(t *testing.T) {
	// Setup
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

	// Mock DB methods
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

	// Monkey patching
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

	patch4 := monkey.Patch(ipfs.EnsureIPFSRunning, func() {
		// no-op, simulate IPFS is running
	})
	defer patch4.Unpatch()

	patch5 := monkey.Patch(ipfs.IPFSGet, func(cid string, outputPath string) error {
		// Simulate IPFS file is available
		return nil
	})
	defer patch5.Unpatch()

	patch6 := monkey.Patch(os.Stat, func(name string) (os.FileInfo, error) {
		return file, nil
	})
	defer patch6.Unpatch()

	patch7 := monkey.Patch(os.ReadDir, func(name string) ([]os.DirEntry, error) {
		return files, nil
	})
	defer patch7.Unpatch()

	// Prepare test context and input values
	cid := "fakeCID"
	path := t.TempDir()
	ctx := context.Background()

	// Execute method under test
	err := thread.StartWork(ctx, "worker1", cid, path, mockDB)

	// Verify that we got the expected thread status (video rendering error)
	require.NoError(t, err)
	require.True(t, expected, "Expected thread status (no error)")

	// Verify mock expectations
	mockDB.AssertExpectations(t)
}

// --- Test for ProposeSolution ---
func TestProposeSolution_FrameAmountKo(t *testing.T) {
	// Setup
	mockDB := new(mocks.DB)
	thread := &VideoRenderingThread{
		ThreadId:   "thread123",
		StartFrame: 0,
		EndFrame:   1,
	}
	cdc := codec.NewProtoCodec(types.NewInterfaceRegistry())
	expected := false

	// Mock DB methods
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

	// Monkey patching
	patch1 := monkey.Patch(vm.CountFilesInDirectory, func(directoryPath string) int {
		return 1
	})
	defer patch1.Unpatch()

	err := thread.ProposeSolution(cdc, "worker-alias-001", "cosmos1abcdefg1234567", "/tmp/test-rendering", mockDB)

	// Verify that we got the expected thread status (frame amount error)
	require.NoError(t, err)
	require.True(t, expected, "Expected thread status (frame amount error)")

	// Verify mock expectations
	mockDB.AssertExpectations(t)
}

func TestProposeSolution_FrameAmountOk_GenerateHashesKo(t *testing.T) {
	// Setup
	mockDB := new(mocks.DB)
	thread := &VideoRenderingThread{
		ThreadId:   "thread123",
		StartFrame: 0,
		EndFrame:   1,
	}
	cdc := codec.NewProtoCodec(types.NewInterfaceRegistry())

	// Mock DB methods
	mockDB.
		On("UpdateThread", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything,
			mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(nil).
		Twice()

	// Monkey patching
	patch1 := monkey.Patch(vm.CountFilesInDirectory, func(directoryPath string) int {
		return 2
	})
	defer patch1.Unpatch()

	patch2 := monkey.Patch(GenerateDirectoryFileHashes, func(directoryPath string) (map[string]string, error) {
		return nil, fmt.Errorf("Generate hash error")
	})
	defer patch2.Unpatch()

	err := thread.ProposeSolution(cdc, "worker-alias-001", "cosmos1abcdefg1234567", "/tmp/test-rendering", mockDB)

	// Verify that we got the expected error
	require.Error(t, err)
	require.Contains(t, err.Error(), "Generate hash error")

	// Verify mock expectations
	mockDB.AssertExpectations(t)
}

func TestProposeSolution_FrameAmountOk_GenerateHashesOk_ExtractPublicKeyKo(t *testing.T) {
	// Setup
	mockDB := new(mocks.DB)
	thread := &VideoRenderingThread{
		ThreadId:   "thread123",
		StartFrame: 0,
		EndFrame:   1,
	}
	cdc := codec.NewProtoCodec(types.NewInterfaceRegistry())

	// Mock DB methods
	mockDB.
		On("UpdateThread", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything,
			mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(nil).
		Twice()

	// Monkey patching
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

	// Verify that we got the expected error
	require.Error(t, err)
	require.Contains(t, err.Error(), "Extracting public key error")

	// Verify mock expectations
	mockDB.AssertExpectations(t)
}

func TestProposeSolution_FrameAmountOk_GenerateHashesOk_ExtractPublicKeyOk_GenerateMessageKo(t *testing.T) {
	// Setup
	mockDB := new(mocks.DB)
	thread := &VideoRenderingThread{
		ThreadId:   "thread123",
		StartFrame: 0,
		EndFrame:   1,
	}
	cdc := codec.NewProtoCodec(types.NewInterfaceRegistry())

	// Mock DB methods
	mockDB.
		On("UpdateThread", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything,
			mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(nil).
		Twice()

	// Monkey patching
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
		return []byte("fake-signable-message"), fmt.Errorf("GenerateSignableMessage error")
	})
	defer patch4.Unpatch()

	err := thread.ProposeSolution(cdc, "worker-alias-001", "cosmos1abcdefg1234567", "/tmp/test-rendering", mockDB)

	// Verify that we got the expected error
	require.Error(t, err)
	require.Contains(t, err.Error(), "GenerateSignableMessage error")

	// Verify mock expectations
	mockDB.AssertExpectations(t)
}

func TestProposeSolution_FrameAmountOk_GenerateHashesOk_ExtractPublicKeyOk_GenerateMessageOk_SignMessageKo(t *testing.T) {
	// Setup
	mockDB := new(mocks.DB)
	thread := &VideoRenderingThread{
		ThreadId:   "thread123",
		StartFrame: 0,
		EndFrame:   1,
	}
	cdc := codec.NewProtoCodec(types.NewInterfaceRegistry())

	// Mock DB methods
	mockDB.
		On("UpdateThread", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything,
			mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(nil).
		Twice()

	// Monkey patching
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

	// Verify that we got the expected error
	require.Error(t, err)
	require.Contains(t, err.Error(), "Signing message error")

	// Verify mock expectations
	mockDB.AssertExpectations(t)
}

func TestProposeSolution_FrameAmountOk_GenerateHashesOk_ExtractPublicKeyOk_GenerateMessageOk_SignMessageOk_SolutionKo(t *testing.T) {
	// Setup
	mockDB := new(mocks.DB)
	thread := &VideoRenderingThread{
		ThreadId:   "thread123",
		StartFrame: 0,
		EndFrame:   1,
	}
	cdc := codec.NewProtoCodec(types.NewInterfaceRegistry())

	// Mock DB methods
	mockDB.
		On("UpdateThread", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything,
			mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(nil).
		Once()

	// Monkey patching
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

	// Verify that we got the expected error
	require.Error(t, err)
	require.Contains(t, err.Error(), "Solution error")

	// Verify mock expectations
	mockDB.AssertExpectations(t)
}

func TestProposeSolution_FrameAmountOk_GenerateHashesOk_ExtractPublicKeyOk_GenerateMessageOk_SignMessageOk_SolutionOk(t *testing.T) {
	// Setup
	mockDB := new(mocks.DB)
	thread := &VideoRenderingThread{
		ThreadId:   "thread123",
		StartFrame: 0,
		EndFrame:   1,
	}
	cdc := codec.NewProtoCodec(types.NewInterfaceRegistry())

	// Mock DB methods
	mockDB.
		On("UpdateThread", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything,
			mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(nil).
		Once()

	mockDB.On("AddLogEntry", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()

	// Monkey patching
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

	// Verify that we got no error
	require.NoError(t, err)

	// Verify mock expectations
	mockDB.AssertExpectations(t)
}

// --- Test for SubmitVerification ---
func TestSubmitVerification_FileCountKo(t *testing.T) {
	// Setup
	mockDB := new(mocks.DB)
	thread := &VideoRenderingThread{
		ThreadId:   "thread123",
		StartFrame: 0,
		EndFrame:   1,
	}
	cdc := codec.NewProtoCodec(types.NewInterfaceRegistry())
	expected := false

	// Mock DB methods
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

	// Monkey patching
	patch1 := monkey.Patch(vm.CountFilesInDirectory, func(directoryPath string) int {
		return 0
	})
	defer patch1.Unpatch()

	err := thread.SubmitVerification(cdc, "worker-alias-001", "cosmos1abcdefg1234567", "/tmp/test-rendering", mockDB)

	// Verify that we got the expected thread status (file amount error)
	require.NoError(t, err)
	require.True(t, expected, "Expected thread status (file amount error)")

	// Verify mock expectations
	mockDB.AssertExpectations(t)
}

func TestSubmitVerification_FileCountOk_FileThresholdKo(t *testing.T) {
	// Setup
	mockDB := new(mocks.DB)
	thread := &VideoRenderingThread{
		ThreadId:   "thread123",
		StartFrame: 0,
		EndFrame:   10,
	}
	cdc := codec.NewProtoCodec(types.NewInterfaceRegistry())
	expected := false

	// Mock DB methods
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

	// Monkey patching
	patch1 := monkey.Patch(vm.CountFilesInDirectory, func(directoryPath string) int {
		return 1
	})
	defer patch1.Unpatch()

	err := thread.SubmitVerification(cdc, "worker-alias-001", "cosmos1abcdefg1234567", "/tmp/test-rendering", mockDB)

	// Verify that we got the expected error
	require.NoError(t, err)
	require.True(t, expected, "Expected thread status (file threshold error)")

	// Verify mock expectations
	mockDB.AssertExpectations(t)
}

func TestSubmitVerification_FileCountOk_FileThresholdOk_GenerateFileHashesKo(t *testing.T) {
	// Setup
	mockDB := new(mocks.DB)
	thread := &VideoRenderingThread{
		ThreadId:   "thread123",
		StartFrame: 0,
		EndFrame:   1,
	}
	cdc := codec.NewProtoCodec(types.NewInterfaceRegistry())

	// Mock DB methods
	mockDB.
		On("UpdateThread", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything,
			mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(nil).
		Twice()

	// Monkey patching
	patch1 := monkey.Patch(vm.CountFilesInDirectory, func(directoryPath string) int {
		return 1
	})
	defer patch1.Unpatch()

	patch2 := monkey.Patch(GenerateDirectoryFileHashes, func(directoryPath string) (map[string]string, error) {
		return nil, fmt.Errorf("Generate hash error")
	})
	defer patch2.Unpatch()

	err := thread.SubmitVerification(cdc, "worker-alias-001", "cosmos1abcdefg1234567", "/tmp/test-rendering", mockDB)

	// Verify that we got the expected error
	require.Error(t, err)
	require.Contains(t, err.Error(), "Generate hash error")

	// Verify mock expectations
	mockDB.AssertExpectations(t)
}

func TestSubmitVerification_FileCountOk_FileThresholdOk_GenerateFileHashesOk_GetPublicKeyKo(t *testing.T) {
	// Setup
	mockDB := new(mocks.DB)
	thread := &VideoRenderingThread{
		ThreadId:   "thread123",
		StartFrame: 0,
		EndFrame:   1,
	}
	cdc := codec.NewProtoCodec(types.NewInterfaceRegistry())

	// Mock DB methods
	mockDB.
		On("UpdateThread", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything,
			mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(nil).
		Twice()

	// Monkey patching
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

	// Verify that we got the expected error
	require.Error(t, err)
	require.Contains(t, err.Error(), "Get public key error")

	// Verify mock expectations
	mockDB.AssertExpectations(t)
}

func TestSubmitVerification_FileCountOk_FileThresholdOk_GenerateFileHashesOk_GetPublicKeyOk_GenerateSignableMessageKo(t *testing.T) {
	// Setup
	mockDB := new(mocks.DB)
	thread := &VideoRenderingThread{
		ThreadId:   "thread123",
		StartFrame: 0,
		EndFrame:   1,
	}
	cdc := codec.NewProtoCodec(types.NewInterfaceRegistry())

	// Mock DB methods
	mockDB.
		On("UpdateThread", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything,
			mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(nil).
		Twice()

	// Monkey patching
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
		return []byte("fake-signable-message"), fmt.Errorf("GenerateSignableMessage error")
	})
	defer patch4.Unpatch()

	err := thread.SubmitVerification(cdc, "worker-alias-001", "cosmos1abcdefg1234567", "/tmp/test-rendering", mockDB)

	// Verify that we got the expected error
	require.Error(t, err)
	require.Contains(t, err.Error(), "GenerateSignableMessage error")

	// Verify mock expectations
	mockDB.AssertExpectations(t)
}

func TestSubmitVerification_FileCountOk_FileThresholdOk_GenerateFileHashesOk_GetPublicKeyOk_GenerateSignableMessageOk_SignMessageKo(t *testing.T) {
	// Setup
	mockDB := new(mocks.DB)
	thread := &VideoRenderingThread{
		ThreadId:   "thread123",
		StartFrame: 0,
		EndFrame:   1,
	}
	cdc := codec.NewProtoCodec(types.NewInterfaceRegistry())

	// Mock DB methods
	mockDB.
		On("UpdateThread", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything,
			mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(nil).
		Twice()

	// Monkey patching
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

	// Verify that we got the expected error
	require.Error(t, err)
	require.Contains(t, err.Error(), "Signing message error")

	// Verify mock expectations
	mockDB.AssertExpectations(t)
}

func TestSubmitVerification_FileCountOk_FileThresholdOk_GenerateFileHashesOk_GetPublicKeyOk_GenerateSignableMessageOk_SignMessageOk_SubmitValidationKo(t *testing.T) {
	// Setup
	mockDB := new(mocks.DB)
	thread := &VideoRenderingThread{
		ThreadId:   "thread123",
		StartFrame: 0,
		EndFrame:   1,
	}
	cdc := codec.NewProtoCodec(types.NewInterfaceRegistry())

	// Mock DB methods
	mockDB.
		On("UpdateThread", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything,
			mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(nil).
		Twice()

	mockDB.On("AddLogEntry", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()

	// Monkey patching
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

	// Verify that we got the expected error
	require.Error(t, err)
	require.Contains(t, err.Error(), "Submit validation error")

	// Verify mock expectations
	mockDB.AssertExpectations(t)
}

func TestSubmitVerification_FileCountOk_FileThresholdOk_GenerateFileHashesOk_GetPublicKeyOk_GenerateSignableMessageOk_SignMessageOk_SubmitValidationOk(t *testing.T) {
	// Setup
	mockDB := new(mocks.DB)
	thread := &VideoRenderingThread{
		ThreadId:   "thread123",
		StartFrame: 0,
		EndFrame:   1,
	}
	cdc := codec.NewProtoCodec(types.NewInterfaceRegistry())

	// Mock DB methods
	mockDB.
		On("UpdateThread", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything,
			mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(nil).
		Once()

	mockDB.On("AddLogEntry", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()

	// Monkey patching
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

	// Verify that we got no error
	require.NoError(t, err)

	// Verify mock expectations
	mockDB.AssertExpectations(t)
}

// --- Test for submitValidation ---

// --- Test for SubmitSolution ---
func TestSubmitSolution_IpfsKo(t *testing.T) {
	// Setup
	mockDB := new(mocks.DB)
	thread := &VideoRenderingThread{
		ThreadId:   "thread123",
		StartFrame: 0,
		EndFrame:   1,
	}
	ctx := context.Background()
	workerAddress := "cosmos1abcd1234workerxyz"
	rootPath := "/tmp/rendering/thread123"

	// Mock DB methods
	mockDB.
		On("UpdateThread", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything,
			mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(nil).
		Twice()

	mockDB.On("AddLogEntry", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()

	// Monkey patching
	patch1 := monkey.Patch(ipfs.UploadSolution, func(ctx context.Context, rootPath, threadId string) (string, error) {
		return "", fmt.Errorf("Ipfs upload solution error")
	})
	defer patch1.Unpatch()

	err := thread.SubmitSolution(ctx, workerAddress, rootPath, mockDB)

	// Verify that we got the expected error
	require.Error(t, err)
	require.Contains(t, err.Error(), "Ipfs upload solution error")

	// Verify mock expectations
	mockDB.AssertExpectations(t)
}

func TestSubmitSolution_IpfsOk_SubmitSolutionKo(t *testing.T) {
	// Setup
	mockDB := new(mocks.DB)
	thread := &VideoRenderingThread{
		ThreadId:   "thread123",
		StartFrame: 0,
		EndFrame:   1,
	}
	ctx := context.Background()
	workerAddress := "cosmos1abcd1234workerxyz"
	rootPath := "/tmp/rendering/thread123"

	// Mock DB methods
	mockDB.
		On("UpdateThread", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything,
			mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(nil).
		Twice()

	mockDB.On("AddLogEntry", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()

	// Monkey patching
	patch1 := monkey.Patch(ipfs.UploadSolution, func(ctx context.Context, rootPath, threadId string) (string, error) {
		return "bafybeibwzifn3f6ld5n3nqsh2gsyw5vcnrbdfzq3e6q3yhdh6kuz3w5xku", nil
	})
	defer patch1.Unpatch()

	patch2 := monkey.Patch(submitSolution, func(address string, taskId string, threadId string, cid string) error {
		return fmt.Errorf("submit solution error")
	})
	defer patch2.Unpatch()

	err := thread.SubmitSolution(ctx, workerAddress, rootPath, mockDB)

	// Verify that we got the expected error
	require.Error(t, err)
	require.Contains(t, err.Error(), "submit solution error")

	// Verify mock expectations
	mockDB.AssertExpectations(t)
}

func TestSubmitSolution_IpfsOk_SubmitSolutionOk(t *testing.T) {
	// Setup
	mockDB := new(mocks.DB)
	thread := &VideoRenderingThread{
		ThreadId:   "thread123",
		StartFrame: 0,
		EndFrame:   1,
	}
	ctx := context.Background()
	workerAddress := "cosmos1abcd1234workerxyz"
	rootPath := "/tmp/rendering/thread123"

	// Mock DB methods
	mockDB.
		On("UpdateThread", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything,
			mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(nil).
		Once()

	mockDB.On("AddLogEntry", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()

	// Monkey patching
	patch1 := monkey.Patch(ipfs.UploadSolution, func(ctx context.Context, rootPath, threadId string) (string, error) {
		return "bafybeibwzifn3f6ld5n3nqsh2gsyw5vcnrbdfzq3e6q3yhdh6kuz3w5xku", nil
	})
	defer patch1.Unpatch()

	patch2 := monkey.Patch(submitSolution, func(address string, taskId string, threadId string, cid string) error {
		return nil
	})
	defer patch2.Unpatch()

	err := thread.SubmitSolution(ctx, workerAddress, rootPath, mockDB)

	// Verify that we got no error
	require.NoError(t, err)

	// Verify mock expectations
	mockDB.AssertExpectations(t)
}

// --- Test for submitSolution ---

// --- Test for IsReverse ---
func TestIsReverse(t *testing.T) {
	thread := VideoRenderingThread{
		Workers: []string{"alice", "bob", "carol", "dave"},
	}

	t.Run("worker in odd position returns true", func(t *testing.T) {
		require.True(t, thread.IsReverse("bob"))  // index 1
		require.True(t, thread.IsReverse("dave")) // index 3
	})

	t.Run("worker in even position returns false", func(t *testing.T) {
		require.False(t, thread.IsReverse("alice")) // index 0
		require.False(t, thread.IsReverse("carol")) // index 2
	})

	t.Run("worker not in list returns false", func(t *testing.T) {
		require.False(t, thread.IsReverse("eve"))
	})
}

// --- Test for GetValidatorReward ---
func TestGetValidatorReward(t *testing.T) {
	thread := &VideoRenderingThread{
		Validations: []*VideoRenderingThread_Validation{
			{
				Validator: "alice",
				Frames: []*VideoRenderingThread_Frame{
					{Filename: "frame_1.png", Signature: "sig_1", Cid: "cid_1", Hash: "hash_1", ValidCount: 1, InvalidCount: 0},
					{Filename: "frame_2.png", Signature: "sig_2", Cid: "cid_2", Hash: "hash_2", ValidCount: 1, InvalidCount: 0},
				},
			},
			{
				Validator: "bob",
				Frames: []*VideoRenderingThread_Frame{
					{Filename: "frame_3.png", Signature: "sig_3", Cid: "cid_3", Hash: "hash_3", ValidCount: 1, InvalidCount: 0},
					{Filename: "frame_4.png", Signature: "sig_4", Cid: "cid_4", Hash: "hash_4", ValidCount: 1, InvalidCount: 0},
					{Filename: "frame_5.png", Signature: "sig_5", Cid: "cid_5", Hash: "hash_5", ValidCount: 1, InvalidCount: 0},
					{Filename: "frame_6.png", Signature: "sig_6", Cid: "cid_6", Hash: "hash_6", ValidCount: 1, InvalidCount: 0},
				},
			},
		},
	}

	totalReward := c_types.NewCoin("token", sdkmath.NewInt(60)) // total reward to distribute

	t.Run("validator receives proportional reward", func(t *testing.T) {
		reward := thread.GetValidatorReward("bob", totalReward)
		require.Equal(t, "jct", reward.Denom)
		require.Equal(t, int64(40), reward.Amount.Int64()) // 4 of 6 frames => 4/6 of 60 = 40
	})

	t.Run("non-validator receives zero", func(t *testing.T) {
		reward := thread.GetValidatorReward("carol", totalReward)
		require.Equal(t, int64(0), reward.Amount.Int64())
	})
}

// --- Test for calculateValidatorPayment ---
func TestCalculateValidatorPayment(t *testing.T) {
	tests := []struct {
		name                 string
		filesValidated       int
		totalFilesValidated  int
		totalValidatorReward sdkmath.Int
		expected             sdkmath.Int
	}{
		{
			name:                 "normal calculation",
			filesValidated:       3,
			totalFilesValidated:  6,
			totalValidatorReward: sdkmath.NewInt(60),
			expected:             sdkmath.NewInt(30),
		},
		{
			name:                 "zero total files",
			filesValidated:       3,
			totalFilesValidated:  0,
			totalValidatorReward: sdkmath.NewInt(60),
			expected:             sdkmath.NewInt(0),
		},
		{
			name:                 "zero validated files",
			filesValidated:       0,
			totalFilesValidated:  6,
			totalValidatorReward: sdkmath.NewInt(60),
			expected:             sdkmath.NewInt(0),
		},
		{
			name:                 "equal files validated and total",
			filesValidated:       6,
			totalFilesValidated:  6,
			totalValidatorReward: sdkmath.NewInt(60),
			expected:             sdkmath.NewInt(60),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calculateValidatorPayment(tt.filesValidated, tt.totalFilesValidated, tt.totalValidatorReward)
			require.True(t, result.Equal(tt.expected), "Expected %s, got %s", tt.expected.String(), result.String())
		})
	}
}

// --- Test for RevealSolution ---
func TestRevealSolution_CalculateCIDsKo(t *testing.T) {
	// Setup
	mockDB := new(mocks.DB)
	thread := &VideoRenderingThread{
		ThreadId:   "thread123",
		StartFrame: 0,
		EndFrame:   1,
	}
	rootPath := "/tmp/rendering/thread123"

	// Monkey patching
	patch1 := monkey.Patch(ipfs.CalculateCIDs, func(dirPath string) (map[string]string, error) {
		return nil, fmt.Errorf("Calculate CIDs error")
	})
	defer patch1.Unpatch()

	err := thread.RevealSolution(rootPath, mockDB)

	// Verify that we got the expected error
	require.Error(t, err)
	require.Contains(t, err.Error(), "Calculate CIDs error")
}

func TestRevealSolution_CalculateCIDsOk_CalculateFileHashKo(t *testing.T) {
	// Setup
	mockDB := new(mocks.DB)
	thread := &VideoRenderingThread{
		ThreadId:   "thread123",
		StartFrame: 0,
		EndFrame:   1,
	}
	rootPath := "/tmp/rendering/thread123"

	// Monkey patching
	patch1 := monkey.Patch(ipfs.CalculateCIDs, func(dirPath string) (map[string]string, error) {
		return map[string]string{
			"frame1.png": "bafybeibwzifkxwq6oyp3dp3ewr2lsccfveq5r7oe3jq2l6efzdr4hw2kdi",
			"frame2.png": "bafybeia6zjsa6uhjqmtn4azj3k74sjn3wsb2elxek6nnvysxug4vqwhwqe",
		}, nil
	})
	defer patch1.Unpatch()

	patch2 := monkey.Patch(CalculateFileHash, func(filePath string) (string, error) {
		return "", fmt.Errorf("Calculate file hash error")
	})
	defer patch2.Unpatch()

	err := thread.RevealSolution(rootPath, mockDB)

	// Verify that we got the expected error
	require.Error(t, err)
	require.Contains(t, err.Error(), "Calculate file hash error")
}

func TestRevealSolution_CalculateCIDsOk_CalculateFileHashOk_ExecuteCliKo(t *testing.T) {
	// Setup
	mockDB := new(mocks.DB)
	thread := &VideoRenderingThread{
		ThreadId:   "thread123",
		StartFrame: 0,
		EndFrame:   1,
		Solution: &VideoRenderingThread_Solution{
			ProposedBy: "alice",
			PublicKey:  "alicePublicKey123",
			Dir:        "/tmp/rendered_frames/solution1",
			Accepted:   true,
			Frames: []*VideoRenderingThread_Frame{
				{
					Filename:     "frame_001.png",
					Signature:    "sig1",
					Cid:          "bafyframe001",
					Hash:         "abc123hash001",
					ValidCount:   3,
					InvalidCount: 0,
				},
				{
					Filename:     "frame_002.png",
					Signature:    "sig2",
					Cid:          "bafyframe002",
					Hash:         "abc123hash002",
					ValidCount:   2,
					InvalidCount: 1,
				},
			},
		}}
	rootPath := "/tmp/rendering/thread123"

	// Monkey patching
	patch1 := monkey.Patch(ipfs.CalculateCIDs, func(dirPath string) (map[string]string, error) {
		return map[string]string{
			"frame1.png": "bafybeibwzifkxwq6oyp3dp3ewr2lsccfveq5r7oe3jq2l6efzdr4hw2kdi",
			"frame2.png": "bafybeia6zjsa6uhjqmtn4azj3k74sjn3wsb2elxek6nnvysxug4vqwhwqe",
		}, nil
	})
	defer patch1.Unpatch()

	patch2 := monkey.Patch(CalculateFileHash, func(filePath string) (string, error) {
		return "6b1b36cbb04b41490bfc0ab2bfa26f86", nil
	})
	defer patch2.Unpatch()

	patch4 := monkey.Patch(ExecuteCli, func(args []string) error {
		return fmt.Errorf("FromFramesToCli error")
	})
	defer patch4.Unpatch()

	err := thread.RevealSolution(rootPath, mockDB)

	// Verify that we got the expected error
	require.Error(t, err)
	require.Contains(t, err.Error(), "FromFramesToCli error")
}

func TestRevealSolution_CalculateCIDsOk_CalculateFileHashOk_ExecuteCliOk_UpdateThreadKo(t *testing.T) {
	// Setup
	mockDB := new(mocks.DB)
	thread := &VideoRenderingThread{
		ThreadId:   "thread123",
		StartFrame: 0,
		EndFrame:   1,
		Solution: &VideoRenderingThread_Solution{
			ProposedBy: "alice",
			PublicKey:  "alicePublicKey123",
			Dir:        "/tmp/rendered_frames/solution1",
			Accepted:   true,
			Frames: []*VideoRenderingThread_Frame{
				{
					Filename:     "frame_001.png",
					Signature:    "sig1",
					Cid:          "bafyframe001",
					Hash:         "abc123hash001",
					ValidCount:   3,
					InvalidCount: 0,
				},
				{
					Filename:     "frame_002.png",
					Signature:    "sig2",
					Cid:          "bafyframe002",
					Hash:         "abc123hash002",
					ValidCount:   2,
					InvalidCount: 1,
				},
			},
		}}
	rootPath := "/tmp/rendering/thread123"

	// Mock DB methods
	mockDB.On("UpdateThread", "thread123", true, true, true, true, true, true, true, false).Return(fmt.Errorf("UpdateThread error")).Once()

	// Monkey patching
	patch1 := monkey.Patch(ipfs.CalculateCIDs, func(dirPath string) (map[string]string, error) {
		return map[string]string{
			"frame1.png": "bafybeibwzifkxwq6oyp3dp3ewr2lsccfveq5r7oe3jq2l6efzdr4hw2kdi",
			"frame2.png": "bafybeia6zjsa6uhjqmtn4azj3k74sjn3wsb2elxek6nnvysxug4vqwhwqe",
		}, nil
	})
	defer patch1.Unpatch()

	patch2 := monkey.Patch(CalculateFileHash, func(filePath string) (string, error) {
		return "6b1b36cbb04b41490bfc0ab2bfa26f86", nil
	})
	defer patch2.Unpatch()

	patch4 := monkey.Patch(ExecuteCli, func(args []string) error {
		return nil
	})
	defer patch4.Unpatch()

	err := thread.RevealSolution(rootPath, mockDB)

	// Verify that we got the expected error
	require.Error(t, err)
	require.Contains(t, err.Error(), "UpdateThread error")

	// Verify mock expectations
	mockDB.AssertExpectations(t)
}

func TestRevealSolution_CalculateCIDsOk_CalculateFileHashOk_ExecuteCliOk_UpdateThreadOk(t *testing.T) {
	// Setup
	mockDB := new(mocks.DB)
	thread := &VideoRenderingThread{
		ThreadId:   "thread123",
		StartFrame: 0,
		EndFrame:   1,
		Solution: &VideoRenderingThread_Solution{
			ProposedBy: "alice",
			PublicKey:  "alicePublicKey123",
			Dir:        "/tmp/rendered_frames/solution1",
			Accepted:   true,
			Frames: []*VideoRenderingThread_Frame{
				{
					Filename:     "frame_001.png",
					Signature:    "sig1",
					Cid:          "bafyframe001",
					Hash:         "abc123hash001",
					ValidCount:   3,
					InvalidCount: 0,
				},
				{
					Filename:     "frame_002.png",
					Signature:    "sig2",
					Cid:          "bafyframe002",
					Hash:         "abc123hash002",
					ValidCount:   2,
					InvalidCount: 1,
				},
			},
		}}
	rootPath := "/tmp/rendering/thread123"

	// Mock DB methods
	mockDB.On("UpdateThread", "thread123", true, true, true, true, true, true, true, false).Return(nil).Once()

	// Monkey patching
	patch1 := monkey.Patch(ipfs.CalculateCIDs, func(dirPath string) (map[string]string, error) {
		return map[string]string{
			"frame1.png": "bafybeibwzifkxwq6oyp3dp3ewr2lsccfveq5r7oe3jq2l6efzdr4hw2kdi",
			"frame2.png": "bafybeia6zjsa6uhjqmtn4azj3k74sjn3wsb2elxek6nnvysxug4vqwhwqe",
		}, nil
	})
	defer patch1.Unpatch()

	patch2 := monkey.Patch(CalculateFileHash, func(filePath string) (string, error) {
		return "6b1b36cbb04b41490bfc0ab2bfa26f86", nil
	})
	defer patch2.Unpatch()

	patch4 := monkey.Patch(ExecuteCli, func(args []string) error {
		return nil
	})
	defer patch4.Unpatch()

	err := thread.RevealSolution(rootPath, mockDB)

	// Verify that we got no error
	require.NoError(t, err)

	// Verify mock expectations
	mockDB.AssertExpectations(t)
}

// --- Test for EvaluateVerifications ---
func TestEvaluateVerifications_DecodePublicKeyFromCLIKo(t *testing.T) {
	// Setup
	thread := &VideoRenderingThread{
		ThreadId:   "thread123",
		StartFrame: 0,
		EndFrame:   1,
		Solution: &VideoRenderingThread_Solution{
			ProposedBy: "alice",
			PublicKey:  "alicePublicKey123",
			Dir:        "/tmp/rendered_frames/solution1",
			Accepted:   true,
			Frames: []*VideoRenderingThread_Frame{
				{
					Filename:     "frame1.png",
					Signature:    "sig1",
					Cid:          "cid1",
					Hash:         "hash1",
					ValidCount:   1,
					InvalidCount: 0,
				},
				{
					Filename:     "frame2.png",
					Signature:    "sig2",
					Cid:          "cid2",
					Hash:         "hash2",
					ValidCount:   2,
					InvalidCount: 0,
				},
			},
		},
		Validations: []*VideoRenderingThread_Validation{
			{
				Validator: "alice",
				Frames: []*VideoRenderingThread_Frame{
					{
						Filename:     "frame1.png",
						Signature:    "sig1",
						Cid:          "cid1",
						Hash:         "hash1",
						ValidCount:   1,
						InvalidCount: 0,
					},
					{
						Filename:     "frame2.png",
						Signature:    "sig2",
						Cid:          "cid2",
						Hash:         "hash2",
						ValidCount:   2,
						InvalidCount: 0,
					},
				},
				PublicKey: "pubkey-alice",
				IsReverse: false,
			},
			{
				Validator: "bob",
				Frames: []*VideoRenderingThread_Frame{
					{
						Filename:     "frame1.png",
						Signature:    "sig1",
						Cid:          "cid1",
						Hash:         "hash1",
						ValidCount:   0,
						InvalidCount: 1,
					},
					{
						Filename:     "frame2.png",
						Signature:    "sig2",
						Cid:          "cid2",
						Hash:         "hash2",
						ValidCount:   3,
						InvalidCount: 0,
					},
				},
				PublicKey: "pubkey-bob",
				IsReverse: true,
			},
		},
	}

	// Monkey patching
	patch1 := monkey.Patch(videoRenderingCrypto.DecodePublicKeyFromCLI, func(encodedPubKey string) (cryptotypes.PubKey, error) {
		return secp256k1.GenPrivKey().PubKey(), fmt.Errorf("DecodePublicKeyFromCLI error")
	})
	defer patch1.Unpatch()

	err := thread.EvaluateVerifications()

	// Verify that we got the expected error
	require.Error(t, err)
	require.Contains(t, err.Error(), "DecodePublicKeyFromCLI error")
}
func TestEvaluateVerifications_DecodePublicKeyFromCLIOk_GenerateSignableMessageKo(t *testing.T) {
	// Setup
	thread := &VideoRenderingThread{
		ThreadId:   "thread123",
		StartFrame: 0,
		EndFrame:   1,
		Solution: &VideoRenderingThread_Solution{
			ProposedBy: "alice",
			PublicKey:  "alicePublicKey123",
			Dir:        "/tmp/rendered_frames/solution1",
			Accepted:   true,
			Frames: []*VideoRenderingThread_Frame{
				{
					Filename:     "frame1.png",
					Signature:    "sig1",
					Cid:          "cid1",
					Hash:         "hash1",
					ValidCount:   1,
					InvalidCount: 0,
				},
				{
					Filename:     "frame2.png",
					Signature:    "sig2",
					Cid:          "cid2",
					Hash:         "hash2",
					ValidCount:   2,
					InvalidCount: 0,
				},
			},
		},
		Validations: []*VideoRenderingThread_Validation{
			{
				Validator: "alice",
				Frames: []*VideoRenderingThread_Frame{
					{
						Filename:     "frame1.png",
						Signature:    "sig1",
						Cid:          "cid1",
						Hash:         "hash1",
						ValidCount:   1,
						InvalidCount: 0,
					},
					{
						Filename:     "frame2.png",
						Signature:    "sig2",
						Cid:          "cid2",
						Hash:         "hash2",
						ValidCount:   2,
						InvalidCount: 0,
					},
				},
				PublicKey: "pubkey-alice",
				IsReverse: false,
			},
			{
				Validator: "bob",
				Frames: []*VideoRenderingThread_Frame{
					{
						Filename:     "frame1.png",
						Signature:    "sig1",
						Cid:          "cid1",
						Hash:         "hash1",
						ValidCount:   0,
						InvalidCount: 1,
					},
					{
						Filename:     "frame2.png",
						Signature:    "sig2",
						Cid:          "cid2",
						Hash:         "hash2",
						ValidCount:   3,
						InvalidCount: 0,
					},
				},
				PublicKey: "pubkey-bob",
				IsReverse: true,
			},
		},
	}

	// Monkey patching
	patch1 := monkey.Patch(videoRenderingCrypto.DecodePublicKeyFromCLI, func(encodedPubKey string) (cryptotypes.PubKey, error) {
		return secp256k1.GenPrivKey().PubKey(), nil
	})
	defer patch1.Unpatch()

	patch2 := monkey.Patch(videoRenderingCrypto.GenerateSignableMessage, func(hash, workerAddr string) ([]byte, error) {
		return nil, fmt.Errorf("GenerateSignableMessage error")
	})
	defer patch2.Unpatch()

	err := thread.EvaluateVerifications()

	// Verify that we got the expected error
	require.Error(t, err)
	require.Contains(t, err.Error(), "GenerateSignableMessage error")
}
func TestEvaluateVerifications_DecodePublicKeyFromCLIOk_GenerateSignableMessageOk_DecodeSignatureFromCLIKo(t *testing.T) {
	// Setup
	thread := &VideoRenderingThread{
		ThreadId:   "thread123",
		StartFrame: 0,
		EndFrame:   1,
		Solution: &VideoRenderingThread_Solution{
			ProposedBy: "alice",
			PublicKey:  "alicePublicKey123",
			Dir:        "/tmp/rendered_frames/solution1",
			Accepted:   true,
			Frames: []*VideoRenderingThread_Frame{
				{
					Filename:     "frame1.png",
					Signature:    "sig1",
					Cid:          "cid1",
					Hash:         "hash1",
					ValidCount:   1,
					InvalidCount: 0,
				},
				{
					Filename:     "frame2.png",
					Signature:    "sig2",
					Cid:          "cid2",
					Hash:         "hash2",
					ValidCount:   2,
					InvalidCount: 0,
				},
			},
		},
		Validations: []*VideoRenderingThread_Validation{
			{
				Validator: "alice",
				Frames: []*VideoRenderingThread_Frame{
					{
						Filename:     "frame1.png",
						Signature:    "sig1",
						Cid:          "cid1",
						Hash:         "hash1",
						ValidCount:   1,
						InvalidCount: 0,
					},
					{
						Filename:     "frame2.png",
						Signature:    "sig2",
						Cid:          "cid2",
						Hash:         "hash2",
						ValidCount:   2,
						InvalidCount: 0,
					},
				},
				PublicKey: "pubkey-alice",
				IsReverse: false,
			},
			{
				Validator: "bob",
				Frames: []*VideoRenderingThread_Frame{
					{
						Filename:     "frame1.png",
						Signature:    "sig1",
						Cid:          "cid1",
						Hash:         "hash1",
						ValidCount:   0,
						InvalidCount: 1,
					},
					{
						Filename:     "frame2.png",
						Signature:    "sig2",
						Cid:          "cid2",
						Hash:         "hash2",
						ValidCount:   3,
						InvalidCount: 0,
					},
				},
				PublicKey: "pubkey-bob",
				IsReverse: true,
			},
		},
	}

	// Monkey patching
	patch1 := monkey.Patch(videoRenderingCrypto.DecodePublicKeyFromCLI, func(encodedPubKey string) (cryptotypes.PubKey, error) {
		return secp256k1.GenPrivKey().PubKey(), nil
	})
	defer patch1.Unpatch()

	patch2 := monkey.Patch(videoRenderingCrypto.GenerateSignableMessage, func(hash, workerAddr string) ([]byte, error) {
		return []byte("fake-signable-message"), nil
	})
	defer patch2.Unpatch()

	patch3 := monkey.Patch(videoRenderingCrypto.DecodeSignatureFromCLI, func(encodedSig string) ([]byte, error) {
		return nil, fmt.Errorf("DecodeSignatureFromCLI error")
	})
	defer patch3.Unpatch()

	err := thread.EvaluateVerifications()

	// Verify that we got the expected error
	require.Error(t, err)
	require.Contains(t, err.Error(), "DecodeSignatureFromCLI error")
}
func TestEvaluateVerifications_DecodePublicKeyFromCLIOk_GenerateSignableMessageOk_DecodeSignatureFromCLIOk_VerifySignatureFalse(t *testing.T) {
	// Setup
	thread := &VideoRenderingThread{
		ThreadId:   "thread123",
		StartFrame: 0,
		EndFrame:   1,
		Solution: &VideoRenderingThread_Solution{
			ProposedBy: "alice",
			PublicKey:  "alicePublicKey123",
			Dir:        "/tmp/rendered_frames/solution1",
			Accepted:   true,
			Frames: []*VideoRenderingThread_Frame{
				{
					Filename:     "frame1.png",
					Signature:    "sig1",
					Cid:          "cid1",
					Hash:         "hash1",
					ValidCount:   1,
					InvalidCount: 0,
				},
				{
					Filename:     "frame2.png",
					Signature:    "sig2",
					Cid:          "cid2",
					Hash:         "hash2",
					ValidCount:   2,
					InvalidCount: 0,
				},
			},
		},
		Validations: []*VideoRenderingThread_Validation{
			{
				Validator: "alice",
				Frames: []*VideoRenderingThread_Frame{
					{
						Filename:     "frame1.png",
						Signature:    "sig1",
						Cid:          "cid1",
						Hash:         "hash1",
						ValidCount:   1,
						InvalidCount: 0,
					},
					{
						Filename:     "frame2.png",
						Signature:    "sig2",
						Cid:          "cid2",
						Hash:         "hash2",
						ValidCount:   2,
						InvalidCount: 0,
					},
				},
				PublicKey: "pubkey-alice",
				IsReverse: false,
			},
			{
				Validator: "bob",
				Frames: []*VideoRenderingThread_Frame{
					{
						Filename:     "frame1.png",
						Signature:    "sig1",
						Cid:          "cid1",
						Hash:         "hash1",
						ValidCount:   0,
						InvalidCount: 1,
					},
					{
						Filename:     "frame2.png",
						Signature:    "sig2",
						Cid:          "cid2",
						Hash:         "hash2",
						ValidCount:   3,
						InvalidCount: 0,
					},
				},
				PublicKey: "pubkey-bob",
				IsReverse: true,
			},
		},
	}
	mockPublicKey := new(mocks.MockPubKey)

	// Mock public key methods
	mockPublicKey.On("VerifySignature", mock.Anything, mock.Anything).Return(false).Times(4) // Called 4 times as there is 4 frames in total

	// Monkey patching
	patch1 := monkey.Patch(videoRenderingCrypto.DecodePublicKeyFromCLI, func(encodedPubKey string) (cryptotypes.PubKey, error) {
		return mockPublicKey, nil
	})
	defer patch1.Unpatch()

	patch2 := monkey.Patch(videoRenderingCrypto.GenerateSignableMessage, func(hash, workerAddr string) ([]byte, error) {
		return []byte("fake-signable-message"), nil
	})
	defer patch2.Unpatch()

	patch3 := monkey.Patch(videoRenderingCrypto.DecodeSignatureFromCLI, func(encodedSig string) ([]byte, error) {
		return []byte{0x12, 0x34, 0xab, 0xcd, 0xef}, nil
	})
	defer patch3.Unpatch()

	err := thread.EvaluateVerifications()

	// Verify that we got no error
	require.NoError(t, err)
}
func TestEvaluateVerifications_DecodePublicKeyFromCLIOk_GenerateSignableMessageOk_DecodeSignatureFromCLIOk_VerifySignatureTrue(t *testing.T) {
	// Setup
	thread := &VideoRenderingThread{
		ThreadId:   "thread123",
		StartFrame: 0,
		EndFrame:   1,
		Solution: &VideoRenderingThread_Solution{
			ProposedBy: "alice",
			PublicKey:  "alicePublicKey123",
			Dir:        "/tmp/rendered_frames/solution1",
			Accepted:   true,
			Frames: []*VideoRenderingThread_Frame{
				{
					Filename:     "frame1.png",
					Signature:    "sig1",
					Cid:          "cid1",
					Hash:         "hash1",
					ValidCount:   1,
					InvalidCount: 0,
				},
				{
					Filename:     "frame2.png",
					Signature:    "sig2",
					Cid:          "cid2",
					Hash:         "hash2",
					ValidCount:   2,
					InvalidCount: 0,
				},
			},
		},
		Validations: []*VideoRenderingThread_Validation{
			{
				Validator: "alice",
				Frames: []*VideoRenderingThread_Frame{
					{
						Filename:     "frame1.png",
						Signature:    "sig1",
						Cid:          "cid1",
						Hash:         "hash1",
						ValidCount:   1,
						InvalidCount: 0,
					},
					{
						Filename:     "frame2.png",
						Signature:    "sig2",
						Cid:          "cid2",
						Hash:         "hash2",
						ValidCount:   2,
						InvalidCount: 0,
					},
				},
				PublicKey: "pubkey-alice",
				IsReverse: false,
			},
			{
				Validator: "bob",
				Frames: []*VideoRenderingThread_Frame{
					{
						Filename:     "frame1.png",
						Signature:    "sig1",
						Cid:          "cid1",
						Hash:         "hash1",
						ValidCount:   0,
						InvalidCount: 1,
					},
					{
						Filename:     "frame2.png",
						Signature:    "sig2",
						Cid:          "cid2",
						Hash:         "hash2",
						ValidCount:   3,
						InvalidCount: 0,
					},
				},
				PublicKey: "pubkey-bob",
				IsReverse: true,
			},
		},
	}
	mockPublicKey := new(mocks.MockPubKey)

	// Mock public key methods
	mockPublicKey.On("VerifySignature", mock.Anything, mock.Anything).Return(true).Times(4) // Called 4 times as there is 4 frames in total

	// Monkey patching
	patch1 := monkey.Patch(videoRenderingCrypto.DecodePublicKeyFromCLI, func(encodedPubKey string) (cryptotypes.PubKey, error) {
		return mockPublicKey, nil
	})
	defer patch1.Unpatch()

	patch2 := monkey.Patch(videoRenderingCrypto.GenerateSignableMessage, func(hash, workerAddr string) ([]byte, error) {
		return []byte("fake-signable-message"), nil
	})
	defer patch2.Unpatch()

	patch3 := monkey.Patch(videoRenderingCrypto.DecodeSignatureFromCLI, func(encodedSig string) ([]byte, error) {
		return []byte{0x12, 0x34, 0xab, 0xcd, 0xef}, nil
	})
	defer patch3.Unpatch()

	err := thread.EvaluateVerifications()

	// Verify that we got no error
	require.NoError(t, err)
}

func TestEvaluateVerifications_DecodePublicKeyFromCLIOk_GenerateSignableMessageOk_DecodeSignatureFromCLIOk_VerifySignatureTrue_FrameNotFound(t *testing.T) {
	// Setup
	thread := &VideoRenderingThread{
		ThreadId:   "thread123",
		StartFrame: 0,
		EndFrame:   1,
		Solution: &VideoRenderingThread_Solution{
			ProposedBy: "alice",
			PublicKey:  "alicePublicKey123",
			Dir:        "/tmp/rendered_frames/solution1",
			Accepted:   true,
			Frames: []*VideoRenderingThread_Frame{
				{
					Filename:     "frame1.png",
					Signature:    "sig1",
					Cid:          "cid1",
					Hash:         "hash1",
					ValidCount:   1,
					InvalidCount: 0,
				},
				{
					Filename:     "frame2.png",
					Signature:    "sig2",
					Cid:          "cid2",
					Hash:         "hash2",
					ValidCount:   2,
					InvalidCount: 0,
				},
			},
		},
		Validations: []*VideoRenderingThread_Validation{
			{
				Validator: "alice",
				Frames: []*VideoRenderingThread_Frame{
					{
						Filename:     "frame1.png",
						Signature:    "sig1",
						Cid:          "cid1",
						Hash:         "hash1",
						ValidCount:   1,
						InvalidCount: 0,
					},
					{
						Filename:     "frame2.png",
						Signature:    "sig2",
						Cid:          "cid2",
						Hash:         "hash2",
						ValidCount:   2,
						InvalidCount: 0,
					},
				},
				PublicKey: "pubkey-alice",
				IsReverse: false,
			},
			{
				Validator: "bob",
				Frames: []*VideoRenderingThread_Frame{
					{
						Filename:     "frame1.png",
						Signature:    "sig1",
						Cid:          "cid1",
						Hash:         "hash1",
						ValidCount:   0,
						InvalidCount: 1,
					},
					// "frame2.png" eliminated
				},
				PublicKey: "pubkey-bob",
				IsReverse: true,
			},
		},
	}
	mockPublicKey := new(mocks.MockPubKey)

	// Mock public key methods
	mockPublicKey.On("VerifySignature", mock.Anything, mock.Anything).Return(true).Times(4) // Called 4 times as there is 4 frames in total

	// Monkey patching
	patch1 := monkey.Patch(videoRenderingCrypto.DecodePublicKeyFromCLI, func(encodedPubKey string) (cryptotypes.PubKey, error) {
		return mockPublicKey, nil
	})
	defer patch1.Unpatch()

	patch2 := monkey.Patch(videoRenderingCrypto.GenerateSignableMessage, func(hash, workerAddr string) ([]byte, error) {
		return []byte("fake-signable-message"), nil
	})
	defer patch2.Unpatch()

	patch3 := monkey.Patch(videoRenderingCrypto.DecodeSignatureFromCLI, func(encodedSig string) ([]byte, error) {
		return []byte{0x12, 0x34, 0xab, 0xcd, 0xef}, nil
	})
	defer patch3.Unpatch()

	err := thread.EvaluateVerifications()

	// Verify that we got no error
	require.NoError(t, err)
}

// --- Test for IsSolutionAccepted ---
// --- Test for VerifySubmittedSolution ---
