package videoRendering

import (
	"context"
	"encoding/json"
	"errors"
	fmt "fmt"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/types"
	"github.com/janction/videoRendering/db"
	"github.com/janction/videoRendering/ipfs"
	"github.com/janction/videoRendering/vm"
)

func (t *VideoRenderingThread) StartWork(worker string, cid string, path string, db *db.DB) error {
	ctx := context.Background()

	if err := db.UpdateThread(t.ThreadId, true, false, false, false, false); err != nil {
		log.Printf("Unable to update thread status, err: %s", err.Error())
	}

	isRunning := vm.IsContainerRunning(ctx, t.ThreadId)
	if !isRunning {
		// task is not running,

		// we remove the container just in case it already exists.
		vm.RemoveContainer(ctx, "myBlender"+t.ThreadId)

		log.Printf("No solution for thread %s. Starting work", t.ThreadId)
		// we don't have a solution, start working
		started := time.Now().Unix()
		ipfs.EnsureIPFSRunning()
		db.AddLogEntry(t.ThreadId, fmt.Sprintf("Started downloading IPFS file %s...", cid), started, 0)
		err := ipfs.IPFSGet(cid, path)
		if err != nil {
			db.AddLogEntry(t.ThreadId, fmt.Sprintf("Error getting IPFS file %s. %s", cid, err.Error()), started, 2)
			log.Printf("Error getting cid %s", cid)
			return err
		}

		finish := time.Now().Unix()
		difference := time.Unix(finish, 0).Sub(time.Unix(started, 0))
		db.AddLogEntry(t.ThreadId, fmt.Sprintf("Successfully downloaded IPFS file %s in %v seconds.", cid, int(difference.Seconds())), finish, 0)

		// we start rendering
		vm.RenderVideo(ctx, cid, t.StartFrame, t.EndFrame, t.ThreadId, path, t.IsReverse(worker), db)

		rendersPath := filepath.Join(path, "output")
		_, err = os.Stat(rendersPath)
		finish = time.Now().Unix()
		difference = time.Unix(finish, 0).Sub(time.Unix(started, 0))
		if err != nil {
			// output path was not created so no rendering happened. we will start over
			log.Printf("Unable to complete rendering of task, retrying. No files at %s", rendersPath)

			db.UpdateThread(t.ThreadId, false, false, false, false, false)
			return nil
		}
		files, _ := os.ReadDir(rendersPath)
		if len(files) != int(t.EndFrame)-int(t.StartFrame)+1 {
			db.UpdateThread(t.ThreadId, false, false, false, false, false)
			log.Printf("Not the amount we expected. retrying. Amount of files %v", len(files))
			return nil
		}
		db.UpdateThread(t.ThreadId, true, true, false, false, false)
		db.AddLogEntry(t.ThreadId, fmt.Sprintf("Thread %s completed succesfully in %v seconds.", t.ThreadId, int(difference.Seconds())), finish, 1)
	} else {
		// Container is running, so we update worker status
		// if worker status is idle, we change it
		log.Printf("Work for thread %s is already going", t.ThreadId)

	}

	return nil
}

func (t VideoRenderingThread) ProposeSolution(ctx context.Context, workerAddress string, rootPath string, db *db.DB) error {
	db.UpdateThread(t.ThreadId, true, true, true, false, false)
	count := vm.CountFilesInDirectory(rootPath)

	if count != (int(t.EndFrame)-int(t.StartFrame))+1 {
		return nil
	}

	output := path.Join(rootPath, "output")
	hashes, err := ipfs.CalculateCIDs(output)

	solution := MapToKeyValueFormat(hashes)
	if err != nil {
		log.Printf("Unable to get hashes in path %s. %s", rootPath, err.Error())
		return err
	}

	executableName := "janctiond"
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

	db.AddLogEntry(t.ThreadId, "Solution proposed. Wainting confirmation...", time.Now().Unix(), 0)

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

func transformSliceToMap(input []string) (map[string]string, error) {
	result := make(map[string]string)

	for _, item := range input {
		parts := strings.SplitN(item, "=", 2) // Split into 2 parts: filename and hash
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid format: %s", item)
		}
		filename := parts[0]
		hash := parts[1]
		result[filename] = hash
	}

	return result, nil
}

func GetAccountSequence(account string) (string, error) {
	executableName := "janctiond"
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

func (t VideoRenderingThread) Verify(ctx context.Context, workerAddress string, rootPath string, db *db.DB) error {
	// we will verify any file we already have rendered.
	db.UpdateThread(t.ThreadId, true, true, true, true, false)

	files := vm.CountFilesInDirectory(rootPath)
	if files == 0 {
		log.Printf("found %v files in path %s", files, rootPath)
		return nil
	}

	// we do have some work, lets compare it with the solution
	output := path.Join(rootPath, "output")
	myWork, err := ipfs.CalculateCIDs(output)

	if err != nil {
		log.Printf("Error getting hashes. Err: %s", err.Error())
		return err
	}

	solution, _ := transformSliceToMap(t.Solution.Hashes)
	var valid bool = true
	db.AddLogEntry(t.ThreadId, "Starting verification of solution...", time.Now().Unix(), 0)
	for filename, hash := range solution {
		if myWork[filename] != hash {
			valid = false
			break
		}
	}

	submitValidation(workerAddress, t.TaskId, t.ThreadId, int64(files), valid)
	db.AddLogEntry(t.ThreadId, "Solution verified", time.Now().Unix(), 0)
	return nil
}

func submitValidation(validator string, taskId, threadId string, amount_files int64, valid bool) error {
	executableName := "janctiond"
	cmd := exec.Command(executableName, "tx", "videoRendering", "submit-validation", taskId, threadId, strconv.FormatInt(amount_files, 10), strconv.FormatBool(valid), "--from", validator, "--yes")
	_, err := cmd.Output()
	log.Printf("executing %s", cmd.String())
	if err != nil {
		return err
	}
	return nil
}

func (t VideoRenderingThread) SubmitSolution(ctx context.Context, workerAddress, rootPath string, db *db.DB) error {
	db.UpdateThread(t.ThreadId, true, true, true, true, true)

	db.AddLogEntry(t.ThreadId, "Submiting solution to IPFS...", time.Now().Unix(), 0)
	cid, err := ipfs.UploadSolution(ctx, rootPath, t.ThreadId)
	if err != nil {
		return err
	}
	err = submitSolution(workerAddress, t.TaskId, t.ThreadId, cid)
	if err != nil {
		db.AddLogEntry(t.ThreadId, fmt.Sprintf("Error submitting solution. %s", err.Error()), time.Now().Unix(), 2)
		return err
	}

	db.AddLogEntry(t.ThreadId, "Solution uploaded to IPFS correctly.", time.Now().Unix(), 0)
	return nil
}

func submitSolution(address, taskId, threadId string, cid string) error {
	executableName := "janctiond"
	args := []string{
		"tx", "videoRendering", "submit-solution",
		taskId, threadId,
	}

	// Append solution arguments
	args = append(args, cid)

	// Append flags
	args = append(args, "--yes", "--from", address)

	cmd := exec.Command(executableName, args...)
	_, err := cmd.Output()
	log.Printf("executing %s", cmd.String())
	if err != nil {
		return err
	}
	return nil
}

func (t VideoRenderingThread) VerifySubmittedSolution(cid string) error {
	result, err := ipfs.ListDirectory(cid)
	if err != nil {
		return err
	}

	solution, err := transformSliceToMap(t.Solution.Hashes)
	if err != nil {
		return err
	}

	for key, value := range solution {
		if result[key] != value {
			return errors.New("provided solution is incorrect")
		}
	}
	return nil
}

func (t VideoRenderingThread) IsReverse(worker string) bool {
	for i, v := range t.Workers {
		if v == worker {
			return i%2 != 0
		}
	}
	return false
}

func (t *VideoRenderingThread) GetValidatorReward(worker string, totalReward types.Coin) types.Coin {
	var totalFiles int
	for _, validation := range t.Validations {
		totalFiles = totalFiles + int(validation.AmountFiles)
	}
	for _, validation := range t.Validations {
		if validation.Validator == worker {
			amount := calculateValidatorPayment(int(validation.AmountFiles), totalFiles, totalReward.Amount)
			return types.NewCoin("jct", amount)
		}
	}
	return types.NewCoin("jct", math.NewInt(0))
}

// Calculate the validator's reward proportionally using sdkmath.Int
func calculateValidatorPayment(filesValidated, totalFilesValidated int, totalValidatorReward math.Int) math.Int {
	if totalFilesValidated == 0 {
		return math.NewInt(0) // Avoid division by zero
	}

	// (filesValidated * totalValidatorReward) / totalFilesValidated
	return totalValidatorReward.Mul(math.NewInt(int64(filesValidated))).Quo(math.NewInt(int64(totalFilesValidated)))
}
