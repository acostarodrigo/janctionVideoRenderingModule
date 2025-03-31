package ipfs

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	shell "github.com/ipfs/go-ipfs-api"
	"github.com/janction/videoRendering/videoRenderingLogger"
)

func IPFSGet(cid string, path string) error {
	videoRenderingLogger.Logger.Info("IPFS Downloading started for %s at %s", cid, path)
	err := os.MkdirAll(path, os.ModePerm)
	if err != nil {
		videoRenderingLogger.Logger.Error("Error Downloading IPFS %s: %s", cid, err.Error())
		return err
	}

	// Connect to the local IPFS node (ensure IPFS is running on localhost:5001)
	sh := shell.NewShell("127.0.0.1:5001")

	// Download the file from IPFS using the CID
	err = sh.Get(cid, path)
	if err != nil {
		videoRenderingLogger.Logger.Error("Error Downloading IPFS %s: %s", cid, err.Error())
		return err
	}

	videoRenderingLogger.Logger.Info("Download completed successfully")
	return nil
}

// CalculateCIDs recursively computes the CIDs of a directory and its contents using `ipfs add --only-hash --recursive`
func CalculateCIDs(dirPath string) (map[string]string, error) {
	cidMap := make(map[string]string)

	// Walk through the directory
	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			fail := fmt.Errorf("error accessing path %s: %w", path, err)
			videoRenderingLogger.Logger.Error(fail.Error())
			return fail
		}

		// Skip directories
		if info.IsDir() {
			videoRenderingLogger.Logger.Debug("Skipping directory %s", info.Name())
			return nil
		}

		// Execute the IPFS add command with -Q and --only-hash to get the CID
		cmd := exec.Command("ipfs", "add", "-Q", "--only-hash", path)
		var out bytes.Buffer
		cmd.Stdout = &out
		cmd.Stderr = &out

		if err := cmd.Run(); err != nil {
			fail := fmt.Errorf("failed to calculate CID for %s: %s, %w", path, out.String(), err)
			videoRenderingLogger.Logger.Error(fail.Error())
			return fail
		}

		// Extract only the file name and add the result to the map
		fileName := filepath.Base(path)
		cid := strings.TrimSpace(out.String())
		cidMap[fileName] = cid
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("error walking directory %s: %w", dirPath, err)
	}

	return cidMap, nil
}

func UploadSolution(ctx context.Context, rootPath, threadId string) (string, error) {
	// Connect to the IPFS daemon
	sh := shell.NewShell("localhost:5001") // Replace with your IPFS API address

	// Construct the path to the thread's output files
	threadOutputPath := filepath.Join(rootPath, "renders", threadId, "output")

	// Ensure the thread output path exists
	info, err := os.Stat(threadOutputPath)
	if err != nil {
		fail := fmt.Errorf("failed to access thread output path: %w", err)
		videoRenderingLogger.Logger.Error(fail.Error())
		return "", fail
	}
	if !info.IsDir() {
		fail := fmt.Errorf("thread output path is not a directory: %s", threadOutputPath)
		videoRenderingLogger.Logger.Error(fail.Error())
		return "", fail
	}

	cid, err := sh.AddDir(threadOutputPath)
	if err != nil {
		fail := fmt.Errorf("failed to upload files for threadId %s: %w", threadId, err)
		videoRenderingLogger.Logger.Error(fail.Error())
		return "", fail
	}

	return cid, nil
}

// CheckIPFSStatus pings the IPFS daemon to check if it's running
func CheckIPFSStatus() error {
	client := http.Client{
		Timeout: 2 * time.Second, // Set timeout to avoid long waits
	}

	req, err := http.NewRequest("POST", "http://localhost:5001/api/v0/id", nil) // Use POST
	if err != nil {
		fail := fmt.Errorf("failed to create request: %v", err)
		videoRenderingLogger.Logger.Error(fail.Error())
		return fail
	}

	resp, err := client.Do(req)
	if err != nil {
		fail := fmt.Errorf("IPFS node unreachable: %v", err)
		videoRenderingLogger.Logger.Error(fail.Error())
		return fail
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fail := fmt.Errorf("IPFS node returned non-200 status: %d", resp.StatusCode)
		videoRenderingLogger.Logger.Error(fail.Error())
		return fail
	}

	videoRenderingLogger.Logger.Info("✅ IPFS node is running")
	return nil
}

// StartIPFS attempts to start the IPFS daemon
func StartIPFS() error {
	cmd := exec.Command("ipfs", "daemon")
	cmd.Stdout = nil // You can redirect this if needed
	cmd.Stderr = nil // You can log errors if needed

	err := cmd.Start() // Start IPFS as a background process
	if err != nil {
		fail := fmt.Errorf("failed to start IPFS daemon: %v", err)
		videoRenderingLogger.Logger.Error(fail.Error())
		return fail
	}

	videoRenderingLogger.Logger.Info("IPFS daemon started successfully")
	return nil
}

// EnsureIPFSRunning checks and starts IPFS if needed
func EnsureIPFSRunning() {
	err := CheckIPFSStatus()
	if err != nil {
		videoRenderingLogger.Logger.Info("⚠️ IPFS not running. Attempting to start...")
		startErr := StartIPFS()
		if startErr != nil {
			videoRenderingLogger.Logger.Error("Failed to start IPFS: %v\n", startErr.Error())
		} else {
			videoRenderingLogger.Logger.Info("✅ IPFS started successfully")
		}
	}
}

// ListDirectory runs `ipfs ls {cid}` and returns a map[filename]CID with a 4s timeout.
func ListDirectory(cid string) (map[string]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "ipfs", "ls", cid) // Use context for timeout

	var out bytes.Buffer
	cmd.Stdout = &out

	err := cmd.Run()
	if ctx.Err() == context.DeadlineExceeded {
		fail := fmt.Errorf("timeout: ipfs ls command took too long")
		videoRenderingLogger.Logger.Error(fail.Error())
		return nil, fail
	}
	if err != nil {
		fail := fmt.Errorf("failed to execute ipfs ls: %v", err)
		videoRenderingLogger.Logger.Error(fail.Error())
		return nil, fail
	}

	result := make(map[string]string)
	scanner := bufio.NewScanner(&out)

	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 3 {
			continue // Ensure it has CID, size, and filename
		}
		cid := fields[0]
		filename := fields[2]
		result[filename] = cid
	}

	if err := scanner.Err(); err != nil {
		fail := fmt.Errorf("error reading command output: %v", err)
		videoRenderingLogger.Logger.Error(fail.Error())
		return nil, fail
	}

	return result, nil
}

// Function to connect to IPFS nodes
func ConnectToIPFSNode(ip, peerId string) {
	seed, _ := GenerateSwarmConnectURL(ip, peerId)
	cmd := exec.Command("ipfs", "swarm", "connect", seed)
	output, err := cmd.CombinedOutput()
	if err != nil {
		videoRenderingLogger.Logger.Error("Failed to connect to %s: %v\nOutput: %s", seed, err, output)
	} else {
		videoRenderingLogger.Logger.Info("Connected to IPFS node: %s\n", seed)
	}
}

// IPFSIDResponse represents the structure of the `ipfs id` JSON response.
type IPFSIDResponse struct {
	ID string `json:"ID"`
}

// GetIPFSPeerID runs `ipfs id` and extracts the Peer ID.
func GetIPFSPeerID() (string, error) {
	cmd := exec.Command("ipfs", "id")
	output, err := cmd.Output()
	if err != nil {
		fail := fmt.Errorf("failed to run ipfs id: %w", err)
		videoRenderingLogger.Logger.Error(fail.Error())
		return "", fail
	}

	var response IPFSIDResponse
	if err := json.Unmarshal(output, &response); err != nil {
		fail := fmt.Errorf("failed to parse ipfs id output: %w", err)
		videoRenderingLogger.Logger.Error(fail.Error())
		return "", fail
	}

	return response.ID, nil
}

// GenerateSwarmConnectURL creates the full IPFS swarm connect URL.
func GenerateSwarmConnectURL(ip, peerID string) (string, error) {
	return fmt.Sprintf("/ip4/%s/tcp/4001/p2p/%s", ip, peerID), nil
}

// Checks at the specified path if a file exists. This path will be tipically
// .janctiond/renders/[threadId]. If IPFS started downloading a file, a temp file will exists
// if it is empty, it is safe to assume download hasn't started and probably won't
func IsDownloadStarted(dir string) bool {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false // directory doesn't exist or isn't readable
	}

	return len(entries) > 0 // true if there's at least one file or subdir
}
