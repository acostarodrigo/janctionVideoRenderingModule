package ipfs

import (
	"fmt"
	"os"

	shell "github.com/ipfs/go-ipfs-api"
)

func IPFSGet(cid string, path string) error {
	fmt.Println("**********************")
	fmt.Println("IPFS Downloading CID " + cid)
	fmt.Println("**********************")

	err := os.MkdirAll(path, os.ModePerm)
	if err != nil {
		return err
	}

	// Connect to the local IPFS node (ensure IPFS is running on localhost:5001)
	sh := shell.NewShell("127.0.0.1:5001")

	// Download the file from IPFS using the CID
	err = sh.Get(cid, path)
	if err != nil {
		fmt.Println("Error downloading from IPFS:", err)
		return err
	}

	fmt.Println("Download completed successfully.")
	return nil
}
