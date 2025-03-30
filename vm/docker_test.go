package vm

import (
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"testing"
)

func TestIsContainerRunning(t *testing.T) {
	ctx := context.Background()

	// Start the blender container for thread2
	name := fmt.Sprintf("myBlender%d", 2)
	runCmd := exec.CommandContext(ctx, "docker", "run", "--name", name, "-d", "lscr.io/linuxserver/blender", "sh", "-c", "sleep infinity")
	runCmd.Run()

	// Expect that the thread1 container is not running
	// but the thread2 container is running
	var tests = []struct {
		threadId uint64
		want     bool
	}{
		{1, false},
		{2, true},
	}

	// Run the tests
	for _, tt := range tests {
		testname := fmt.Sprintf("%d", tt.threadId)
		t.Run(testname, func(t *testing.T) {
			threadId := strconv.FormatUint(tt.threadId, 10)
			ans := IsContainerRunning(ctx, threadId)
			if ans != tt.want {
				t.Errorf("got %t, want %t", ans, tt.want)
			}
		})
	}

	// Remove the thread2 container
	runCmd = exec.CommandContext(ctx, "docker", "rm", name, "-f", "myBlender2")
	runCmd.Run()
}

