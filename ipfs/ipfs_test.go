package ipfs

import (
	"fmt"
	"testing"

	"github.com/janction/videoRendering/ipfs"
)

func TestCalculateCIDs(t *testing.T) {
	result, _ := ipfs.CalculateCIDs("/Users/rodrigoacosta/.janctiond/renders/10/output")
	fmt.Println(result)
}
