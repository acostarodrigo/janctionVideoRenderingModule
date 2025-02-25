package videoRendering

import (
	io "io"
	"log"
	"net/http"
	"os/exec"
	"strings"

	"github.com/cosmos/cosmos-sdk/types"
	"github.com/janction/videoRendering/db"
	"github.com/janction/videoRendering/ipfs"
)

func (w Worker) RegisterWorker(address string, stake types.Coin, db *db.DB) error {
	db.Addworker(address)
	executableName := "janctiond"
	ip, _ := getPublicIP()
	ipfsId, _ := ipfs.GetIPFSPeerID()
	cmd := exec.Command(executableName, "tx", "videoRendering", "add-worker", ip, ipfsId, stake.String(), "--from", address, "--yes")
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

func (w *Worker) DeclareWinner(payment types.Coin) {
	w.Reputation.Points = w.Reputation.Points + 1
	w.Reputation.Solutions = w.Reputation.Solutions + 1
	w.Reputation.Winnings = w.Reputation.Winnings.Add(payment)
}
