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
	signature, publicKey, _ := videoRenderingCrypto.SignMessage("/Users/rodrigoacosta/.janctiond", "alice", message, cdc)
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
	message, _ := videoRenderingCrypto.GenerateSignableMessage("Qma5Keu7JY89US3enHJj2vdhKecNp5NWufU1UBhepyDD4E", "janction1dfr2q9edz0dxd8a5g6p2kwyzaenrk9xgv3zypl")
	signature := "71hVEuYxAfG4xHr4E7GLHSsExRFHNxlC33mLVjWWuBdTrihC0RJq8yLUUqd/va7BUd6fQyONECYTnjz7kNmjwQ=="
	publicKey := "A4sXU6+iMgZbrTDdyFVwQQqmro9QdDqDqwPu+4Fva4zU"

	pk, _ := videoRenderingCrypto.DecodePublicKeyFromCLI(publicKey)

	sig, _ := videoRenderingCrypto.DecodeSignatureFromCLI(signature)
	valid := pk.VerifySignature(message, sig)
	if !valid {
		t.Error("Signature Not valid")
	} else {
		t.Log("Signature is valid")
	}
}

func TestPublicKey(t *testing.T) {
	message, err := videoRenderingCrypto.GenerateSignableMessage("QmRe3MVV1NeF84sgiBCeKBhwDGFVcyLPzcky4fN2cKvTzs", "janction1rkzs8h4w5dj07fhpcc2x607nj5905vd98qyl2u")
	if err != nil {
		t.Error(err)
	}

	cdc := moduletestutil.MakeTestEncodingConfig().Codec
	sig, publicKey, _ := videoRenderingCrypto.SignMessage("/Users/rodrigoacosta/.janctiond", "alice", message, cdc)
	pk, _ := videoRenderingCrypto.GetPublicKey("/Users/rodrigoacosta/.janctiond", "alice", cdc)

	if pk.Address().String() != publicKey.Address().String() {
		t.Error("Public keys are not the same")
	} else {
		t.Logf("pk are equal: %s = %s", publicKey.Address().String(), pk.Address().String())
	}

	if videoRenderingCrypto.EncodePublicKeyForCLI(publicKey) != videoRenderingCrypto.EncodePublicKeyForCLI(pk) {
		t.Error("Not the same")
	} else {
		t.Logf("%s = %s", videoRenderingCrypto.EncodePublicKeyForCLI(publicKey), videoRenderingCrypto.EncodePublicKeyForCLI(pk))
	}

	valid := pk.VerifySignature(message, sig)
	if !valid {
		t.Error("Not valid")
	}

	valid = publicKey.VerifySignature(message, sig)
	if !valid {
		t.Error("Not valid")
	}
}
