package videoRendering

import (
	"context"
	fmt "fmt"
	"log"
	"os/exec"
	"strings"

	"github.com/janction/videoRendering/ipfs"
	"github.com/janction/videoRendering/vm"
)

func (t *VideoRenderingThread) StartWork(worker string, cid string, path string) error {
	ctx := context.Background()
	isRunning := vm.IsContainerRunning(ctx, t.ThreadId)
	if !isRunning {
		// task is not running,
		// it could have finished? before we start it we check if we already have all the files to propose a solution
		count := vm.CountFilesInDirectory(path)

		if count == (int(t.EndFrame)-int(t.StartFrame))+1 {
			// we have a solution!!
			log.Printf("proposing solution for thread  %s", t.ThreadId)
			t.ProposeSolution(ctx, worker, path)
			return nil

		} else {
			log.Printf("No solution for thread %s. Starting work", t.ThreadId)
			// we don't have a solution, start working
			err := ipfs.IPFSGet(cid, path)
			if err != nil {
				log.Printf("Error getting cid %s", cid)
				return err
			}

			err = vm.RenderVideoThread(ctx, cid, uint64(t.StartFrame), uint64(t.EndFrame), t.ThreadId, path)
			if err != nil {
				return err
			}

		}
	} else {
		// Container is running, so we update worker status
		// if worker status is idle, we change it
		log.Printf("Work for thread %s is already going", t.ThreadId)

	}

	return nil
}

func (t VideoRenderingThread) ProposeSolution(ctx context.Context, workerAddress string, rootPath string) error {
	hashes, err := vm.HashFilesInDirectory(rootPath)
	if err != nil {
		log.Printf("Unable to get hashes in path %s. %s", rootPath, err.Error())
		return err
	}

	solution := MapToKeyValueFormat(hashes)

	// TODO call cmd with message subscribeWorkerToTask
	executableName := "minid"

	cmd := exec.Command(executableName, "tx", "videoRendering", "propose-solution", t.TaskId, t.ThreadId, solution, "--yes", "--from", workerAddress)
	log.Printf("Executing %s", cmd.String())
	_, err = cmd.Output()
	if err != nil {
		log.Printf("Error Executing %s", err.Error())
		return err
	}

	return nil
}

// MapToKeyValueFormat converts a map[string]string to a "key=value,key=value" format
func MapToKeyValueFormat(inputMap map[string]string) string {
	var parts []string

	// Iterate through the map and build the key=value pairs
	for key, value := range inputMap {
		parts = append(parts, fmt.Sprintf("%s=%s", key, value))
	}

	// Join the key=value pairs with commas
	return strings.Join(parts, ",")
}
