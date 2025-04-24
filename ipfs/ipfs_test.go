package ipfs

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"bou.ke/monkey"
	shell "github.com/ipfs/go-ipfs-api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIPFSGet(t *testing.T) {
	// Patch shell.Shell.Get to avoid real IPFS interaction
	patch := monkey.Patch((*shell.Shell).Get, func(_ *shell.Shell, cid string, outDir string) error {
		// Simulate success only for the "validCID"
		if cid == "validCID" {
			return nil
		}
		// Simulate failure for any other CID
		return errors.New("Mock error")
	})
	defer patch.Unpatch()

	path := "/tmp/fakepath"

	// Define table-driven test cases
	tests := []struct {
		name            string
		cid             string
		expectedToError bool
	}{
		{
			name:            "valid",
			cid:             "validCID",
			expectedToError: false,
		},
		{
			name:            "invalid",
			cid:             "invalidCID",
			expectedToError: true,
		},
	}

	// Run each test case in a subtest
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := IPFSGet(tt.cid, path)
			if (err != nil) != tt.expectedToError {
				t.Errorf("Test %q failed: expectedToError=%v, got err=%v", tt.name, tt.expectedToError, err)
			}
		})
	}
}

func createTempFiles(t *testing.T) (string, map[string]string) {
	dir := t.TempDir()
	files := map[string]string{
		"file1.txt": "file1 content",
		"file2.txt": "file2 content",
	}
	for name, content := range files {
		err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644)
		assert.NoError(t, err)
	}
	return dir, files
}

func fakeExecCommand(output string) *exec.Cmd {
	return &exec.Cmd{
		Path:   "/bin/echo",
		Args:   []string{"echo", output},
		Stdout: &bytes.Buffer{},
		Stderr: &bytes.Buffer{},
	}
}

func fakeExecCommandWithError(stderr string) *exec.Cmd {
	cmd := exec.Command("false")
	var out bytes.Buffer
	out.WriteString(stderr)
	cmd.Stdout = &out
	cmd.Stderr = &out
	return cmd
}

func TestCalculateCIDs(t *testing.T) {
	// Create temporary files and expected contents for testing
	dir, fileContents := createTempFiles(t)

	// Patch exec.Command to simulate the behavior of the IPFS command
	patch := monkey.Patch(exec.Command, func(name string, args ...string) *exec.Cmd {
		// Get the file path from the last argument
		path := args[len(args)-1]

		// Read the file to simulate CID calculation
		content, err := os.ReadFile(path)
		if err != nil {
			// Simulate command failure if the file doesn't exist
			return fakeExecCommandWithError("file not found")
		}

		// Simulate CID by hashing the content and truncating the result
		hash := sha256.Sum256(content)
		cid := hex.EncodeToString(hash[:])[:10]
		return fakeExecCommand(cid)
	})
	defer patch.Unpatch()

	// Run the function under test
	cidMap, err := CalculateCIDs(dir)
	assert.NoError(t, err)

	// Compare the expected CIDs (from hashes) with the result
	for name, content := range fileContents {
		hash := sha256.Sum256([]byte(content))
		expectedCID := hex.EncodeToString(hash[:])[:10]
		assert.Equal(t, expectedCID, cidMap[name])
	}
}

func TestUploadSolution(t *testing.T) {
	dir := t.TempDir()
	threadId := "thread123"
	threadOutputPath := filepath.Join(dir, "renders", threadId, "output")

	// Create output directory and a dummy file
	err := os.MkdirAll(threadOutputPath, 0755)
	require.NoError(t, err)

	err = os.WriteFile(filepath.Join(threadOutputPath, "result.png"), []byte("fake image content"), 0644)
	require.NoError(t, err)

	// Patch the AddDir method to simulate a successful CID return
	patch := monkey.Patch((*shell.Shell).AddDir, func(s *shell.Shell, path string, opts ...func(*shell.RequestBuilder) error) (string, error) {
		if strings.Contains(path, "output") {
			return "QmFakeCID123", nil
		}
		return "", errors.New("unexpected path")
	})
	defer patch.Unpatch()

	cid, err := UploadSolution(context.Background(), dir, threadId)
	assert.NoError(t, err)
	assert.Equal(t, "QmFakeCID123", cid)
}

func TestCheckIPFSStatus(t *testing.T) {
	// "Happy Path" subtest: simulate a 200 OK response by patching RoundTrip.
	t.Run("happy path", func(t *testing.T) {
		// Patch the RoundTrip method of the default transport.
		patch := monkey.PatchInstanceMethod(
			// http.DefaultTransport is an interface; its underlying value is *http.Transport.
			reflect.TypeOf(http.DefaultTransport.(*http.Transport)),
			"RoundTrip",
			func(rt *http.Transport, req *http.Request) (*http.Response, error) {
				// Check that this is our target request.
				if req.Method == "POST" && req.URL.String() == "http://localhost:5001/api/v0/id" {
					return &http.Response{
						StatusCode: http.StatusOK,
						// Create a ReadCloser from a buffer using io.NopCloser.
						Body:    io.NopCloser(bytes.NewBufferString(`{"ID": "mockID"}`)),
						Header:  make(http.Header),
						Request: req,
					}, nil
				}
				return nil, fmt.Errorf("unexpected request")
			},
		)
		defer patch.Unpatch()

		err := CheckIPFSStatus()
		assert.NoError(t, err)
	})

	// "Error" subtest: simulate a 400 Bad Request response.
	t.Run("bad request", func(t *testing.T) {
		patch := monkey.PatchInstanceMethod(
			reflect.TypeOf(http.DefaultTransport.(*http.Transport)),
			"RoundTrip",
			func(rt *http.Transport, req *http.Request) (*http.Response, error) {
				if req.Method == "POST" && req.URL.String() == "http://localhost:5001/api/v0/id" {
					return &http.Response{
						StatusCode: http.StatusBadRequest,
						Body:       io.NopCloser(bytes.NewBufferString("Bad Request")),
						Header:     make(http.Header),
						Request:    req,
					}, nil
				}
				return nil, fmt.Errorf("unexpected request")
			},
		)
		defer patch.Unpatch()

		err := CheckIPFSStatus()
		assert.Error(t, err)
	})
}

func TestStartIPFS(t *testing.T) {
	// Happy path: simulate successful start (Start returns nil)
	t.Run("happy path", func(t *testing.T) {
		// Patch the Start method of *exec.Cmd to always return nil (success)
		patch := monkey.PatchInstanceMethod(reflect.TypeOf(&exec.Cmd{}), "Start", func(cmd *exec.Cmd) error {
			return nil
		})
		defer patch.Unpatch()

		err := StartIPFS()
		assert.NoError(t, err)
	})

	// Negative path: simulate failure (Start returns an error)
	t.Run("failure", func(t *testing.T) {
		// Patch the Start method of *exec.Cmd to simulate an error when starting the daemon
		patch := monkey.PatchInstanceMethod(reflect.TypeOf(&exec.Cmd{}), "Start", func(cmd *exec.Cmd) error {
			return fmt.Errorf("mock start error")
		})
		defer patch.Unpatch()

		err := StartIPFS()
		assert.Error(t, err)
	})
}

func TestEnsureIPFSRunning(t *testing.T) {
	// Test case: IPFS is already running
	t.Run("IPFS is already running", func(t *testing.T) {
		// Mock CheckIPFSStatus to simulate a running IPFS daemon by returning nil
		patchCheck := monkey.Patch(CheckIPFSStatus, func() error {
			return nil
		})
		defer patchCheck.Unpatch()

		// Mock StartIPFS to detect if it's called when it shouldn't be
		patchStart := monkey.Patch(StartIPFS, func() error {
			t.Error("StartIPFS should not be called when IPFS is already running")
			return nil
		})
		defer patchStart.Unpatch()

		EnsureIPFSRunning()
	})

	// Test case: IPFS is not running, and StartIPFS succeeds
	t.Run("IPFS not running, StartIPFS succeeds", func(t *testing.T) {
		checkCalls := 0
		startCalls := 0

		// Mock CheckIPFSStatus to simulate a stopped IPFS daemon by returning an error
		patchCheck := monkey.Patch(CheckIPFSStatus, func() error {
			checkCalls++
			return fmt.Errorf("IPFS is not running")
		})
		defer patchCheck.Unpatch()

		// Mock StartIPFS to simulate successful start by returning nil
		patchStart := monkey.Patch(StartIPFS, func() error {
			startCalls++
			return nil
		})
		defer patchStart.Unpatch()

		EnsureIPFSRunning()

		// Verify that CheckIPFSStatus was called once
		if checkCalls != 1 {
			t.Errorf("Expected CheckIPFSStatus to be called once, but it was called %d times", checkCalls)
		}
		// Verify that StartIPFS was called once
		if startCalls != 1 {
			t.Errorf("Expected StartIPFS to be called once, but it was called %d times", startCalls)
		}
	})

	// Test case: IPFS is not running, and StartIPFS fails
	t.Run("IPFS not running, StartIPFS fails", func(t *testing.T) {
		checkCalls := 0
		startCalls := 0

		// Mock CheckIPFSStatus to simulate a stopped IPFS daemon by returning an error
		patchCheck := monkey.Patch(CheckIPFSStatus, func() error {
			checkCalls++
			return fmt.Errorf("IPFS is not running")
		})
		defer patchCheck.Unpatch()

		// Mock StartIPFS to simulate a failure in starting by returning an error
		patchStart := monkey.Patch(StartIPFS, func() error {
			startCalls++
			return fmt.Errorf("Failed to start IPFS")
		})
		defer patchStart.Unpatch()

		EnsureIPFSRunning()

		// Verify that CheckIPFSStatus was called once
		if checkCalls != 1 {
			t.Errorf("Expected CheckIPFSStatus to be called once, but it was called %d times", checkCalls)
		}
		// Verify that StartIPFS was called once
		if startCalls != 1 {
			t.Errorf("Expected StartIPFS to be called once, but it was called %d times", startCalls)
		}
	})
}

func TestListDirectory(t *testing.T) {
}

func TestConnectToIPFSNode(t *testing.T) {
	// Happy path: simulate successful connection (CombinedOutput returns nil and no error)
	t.Run("happy path", func(t *testing.T) {
		// Patch the CombinedOutput method of *exec.Cmd to simulate a successful connection
		patch := monkey.PatchInstanceMethod(reflect.TypeOf(&exec.Cmd{}), "CombinedOutput", func(cmd *exec.Cmd) ([]byte, error) {
			return []byte("connected successfully"), nil
		})
		defer patch.Unpatch()

		ConnectToIPFSNode("127.0.0.1", "peerId123")
	})

	// Negative path: simulate connection failure (CombinedOutput returns an error)
	t.Run("failure", func(t *testing.T) {
		// Patch the CombinedOutput method of *exec.Cmd to simulate an error when connecting
		patch := monkey.PatchInstanceMethod(reflect.TypeOf(&exec.Cmd{}), "CombinedOutput", func(cmd *exec.Cmd) ([]byte, error) {
			return nil, fmt.Errorf("mock connection error")
		})
		defer patch.Unpatch()

		ConnectToIPFSNode("127.0.0.1", "peerId123")
	})
}

func TestGetIPFSPeerID(t *testing.T) {
	// Happy path: simulate successful execution of `ipfs id` command
	t.Run("happy path", func(t *testing.T) {
		// Patch the Output method of *exec.Cmd to simulate a successful output
		patch := monkey.PatchInstanceMethod(reflect.TypeOf(&exec.Cmd{}), "Output", func(cmd *exec.Cmd) ([]byte, error) {
			// Simulate a successful response from `ipfs id` (json format)
			return []byte(`{"ID": "QmExamplePeerID"}`), nil
		})
		defer patch.Unpatch()

		peerID, err := GetIPFSPeerID()

		// Assert no error occurred and peerID is the expected one
		assert.NoError(t, err)
		assert.Equal(t, "QmExamplePeerID", peerID)
	})

	// Negative path: simulate failure of `ipfs id` command
	t.Run("failure", func(t *testing.T) {
		// Patch the Output method of *exec.Cmd to simulate a failure (error from the command)
		patch := monkey.PatchInstanceMethod(reflect.TypeOf(&exec.Cmd{}), "Output", func(cmd *exec.Cmd) ([]byte, error) {
			// Simulate an error (command fails)
			return nil, fmt.Errorf("mock ipfs id error")
		})
		defer patch.Unpatch()

		peerID, err := GetIPFSPeerID()

		// Assert an error occurred and peerID should be empty
		assert.Error(t, err)
		assert.Empty(t, peerID)
	})

	// Negative path: simulate failure in parsing the JSON output
	t.Run("json parsing failure", func(t *testing.T) {
		// Patch the Output method of *exec.Cmd to simulate a successful output, but malformed JSON
		patch := monkey.PatchInstanceMethod(reflect.TypeOf(&exec.Cmd{}), "Output", func(cmd *exec.Cmd) ([]byte, error) {
			// Return malformed JSON (missing closing brace)
			return []byte(`{"ID": "QmExamplePeerID"`), nil
		})
		defer patch.Unpatch()

		peerID, err := GetIPFSPeerID()

		// Assert that an error occurred and peerID should be empty
		assert.Error(t, err)
		assert.Empty(t, peerID)
	})
}

func TestGenerateSwarmConnectURL(t *testing.T) {
	tests := []struct {
		ip       string
		peerID   string
		expected string
	}{
		{
			ip:       "192.168.1.1",
			peerID:   "QmTestPeerID123",
			expected: "/ip4/192.168.1.1/tcp/4001/p2p/QmTestPeerID123",
		},
		{
			ip:       "10.0.0.2",
			peerID:   "AnotherPeerID456",
			expected: "/ip4/10.0.0.2/tcp/4001/p2p/AnotherPeerID456",
		},
	}

	for _, tt := range tests {
		result, err := GenerateSwarmConnectURL(tt.ip, tt.peerID)
		if err != nil {
			t.Fatalf("Expected no error, but got: %v", err)
		}
		if result != tt.expected {
			t.Errorf("Expected %s, but got %s", tt.expected, result)
		}
	}
}

func TestIsDownloadStarted(t *testing.T) {
	tests := []struct {
		dir      string
		setup    func() // Function to create the directory and its content
		expected bool
	}{
		{
			dir:      "testDirEmpty",
			setup:    func() { os.Mkdir("testDirEmpty", 0755) },
			expected: false, // Empty directory
		},
		{
			dir:      "testDirWithFiles",
			setup:    func() { os.Mkdir("testDirWithFiles", 0755); os.Create("testDirWithFiles/file1.txt") },
			expected: true, // Directory with at least one file
		},
		{
			dir:      "testDirWithSubdir",
			setup:    func() { os.Mkdir("testDirWithSubdir", 0755); os.Mkdir("testDirWithSubdir/subdir", 0755) },
			expected: true, // Directory with a subdirectory
		},
		{
			dir:      "testDirError",
			setup:    func() { /* simulate directory not existing */ },
			expected: false, // Directory doesn't exist
		},
	}

	for _, tt := range tests {
		// Setup the directory and its contents
		tt.setup()

		// Ensure the function works as expected
		t.Run(tt.dir, func(t *testing.T) {
			result := IsDownloadStarted(tt.dir)
			if result != tt.expected {
				t.Errorf("For dir %s, expected %v but got %v", tt.dir, tt.expected, result)
			}
		})

		// Cleanup
		os.RemoveAll(tt.dir)
	}
}
