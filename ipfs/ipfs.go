package ipfs

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

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
