package videoRendering

import (
	"log"
	"os/exec"

	"github.com/janction/videoRendering/db"
)

func (w Worker) RegisterWorker(address string, db *db.DB) error {
	db.Addworker(address)
	executableName := "janctiond"
	cmd := exec.Command(executableName, "tx", "videoRendering", "add-worker", "--from", address, "--yes")
	_, err := cmd.Output()
	log.Printf("executing %s", cmd.String())
	if err != nil {
		db.DeleteWorker(address)
		return err
	}
	return nil
}
