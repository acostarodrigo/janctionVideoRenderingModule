package vm

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
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

func RenderVideo(ctx context.Context, cid string, start int64, end int64, id string, path string, reverse bool, db db.Database) {
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

func renderVideoFrame(ctx context.Context, cid string, frameNumber int64, id string, path string, db db.Database) error {
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

	var blenderArgs []string
	if isARM64() {
		blenderArgs = append(blenderArgs, "blender")
	}
	blenderArgs = append(blenderArgs, "--background")
	blenderArgs = append(blenderArgs, fmt.Sprintf("/workspace/%s", cid))
	blenderArgs = append(blenderArgs, "--engine")
	blenderArgs = append(blenderArgs, "CYCLES")

	blenderArgs = append(blenderArgs, "--render-output")
	blenderArgs = append(blenderArgs, "/workspace/output/frame_######")
	blenderArgs = append(blenderArgs, "--render-format")
	blenderArgs = append(blenderArgs, "PNG")
	blenderArgs = append(blenderArgs, "--render-frame")
	blenderArgs = append(blenderArgs, strconv.FormatInt(frameNumber, 10))

	var dockerArgs []string
	dockerArgs = append(dockerArgs, "run")

	dockerArgs = append(dockerArgs, "--name")
	dockerArgs = append(dockerArgs, n)
	dockerArgs = append(dockerArgs, "-v")
	dockerArgs = append(dockerArgs, bindPath)
	dockerArgs = append(dockerArgs, "-d")

	// TODO if on Mac, we use another image that is non deterministic
	if isARM64() {
		dockerArgs = append(dockerArgs, "blender_render")
	} else {
		// if non Mac, this image generates deterministics results
		dockerArgs = append(dockerArgs, "blendergrid/blender")
	}
	dockerArgs = append(dockerArgs, blenderArgs...)

	// Create and start the container
	runCmd := exec.CommandContext(ctx, "docker", dockerArgs...)
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
		renderVideoFrame(ctx, cid, frameNumber, id, path, db)
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

func isARM64() bool {
	videoRenderingLogger.Logger.Debug("isARM64: %s", runtime.GOARCH)
	return runtime.GOARCH == "arm64"
}

func IsContainerExited(threadId string) (bool, error) {
	containerName := "myBlender" + threadId
	cmd := exec.Command(
		"docker", "ps", "-a",
		"--filter", "name="+containerName,
		"--filter", "status=exited",
		"--format", "{{.Names}}",
	)

	var out bytes.Buffer
	cmd.Stdout = &out

	if err := cmd.Run(); err != nil {
		return false, err
	}

	// Trim and check if the container name appears in output
	result := strings.TrimSpace(out.String())
	return result == containerName, nil
}
