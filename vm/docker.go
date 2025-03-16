package vm

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/janction/videoRendering/db"
	"github.com/janction/videoRendering/videoRenderingLogger"
)

func IsContainerRunning(ctx context.Context, threadId string) bool {
	name := fmt.Sprintf("myBlender%s", threadId)

	// Command to check for running containers
	cmd := exec.CommandContext(ctx, "docker", "ps", "--filter", fmt.Sprintf("name=%s", name), "--format", "{{.Names}}")

	output, err := cmd.Output()
	if err != nil {
		videoRenderingLogger.Logger.Error("Error executing Docker command: %v\n", err)
		return false
	}

	// Trim output and compare with container name
	containerName := strings.TrimSpace(string(output))
	return containerName == name
}

func RenderVideo(ctx context.Context, cid string, start int64, end int64, id string, path string, reverse bool, db *db.DB) {
	if reverse {
		for i := end; i >= start; i-- {
			videoRenderingLogger.Logger.Info("Rendering frame %v in reverse", i)
			renderVideoFrame(ctx, cid, i, id, path, db)
		}
	} else {
		for i := start; i <= end; i++ {
			videoRenderingLogger.Logger.Info("Rendering frame %v", i)
			renderVideoFrame(ctx, cid, i, id, path, db)
		}
	}
}

func renderVideoFrame(ctx context.Context, cid string, frameNumber int64, id string, path string, db *db.DB) error {
	n := "myBlender" + id

	started := time.Now().Unix()
	db.AddLogEntry(id, fmt.Sprintf("Started rendering frame %v...", frameNumber), started, 0)

	// Check if the container exists using `docker ps -a`
	checkCmd := exec.CommandContext(ctx, "docker", "ps", "-a", "--filter", fmt.Sprintf("name=%s", n), "--format", "{{.Names}}")
	output, err := checkCmd.Output()
	if err != nil {
		db.AddLogEntry(id, "Error trying to verify if container already exists.", started, 2)
		fail := fmt.Errorf("failed to check container existence: %w", err)
		videoRenderingLogger.Logger.Error(fail.Error())
		return fail
	}

	// If the container already exists, exit the function
	if string(output) != "" {
		videoRenderingLogger.Logger.Debug("Container already exists.")
		return nil
	}

	// Construct the bind path and command
	bindPath := fmt.Sprintf("%s:/workspace", path)
	command := fmt.Sprintf(
		"blender -b ../workspace/%s -o ../workspace/output/frame_###### -F PNG -E CYCLES -s %d -e %d -a -P set_params.py",
		cid, frameNumber, frameNumber,
	)

	// Create and start the container
	runCmd := exec.CommandContext(ctx, "docker", "run", "--name", n, "-v", bindPath, "-d", "blender_render", "sh", "-c", command)
	videoRenderingLogger.Logger.Info("Starting docker: %s", runCmd.String())
	err = runCmd.Run()
	if err != nil {
		db.AddLogEntry(id, fmt.Sprintf("Error in crearing the container. %s", err.Error()), started, 1)
		videoRenderingLogger.Logger.Error("failed to create and start container: %s", err.Error())
		return fmt.Errorf("failed to create and start container: %w", err)
	}

	// Wait for the container to finish
	waitCmd := exec.CommandContext(ctx, "docker", "wait", n)
	err = waitCmd.Run()
	if err != nil {
		videoRenderingLogger.Logger.Error("failed to wait for container: %s", err.Error())
		return fmt.Errorf("failed to wait for container: %w", err)
	}

	// Retrieve and print logs
	logsCmd := exec.CommandContext(ctx, "docker", "logs", n)
	logsOutput, err := logsCmd.Output()
	if err != nil {
		videoRenderingLogger.Logger.Error("failed to retrieve container logs: %s", err.Error())
		return fmt.Errorf("failed to retrieve container logs: %w", err)
	}
	videoRenderingLogger.Logger.Info("Container logs:")
	videoRenderingLogger.Logger.Info(string(logsOutput))

	RemoveContainer(ctx, n)

	// Verify the frame exists and log
	frameFile := FormatFrameFilename(int(frameNumber))
	framePath := filepath.Join(path, "output", frameFile)
	finish := time.Now().Unix()
	difference := time.Unix(finish, 0).Sub(time.Unix(started, 0))
	if _, err := os.Stat(framePath); errors.Is(err, os.ErrNotExist) {
		db.AddLogEntry(id, fmt.Sprintf("Error while rendering frame %v. %s file is not there", frameNumber, framePath), started, 2)
	} else {
		db.AddLogEntry(id, fmt.Sprintf("Successfully rendered frame %v in %v seconds.", frameNumber, int(difference.Seconds())), finish, 1)
	}
	return nil
}

func RemoveContainer(ctx context.Context, name string) error {
	// Remove the container after completion
	rmCmd := exec.CommandContext(ctx, "docker", "rm", name)
	err := rmCmd.Run()
	if err != nil {
		videoRenderingLogger.Logger.Error(err.Error())
	}
	return err
}

// CountFilesInDirectory counts the number of files in a given directory
func CountFilesInDirectory(directoryPath string) int {
	// Read the directory contents
	files, err := os.ReadDir(directoryPath)
	if err != nil {
		videoRenderingLogger.Logger.Error(err.Error())
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

// FormatFrameFilename returns the correct filename for a given frame number.
func FormatFrameFilename(frameNumber int) string {
	return fmt.Sprintf("frame_%06d.png", frameNumber)
}
