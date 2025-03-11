package videoRenderingCrypto_test

import (
	"fmt"
	"testing"

	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"
	videoRenderingCrypto "github.com/janction/videoRendering/crypto"
)

func TestWorkerSign(t *testing.T) {
	message := []byte("Validate file possession") // The message to sign
	cdc := moduletestutil.MakeTestEncodingConfig().Codec
	signature, publicKey, err := videoRenderingCrypto.SignMessage("/Users/rodrigoacosta/.janctiond", "alice", message, cdc)
	fmt.Println("error", err)
	fmt.Println("signature", signature)
	fmt.Println("publicKey", publicKey)
}
