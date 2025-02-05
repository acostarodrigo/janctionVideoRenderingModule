package videoRendering

import (
	io "io"
	"log"
	"net/http"
	"os/exec"
	"strings"

	"github.com/janction/videoRendering/db"
)

func (w Worker) RegisterWorker(address string, db *db.DB) error {
	db.Addworker(address)
	executableName := "janctiond"
	ip, _ := getPublicIP()
	cmd := exec.Command(executableName, "tx", "videoRendering", "add-worker", ip, "--from", address, "--yes")
	_, err := cmd.Output()
	log.Printf("executing %s", cmd.String())
	if err != nil {
		db.DeleteWorker(address)
		return err
	}
	return nil
}

// GetPublicIP fetches the public IP of the machine
func getPublicIP() (string, error) {
	resp, err := http.Get("https://api.ipify.org")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	ip, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(ip)), nil
}
