package videoRendering

import (
	"context"
	"encoding/json"
	fmt "fmt"
	"log"
	"os/exec"

	"github.com/janction/videoRendering/db"
	"github.com/janction/videoRendering/ipfs"
	"github.com/janction/videoRendering/vm"
)

func (t *VideoRenderingThread) StartWork(worker string, cid string, path string, db *db.DB) error {
	ctx := context.Background()
	log.Printf("Updating thread %s", t.ThreadId)
	if err := db.UpdateThread(t.ThreadId, true, false, false, false); err != nil {
		log.Printf("Unable to update thread status, err: %s", err.Error())
	}

	isRunning := vm.IsContainerRunning(ctx, t.ThreadId)
	if !isRunning {

		// task is not running,

		log.Printf("No solution for thread %s. Starting work", t.ThreadId)
		// we don't have a solution, start working
		err := ipfs.IPFSGet(cid, path)
		if err != nil {
			log.Printf("Error getting cid %s", cid)
			return err
		}

		err = vm.RenderVideoThread(ctx, cid, uint64(t.StartFrame), uint64(t.EndFrame), t.ThreadId, path)
		db.UpdateThread(t.ThreadId, true, true, false, false)
		if err != nil {
			return err
		}
	} else {
		// Container is running, so we update worker status
		// if worker status is idle, we change it
		log.Printf("Work for thread %s is already going", t.ThreadId)

	}

	return nil
}

func (t VideoRenderingThread) ProposeSolution(ctx context.Context, workerAddress string, rootPath string, db *db.DB) error {
	db.UpdateThread(t.ThreadId, true, true, true, false)
	count := vm.CountFilesInDirectory(rootPath)

	if count != (int(t.EndFrame)-int(t.StartFrame))+1 {
		return nil
	}

	hashes, err := vm.HashFilesInDirectory(rootPath)
	solution := MapToKeyValueFormat(hashes)
	if err != nil {
		log.Printf("Unable to get hashes in path %s. %s", rootPath, err.Error())
		return err
	}

	// // TODO call cmd with message subscribeWorkerToTask
	// sequence, err := GetAccountSequence(workerAddress)
	// if err != nil {
	// 	log.Printf("Error fetching sequence number: %s", err.Error())
	// 	return err
	// }
	executableName := "minid"
	// Base arguments
	args := []string{
		"tx", "videoRendering", "propose-solution",
		t.TaskId, t.ThreadId,
	}

	// Append solution arguments
	args = append(args, solution...)

	// Append flags
	args = append(args, "--yes", "--from", workerAddress)

	cmd := exec.Command(executableName, args...)
	log.Printf("Executing %s", cmd.String())
	_, err = cmd.Output()
	if err != nil {
		log.Printf("Error Executing %s", err.Error())
		return err
	}

	return nil
}

// MapToKeyValueFormat converts a map[string]string to a "key=value,key=value" format
func MapToKeyValueFormat(inputMap map[string]string) []string {
	var parts []string

	// Iterate through the map and build the key=value pairs
	for key, value := range inputMap {
		parts = append(parts, fmt.Sprintf("%s=%s", key, value))
	}

	// Join the key=value pairs with commas
	return parts
}

func GetAccountSequence(account string) (string, error) {
	executableName := "minid"
	cmd := exec.Command(executableName, "query", "auth", "account", account, "--output", "json")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to query account: %v", err)
	}

	sequence, err := ParseSequenceFromOutput(string(output))
	if err != nil {
		return "", fmt.Errorf("failed to parse sequence: %v", err)
	}

	return sequence, nil
}

func ParseSequenceFromOutput(output string) (string, error) {
	// Parse the YAML or JSON response to extract the sequence number
	type AccountResponse struct {
		Account struct {
			Value struct {
				Sequence string `yaml:"sequence" json:"sequence"`
			} `yaml:"value" json:"value"`
		} `yaml:"account" json:"account"`
	}

	var response AccountResponse
	if err := json.Unmarshal([]byte(output), &response); err != nil {
		return "", fmt.Errorf("failed to parse account output: %v", err)
	}

	return response.Account.Value.Sequence, nil
}
