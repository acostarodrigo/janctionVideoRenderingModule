package ipfs

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	shell "github.com/ipfs/go-ipfs-api"
)

func IPFSGet(cid string, path string) error {
	fmt.Println("**********************")
	fmt.Println("IPFS Downloading CID " + cid)
	fmt.Println("**********************")

	err := os.MkdirAll(path, os.ModePerm)
	if err != nil {
		return err
	}

	// Connect to the local IPFS node (ensure IPFS is running on localhost:5001)
	sh := shell.NewShell("127.0.0.1:5001")

	// Download the file from IPFS using the CID
	err = sh.Get(cid, path)
	if err != nil {
		fmt.Println("Error downloading from IPFS:", err)
		return err
	}

	fmt.Println("Download completed successfully.")
	return nil
}

// CalculateCIDs recursively computes the CIDs of a directory and its contents using `ipfs add --only-hash --recursive`
func CalculateCIDs(dirPath string) (map[string]string, error) {
	cidMap := make(map[string]string)

	// Walk through the directory
	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return fmt.Errorf("error accessing path %s: %w", path, err)
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Execute the IPFS add command with -Q and --only-hash to get the CID
		cmd := exec.Command("ipfs", "add", "-Q", "--only-hash", path)
		var out bytes.Buffer
		cmd.Stdout = &out
		cmd.Stderr = &out

		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to calculate CID for %s: %s, %w", path, out.String(), err)
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
		return "", fmt.Errorf("failed to access thread output path: %w", err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("thread output path is not a directory: %s", threadOutputPath)
	}

	cid, err := sh.AddDir(threadOutputPath)
	if err != nil {
		return "", fmt.Errorf("failed to upload files for threadId %s: %w", threadId, err)
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
		return fmt.Errorf("failed to create request: %v", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("IPFS node unreachable: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("IPFS node returned non-200 status: %d", resp.StatusCode)
	}

	fmt.Println("✅ IPFS node is running")
	return nil
}

// StartIPFS attempts to start the IPFS daemon
func StartIPFS() error {
	cmd := exec.Command("ipfs", "daemon")
	cmd.Stdout = nil // You can redirect this if needed
	cmd.Stderr = nil // You can log errors if needed

	err := cmd.Start() // Start IPFS as a background process
	if err != nil {
		return fmt.Errorf("failed to start IPFS daemon: %v", err)
	}

	fmt.Println("IPFS daemon started successfully")
	return nil
}

// EnsureIPFSRunning checks and starts IPFS if needed
func EnsureIPFSRunning() {
	err := CheckIPFSStatus()
	if err != nil {
		fmt.Println("⚠️ IPFS not running. Attempting to start...")
		startErr := StartIPFS()
		if startErr != nil {
			fmt.Printf("Failed to start IPFS: %v\n", startErr)
		} else {
			fmt.Println("✅ IPFS started successfully")
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
		return nil, fmt.Errorf("timeout: ipfs ls command took too long")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to execute ipfs ls: %v", err)
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
		return nil, fmt.Errorf("error reading command output: %v", err)
	}

	return result, nil
}
