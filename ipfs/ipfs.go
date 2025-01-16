package ipfs

import (
	"context"
	"fmt"
	"log"
	"os"
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

func UploadSolution(ctx context.Context, rootPath, threadId string) ([]string, error) {
	// Connect to the IPFS daemon
	sh := shell.NewShell("localhost:5001") // Replace with your IPFS API address

	// Construct the path to the thread's output files
	threadOutputPath := filepath.Join(rootPath, "renders", threadId, "output")

	// Ensure the thread output path exists
	info, err := os.Stat(threadOutputPath)
	if err != nil {
		return nil, fmt.Errorf("failed to access thread output path: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("thread output path is not a directory: %s", threadOutputPath)
	}

	// List of files to upload
	var addedFileCIDs []string

	// Walk the thread output path and upload PNG files
	err = filepath.Walk(threadOutputPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		// Only upload PNG files
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".png") {
			// Open the file as a reader
			file, err := os.Open(path)
			if err != nil {
				return fmt.Errorf("failed to open file %s: %w", path, err)
			}

			// Add the file to IPFS
			cid, err := sh.Add(file)
			file.Close() // Close the file after it's successfully uploaded
			if err != nil {
				return fmt.Errorf("failed to upload file %s: %w", path, err)
			}

			log.Printf("Uploaded file %s with CID %s", path, cid)

			// Add the CID to the list of added files
			addedFileCIDs = append(addedFileCIDs, cid)

		}
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to upload files for threadId %s: %w", threadId, err)
	}

	return addedFileCIDs, nil
}
