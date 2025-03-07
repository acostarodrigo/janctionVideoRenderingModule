package videoRendering

import (
	"context"
	fmt "fmt"
	"os"
	"path"
	"path/filepath"
	"time"

	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/types"
	"github.com/janction/videoRendering/db"
	"github.com/janction/videoRendering/ipfs"
	"github.com/janction/videoRendering/videoRenderingLogger"
	"github.com/janction/videoRendering/vm"
	"github.com/janction/videoRendering/zkp"
)

func (t *VideoRenderingThread) StartWork(worker string, cid string, path string, db *db.DB) error {
	ctx := context.Background()

	if err := db.UpdateThread(t.ThreadId, true, false, false, false, false, false); err != nil {
		videoRenderingLogger.Logger.Error("Unable to update thread status, err: %s", err.Error())
	}

	isRunning := vm.IsContainerRunning(ctx, t.ThreadId)
	if !isRunning {
		// task is not running,

		// we remove the container just in case it already exists.
		vm.RemoveContainer(ctx, "myBlender"+t.ThreadId)

		videoRenderingLogger.Logger.Info("No solution for thread %s. Starting work", t.ThreadId)
		// we don't have a solution, start working
		started := time.Now().Unix()
		ipfs.EnsureIPFSRunning()
		db.AddLogEntry(t.ThreadId, fmt.Sprintf("Started downloading IPFS file %s...", cid), started, 0)
		err := ipfs.IPFSGet(cid, path)
		if err != nil {
			db.AddLogEntry(t.ThreadId, fmt.Sprintf("Error getting IPFS file %s. %s", cid, err.Error()), started, 2)
			videoRenderingLogger.Logger.Error("Error getting cid %s", cid)
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
			videoRenderingLogger.Logger.Error("Unable to complete rendering of task, retrying. No files at %s", rendersPath)

			db.UpdateThread(t.ThreadId, false, false, false, false, false, false)
			return nil
		}
		files, _ := os.ReadDir(rendersPath)
		if len(files) != int(t.EndFrame)-int(t.StartFrame)+1 {
			db.UpdateThread(t.ThreadId, false, false, false, false, false, false)
			videoRenderingLogger.Logger.Error("Not the amount we expected. retrying. Amount of files %v", len(files))
			return nil
		}
		db.UpdateThread(t.ThreadId, true, true, false, false, false, false)
		db.AddLogEntry(t.ThreadId, fmt.Sprintf("Thread %s completed succesfully in %v seconds.", t.ThreadId, int(difference.Seconds())), finish, 1)
	} else {
		// Container is running, so we update worker status
		// if worker status is idle, we change it
		videoRenderingLogger.Logger.Info("Work for thread %s is already going", t.ThreadId)

	}

	return nil
}

func (t VideoRenderingThread) ProposeSolution(ctx context.Context, workerAddress string, rootPath string, db *db.DB, provingKeyPath string) error {
	db.UpdateThread(t.ThreadId, true, true, true, false, false, false)
	count := vm.CountFilesInDirectory(rootPath)

	if count != (int(t.EndFrame)-int(t.StartFrame))+1 {
		return nil
	}

	output := path.Join(rootPath, "output")
	cids, err := ipfs.CalculateCIDs(output)

	for key, cid := range cids {
		prove, err := zkp.GenerateFrameProof(cid, workerAddress, provingKeyPath)
		if err != nil {
			videoRenderingLogger.Logger.Error("Error %s, %s", cid, provingKeyPath)
			videoRenderingLogger.Logger.Error("Error %s", err.Error())
			panic(err)
		}
		videoRenderingLogger.Logger.Info("Calculating prove %s for cid %s", prove, cid)
		// TODO handle error
		cids[key] = prove
	}

	solution := MapToKeyValueFormat(cids)
	if err != nil {
		videoRenderingLogger.Logger.Error("Unable to get hashes in path %s. %s", rootPath, err.Error())
		return err
	}

	// Base arguments
	args := []string{
		"tx", "videoRendering", "propose-solution",
		t.TaskId, t.ThreadId,
	}

	// Append solution arguments
	args = append(args, solution...)

	// Append flags
	args = append(args, "--yes", "--from", workerAddress)
	err = ExecuteCli(args)
	if err != nil {
		videoRenderingLogger.Logger.Error(err.Error())
		return err
	}

	db.AddLogEntry(t.ThreadId, "Solution proposed. Wainting confirmation...", time.Now().Unix(), 0)

	return nil
}

func (t VideoRenderingThread) Verify(ctx context.Context, workerAddress string, rootPath string, db *db.DB, provingKeyPath string) error {
	// we will verify any file we already have rendered.
	db.UpdateThread(t.ThreadId, true, true, true, true, false, false)

	files := vm.CountFilesInDirectory(rootPath)
	if files == 0 {
		videoRenderingLogger.Logger.Error("found %v files in path %s", files, rootPath)
		return nil
	}

	// we do have some work, lets compare it with the solution
	output := path.Join(rootPath, "output")
	myWork, err := ipfs.CalculateCIDs(output)

	if err != nil {
		videoRenderingLogger.Logger.Error("Error getting hashes. Err: %s", err.Error())
		return err
	}

	for key, cid := range myWork {
		proof, err := zkp.GenerateFrameProof(cid, workerAddress, provingKeyPath)
		if err != nil {
			videoRenderingLogger.Logger.Error(err.Error())
			return err
		}

		myWork[key] = proof
	}

	db.AddLogEntry(t.ThreadId, "Starting verification of solution...", time.Now().Unix(), 0)

	submitValidation(workerAddress, t.TaskId, t.ThreadId, MapToKeyValueFormat(myWork))
	db.AddLogEntry(t.ThreadId, "Solution verified", time.Now().Unix(), 0)
	return nil
}

func submitValidation(validator string, taskId, threadId string, zkps []string) error {
	// Base arguments
	args := []string{
		"tx", "videoRendering", "submit-validation",
		taskId, threadId,
	}
	args = append(args, zkps...)
	args = append(args, "--from")
	args = append(args, validator)
	args = append(args, "--yes")
	err := ExecuteCli(args)
	if err != nil {
		return err
	}
	return nil
}

func (t VideoRenderingThread) SubmitSolution(ctx context.Context, workerAddress, rootPath string, db *db.DB) error {
	db.UpdateThread(t.ThreadId, true, true, true, true, true, true)

	db.AddLogEntry(t.ThreadId, "Submiting solution to IPFS...", time.Now().Unix(), 0)
	cid, err := ipfs.UploadSolution(ctx, rootPath, t.ThreadId)
	if err != nil {
		videoRenderingLogger.Logger.Error(err.Error())
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
	args := []string{
		"tx", "videoRendering", "submit-solution",
		taskId, threadId,
	}

	// Append solution arguments
	args = append(args, cid)

	// Append flags
	args = append(args, "--yes", "--from", address)

	err := ExecuteCli(args)
	if err != nil {
		return err
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
		totalFiles = totalFiles + int(len(validation.Frames))
	}
	for _, validation := range t.Validations {
		if validation.Validator == worker {
			amount := calculateValidatorPayment(int(len(validation.Frames)), totalFiles, totalReward.Amount)
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

func (t *VideoRenderingThread) RevealSolution(rootPath string, db *db.DB) error {
	output := path.Join(rootPath, "renders", t.ThreadId, "output")
	cids, err := ipfs.CalculateCIDs(output)
	if err != nil {
		videoRenderingLogger.Logger.Error(err.Error())
		return err
	}
	solution := MapToKeyValueFormat(cids)

	// Base arguments
	args := []string{
		"tx", "videoRendering", "reveal-solution",
		t.TaskId, t.ThreadId,
	}
	args = append(args, solution...)
	args = append(args, "--from")
	args = append(args, t.Solution.ProposedBy)
	args = append(args, "--yes")
	err = ExecuteCli(args)

	if err != nil {
		return err
	}
	err = db.UpdateThread(t.ThreadId, true, true, true, true, true, false)
	if err != nil {
		return err
	}
	return nil
}
