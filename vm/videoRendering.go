package vm

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path"
	"strings"
)

func IsContainerRunning(ctx context.Context, threadId string) bool {
	name := fmt.Sprintf("myBlender%s", threadId)

	// Command to check for running containers
	cmd := exec.CommandContext(ctx, "docker", "ps", "--filter", fmt.Sprintf("name=%s", name), "--format", "{{.Names}}")

	output, err := cmd.Output()
	if err != nil {
		fmt.Printf("Error executing Docker command: %v\n", err)
		return false
	}

	// Trim output and compare with container name
	containerName := strings.TrimSpace(string(output))
	return containerName == name
}

func RenderVideoThread(ctx context.Context, cid string, s uint64, e uint64, id string, path string) error {
	n := "myBlender" + id

	// Check if the container exists using `docker ps -a`
	checkCmd := exec.CommandContext(ctx, "docker", "ps", "-a", "--filter", fmt.Sprintf("name=%s", n), "--format", "{{.Names}}")
	output, err := checkCmd.Output()
	if err != nil {
		return fmt.Errorf("failed to check container existence: %w", err)
	}

	// If the container already exists, exit the function
	if string(output) != "" {
		fmt.Println("Container already exists.")
		return nil
	}

	// Construct the bind path and command
	bindPath := fmt.Sprintf("%s:/workspace", path)
	command := fmt.Sprintf(
		"blender -b /workspace/%s -o /workspace/output/frame_###### -F PNG -E CYCLES -s %d -e %d -a",
		cid, s, e,
	)

	// Create and start the container
	runCmd := exec.CommandContext(ctx, "docker", "run", "--name", n, "-v", bindPath, "-d", "blender_render", "sh", "-c", command)
	err = runCmd.Run()
	if err != nil {
		return fmt.Errorf("failed to create and start container: %w", err)
	}

	// Wait for the container to finish
	waitCmd := exec.CommandContext(ctx, "docker", "wait", n)
	err = waitCmd.Run()
	if err != nil {
		return fmt.Errorf("failed to wait for container: %w", err)
	}

	// Retrieve and print logs
	logsCmd := exec.CommandContext(ctx, "docker", "logs", n)
	logsOutput, err := logsCmd.Output()
	if err != nil {
		return fmt.Errorf("failed to retrieve container logs: %w", err)
	}
	fmt.Println("Container logs:")
	fmt.Println(string(logsOutput))

	// Remove the container after completion
	rmCmd := exec.CommandContext(ctx, "docker", "rm", n)
	err = rmCmd.Run()
	if err != nil {
		return fmt.Errorf("failed to remove container: %w", err)
	}

	return nil
}

// CountFilesInDirectory counts the number of files in a given directory
func CountFilesInDirectory(directoryPath string) int {
	output := path.Join(directoryPath, "output")
	// Read the directory contents
	files, err := os.ReadDir(output)
	if err != nil {
		return 0
	}

	// Count only files (not subdirectories)
	fileCount := 0
	for _, file := range files {
		if !file.IsDir() {
			fileCount++
		}
	}
	return fileCount
}

// // HashFilesInDirectory calculates the SHA-256 hashes of all PNG files in a directory
// func HashFilesInDirectory(rootPath string) (map[string]string, error) {
// 	directoryPath := path.Join(rootPath, "output")
// 	hashes, err := ipfs.CalculateCIDs(directoryPath)

// 	return hashes, err
// 	// // Map to store file names and their corresponding hashes
// 	// hashes := make(map[string]string)

// 	// // Walk through the directory
// 	// err := filepath.Walk(directoryPath, func(path string, info os.FileInfo, err error) error {
// 	// 	if err != nil {
// 	// 		return err
// 	// 	}

// 	// 	// Skip directories
// 	// 	if info.IsDir() {
// 	// 		return nil
// 	// 	}

// 	// 	// Check if the file has a .png extension
// 	// 	if filepath.Ext(info.Name()) == ".png" {
// 	// 		// Open the file
// 	// 		file, err := os.Open(path)
// 	// 		if err != nil {
// 	// 			return err
// 	// 		}
// 	// 		defer file.Close()

// 	// 		// Compute the hash
// 	// 		hasher := sha256.New()
// 	// 		if _, err := io.Copy(hasher, file); err != nil {
// 	// 			return err
// 	// 		}

// 	// 		// Convert the hash to a hex string
// 	// 		hash := hex.EncodeToString(hasher.Sum(nil))

// 	// 		// Store the hash in the map
// 	// 		hashes[info.Name()] = hash
// 	// 	}

// 	// 	return nil
// 	// })

// 	// if err != nil {
// 	// 	return nil, err
// 	// }

// 	// return hashes, nil
// }
