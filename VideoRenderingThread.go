package videoRendering

import (
	"context"
	fmt "fmt"
	"os"
	"path"
	"path/filepath"
	"slices"
	"time"

	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/types"
	videoRenderingCrypto "github.com/janction/videoRendering/crypto"
	"github.com/janction/videoRendering/db"
	"github.com/janction/videoRendering/ipfs"
	"github.com/janction/videoRendering/videoRenderingLogger"
	"github.com/janction/videoRendering/vm"
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

func (t VideoRenderingThread) ProposeSolution(codec codec.Codec, alias, workerAddress string, rootPath string, db *db.DB) error {
	db.UpdateThread(t.ThreadId, true, true, true, false, false, false)

	output := path.Join(rootPath, "renders", t.ThreadId, "output")
	count := vm.CountFilesInDirectory(output)

	if count != (int(t.EndFrame)-int(t.StartFrame))+1 {
		videoRenderingLogger.Logger.Error("not enought local frames to propose solution: %v", count)
		db.UpdateThread(t.ThreadId, true, true, false, false, false, false)
		return nil
	}

	cids, err := ipfs.CalculateCIDs(output)
	if err != nil {
		videoRenderingLogger.Logger.Error("Unable to calculate CIDs: %s", err.Error())
		db.UpdateThread(t.ThreadId, true, true, false, false, false, false)
		return err
	}

	pkey, err := videoRenderingCrypto.ExtractPublicKey(rootPath, alias, codec)
	if err != nil {
		videoRenderingLogger.Logger.Error("Unable to extract public key for alias %s at path %s: %s", alias, rootPath, err.Error())
		db.UpdateThread(t.ThreadId, true, true, false, false, false, false)
		return err
	}

	publicKey := videoRenderingCrypto.EncodePublicKeyForCLI(pkey)
	for key, cid := range cids {
		sigMsg, err := videoRenderingCrypto.GenerateSignableMessage(cid, workerAddress)

		if err != nil {
			videoRenderingLogger.Logger.Error("Unable to generate message for worker %s and CID %s: %s", workerAddress, cid, err.Error())
			db.UpdateThread(t.ThreadId, true, true, false, false, false, false)
			return err
		}

		signature, _, err := videoRenderingCrypto.SignMessage(rootPath, alias, sigMsg, codec)

		if err != nil {
			videoRenderingLogger.Logger.Error("Unable to sign message for worker %s and CID %s: %s", workerAddress, cid, err.Error())
			db.UpdateThread(t.ThreadId, true, true, false, false, false, false)
			return err
		}
		cids[key] = videoRenderingCrypto.EncodeSignatureForCLI(signature)
	}

	solution := MapToKeyValueFormat(cids)
	if err != nil {
		videoRenderingLogger.Logger.Error("Unable to get hashes in path %s. %s", rootPath, err.Error())
		db.UpdateThread(t.ThreadId, true, true, false, false, false, false)
		return err
	}

	// Base arguments
	args := []string{
		"tx", "videoRendering", "propose-solution",
		t.TaskId, t.ThreadId,
	}

	// Append solution arguments
	args = append(args, publicKey)
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

func (t VideoRenderingThread) Verify(codec codec.Codec, alias, workerAddress string, rootPath string, db *db.DB) error {
	// we will verify any file we already have rendered.
	db.UpdateThread(t.ThreadId, true, true, true, true, false, false)
	output := path.Join(rootPath, "renders", t.ThreadId, "output")
	files := vm.CountFilesInDirectory(rootPath)
	if files == 0 {
		videoRenderingLogger.Logger.Error("found %v files in path %s", files, rootPath)
		db.UpdateThread(t.ThreadId, true, true, true, false, false, false)
		return nil
	}

	// we do have some work, lets compare it with the solution
	myWork, err := ipfs.CalculateCIDs(output)

	if err != nil {
		videoRenderingLogger.Logger.Error("error getting hashes. Err: %s", err.Error())
		db.UpdateThread(t.ThreadId, true, true, true, false, false, false)
		return err
	}

	publicKey, err := videoRenderingCrypto.GetPublicKey(rootPath, alias, codec)
	if err != nil {
		videoRenderingLogger.Logger.Error("Error getting public key for alias %s at path %s: %s", alias, rootPath, err.Error())
		db.UpdateThread(t.ThreadId, true, true, true, false, false, false)
		return err
	}

	for key, cid := range myWork {
		message, err := videoRenderingCrypto.GenerateSignableMessage(cid, workerAddress)
		if err != nil {
			videoRenderingLogger.Logger.Error("unable to generate message to sign %s: %s", message, err.Error())
			db.UpdateThread(t.ThreadId, true, true, true, false, false, false)
			return err
		}

		signature, _, err := videoRenderingCrypto.SignMessage(rootPath, alias, message, codec)

		if err != nil {
			videoRenderingLogger.Logger.Error("unable to sign message %s: %s", message, err.Error())
			db.UpdateThread(t.ThreadId, true, true, true, false, false, false)
			return err
		}
		myWork[key] = videoRenderingCrypto.EncodeSignatureForCLI(signature)
	}

	db.AddLogEntry(t.ThreadId, "Starting verification of solution...", time.Now().Unix(), 0)

	submitValidation(workerAddress, t.TaskId, t.ThreadId, videoRenderingCrypto.EncodePublicKeyForCLI(publicKey), MapToKeyValueFormat(myWork))
	db.AddLogEntry(t.ThreadId, "Solution verified", time.Now().Unix(), 0)
	return nil
}

func submitValidation(validator string, taskId, threadId, publicKey string, signatures []string) error {
	// Base arguments
	args := []string{
		"tx", "videoRendering", "submit-validation",
		taskId, threadId,
	}
	args = append(args, publicKey)
	args = append(args, signatures...)
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
		db.UpdateThread(t.ThreadId, true, true, true, true, true, false)
		videoRenderingLogger.Logger.Error(err.Error())
		return err
	}
	err = submitSolution(workerAddress, t.TaskId, t.ThreadId, cid)
	if err != nil {
		db.UpdateThread(t.ThreadId, true, true, true, true, true, false)
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

// Evaluates if the verifications sent are valid
func (t *VideoRenderingThread) EvaluateVerifications() error {
	for _, frame := range t.Solution.Frames {
		for _, validation := range t.Validations {
			idx := slices.IndexFunc(validation.Frames, func(f *VideoRenderingThread_Frame) bool { return f.Filename == frame.Filename })

			if idx < 0 {
				// This verification doesn't have the frame of the solution, we skip it hoping another validation has it
				videoRenderingLogger.Logger.Debug("Solution Frame %s, not found at validation of validator %s ", frame.Filename, validation.Validator)
				continue
			}

			videoRenderingLogger.Logger.Debug("Verifying frame %s from validator %s", validation.Frames[idx].Filename, validation.Validator)
			pk, err := videoRenderingCrypto.DecodePublicKeyFromCLI(validation.PublicKey)
			if err != nil {
				videoRenderingLogger.Logger.Error("unable to get public key from cli: %s", err.Error())
				return err
			}

			message, err := videoRenderingCrypto.GenerateSignableMessage(frame.Cid, validation.Validator)
			if err != nil {
				videoRenderingLogger.Logger.Error("unable to recreate original message %sto verify: %s", message, err.Error())
				return err
			}
			sig, err := videoRenderingCrypto.DecodeSignatureFromCLI(validation.Frames[idx].Signature)
			if err != nil {
				videoRenderingLogger.Logger.Error("unable to decode signature: %s", err.Error())
				return err
			}
			valid := pk.VerifySignature(message, sig)

			videoRenderingLogger.Logger.Debug("Verifying message cid: %s, address: %s with signature %s from pk %s ", frame.Cid, validation.Validator, validation.Frames[idx].Signature, validation.PublicKey)
			if valid {
				// verification passed
				frame.ValidCount++
			} else {
				videoRenderingLogger.Logger.Debug("Verification for frame %s from pk %s not passed!", validation.Frames[idx].Filename, validation.Validator)
				frame.InvalidCount++
			}
		}

	}
	return nil
}

// for those frames evaluated, if we have at least one that has more
// invalid counts than valid ones, we rejected. Otherwise is accepted
func (t *VideoRenderingThread) IsSolutionAccepted() bool {

	for _, frame := range t.Solution.Frames {
		if frame.ValidCount > 0 || frame.InvalidCount > 0 {
			if frame.InvalidCount-frame.ValidCount >= 0 {
				t.Solution.Accepted = false
				return false
			}

		}
	}
	t.Solution.Accepted = true
	return true
}

// validates the IPFS dir contains all files in the solution
func (t *VideoRenderingThread) VerifySubmittedSolution(dir string) error {
	files, err := ipfs.ListDirectory(dir)
	if err != nil {
		videoRenderingLogger.Logger.Error("VerifySubmittedSolution dir: %s:%s", dir, err.Error())
		return err
	}
	for _, frame := range t.Solution.Frames {
		if frame.Cid != files[frame.Filename] {
			err := fmt.Errorf("frame %s [%s] doesn't exists in %s", frame.Filename, frame.Cid, dir)
			videoRenderingLogger.Logger.Error(err.Error())
			return err
		} else {
			videoRenderingLogger.Logger.Debug("VerifySubmittedSolution file: %s [%s] exists in dir %s", frame.Filename, frame.Cid, dir)
		}

	}
	return nil
}
