package ipfs_test

import (
	"fmt"
	"testing"

	"github.com/janction/videoRendering/ipfs"
)

func TestConnectionToSeeds(t *testing.T) {
	config, _ := ipfs.ReadConfig("./config.yaml")
	fmt.Println(len(config.IPFSSeeds))
	ipfs.ConnectToIPFSNodes(config.IPFSSeeds)
}
