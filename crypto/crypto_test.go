package videoRenderingCrypto_test

import (
	"testing"

	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"
	videoRenderingCrypto "github.com/janction/videoRendering/crypto"
)

func TestPublicKeyMatch(t *testing.T) {
	cdc := moduletestutil.MakeTestEncodingConfig().Codec
	message := []byte("Validate file possession") // The message to sign
	pk, err := videoRenderingCrypto.GetPublicKey("/Users/rodrigoacosta/.janctiond", "alice", cdc)
	if err != nil {
		t.Error(err)
	}

	_, publicKey, err := videoRenderingCrypto.SignMessage("/Users/rodrigoacosta/.janctiond", "alice", message, cdc)
	if err != nil {
		t.Error(err)
	}
	if pk.String() != publicKey.String() {
		t.Errorf("invalid public key %s != %s", pk.String(), publicKey.String())
	}
}
func TestWorkerSignAndValidation(t *testing.T) {
	message, err := videoRenderingCrypto.GenerateSignableMessage("cid", "address")
	if err != nil {
		t.Error(err)
	}

	cdc := moduletestutil.MakeTestEncodingConfig().Codec
	signature, publicKey, err := videoRenderingCrypto.SignMessage("/Users/rodrigoacosta/.janctiond", "alice", message, cdc)
	if err != nil {
		t.Error(err)
	}

	valid := publicKey.VerifySignature(message, signature)
	if !valid {
		t.Error("Signature is not valid")
	} else {
		t.Log("Signature is valid")
	}
}

func TestSerializationPublicKey(t *testing.T) {
	cdc := moduletestutil.MakeTestEncodingConfig().Codec
	pk, err := videoRenderingCrypto.GetPublicKey("/Users/rodrigoacosta/.janctiond", "alice", cdc)
	if err != nil {
		t.Error(err)
	}

	encodedPk := videoRenderingCrypto.EncodePublicKeyForCLI(pk)
	t.Log("base 64 publicKey", encodedPk)

	pubkey, err := videoRenderingCrypto.DecodePublicKeyFromCLI(encodedPk)
	if err != nil {
		t.Error(err)
	}

	t.Log("decoded public key", pubkey.String())
}

func TestSerializationSignature(t *testing.T) {
	message, err := videoRenderingCrypto.GenerateSignableMessage("QmRe3MVV1NeF84sgiBCeKBhwDGFVcyLPzcky4fN2cKvTzs", "janction1lxwfqmcfcwunzchskvc3vrztthkwkgst6zd9y7")
	if err != nil {
		t.Error(err)
	}

	cdc := moduletestutil.MakeTestEncodingConfig().Codec
	signature, publicKey, err := videoRenderingCrypto.SignMessage("/Users/rodrigoacosta/.janctiond", "alice", message, cdc)
	encodedSig := videoRenderingCrypto.EncodeSignatureForCLI(signature)
	encodedPubkey := videoRenderingCrypto.EncodePublicKeyForCLI(publicKey)

	t.Log("Signature:", encodedSig, "pubKey:", encodedPubkey)
	sig, err := videoRenderingCrypto.DecodeSignatureFromCLI(encodedSig)
	if err != nil {
		t.Error(err)
	}
	pubkey, err := videoRenderingCrypto.DecodePublicKeyFromCLI(encodedPubkey)
	if err != nil {
		t.Error(err)
	}
	valid := pubkey.VerifySignature(message, sig)
	if !valid {
		t.Error("Signature is not valid")
	} else {
		t.Log("signature is valid!")
	}
}

func TestVerifySignature(t *testing.T) {
	message, _ := videoRenderingCrypto.GenerateSignableMessage("QmSxV8oPyTQjQ9BuKRJa3Tb3NsxoLSbzNvTGLRjPLjuhYq", "janction13aqlsjwfwfxy0k2kae2qxf62alual7f9r4p0um")
	signature := "KUN4UfDqSMko6j+f63YBgp9pFQarpK0yVkW0eLP5Xkk9nanxqsLSY2IFZ5TE/sfbW4iOlkez2Bb0uBH4jib9jw=="
	publicKey := "AqyMQ9z9mnfnKX9E4na1xKl/5QuvmOyzhUyGXDRMMSC0"

	pk, _ := videoRenderingCrypto.DecodePublicKeyFromCLI(publicKey)

	sig, _ := videoRenderingCrypto.DecodeSignatureFromCLI(signature)
	valid := pk.VerifySignature(message, sig)
	if !valid {
		t.Error("Signature Not valid")
	} else {
		t.Log("Signature is valid")
	}
}
