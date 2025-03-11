package videoRenderingCrypto

import (
	"crypto/sha256"
	"fmt"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	"github.com/janction/videoRendering/videoRenderingLogger"
)

// Loads the janctiond Keyring
func getKeyRing(rootDir string, codec codec.Codec) (keyring.Keyring, error) {
	// Use BackendFile to access persistent keys stored in ~/.janctiond/keyring-file
	kr, err := keyring.New("janction", keyring.BackendTest, rootDir, nil, codec)
	if err != nil {
		videoRenderingLogger.Logger.Error("Unable to load keyring at %s: %s", rootDir, err.Error())
		return nil, err
	}

	// Check if keys exist
	keys, err := kr.List()
	if err != nil {
		videoRenderingLogger.Logger.Error("Error listing keys: %s", err.Error())
		return nil, err
	}

	if len(keys) == 0 {
		videoRenderingLogger.Logger.Info("No keys found in keyring")
	} else {
		videoRenderingLogger.Logger.Info("Loaded keys:", keys)
	}

	return kr, nil
}

func SignMessage(rootDir, alias string, message []byte, codec codec.Codec) ([]byte, cryptotypes.PubKey, error) {
	keyRing, err := getKeyRing(rootDir, codec)
	if err != nil {
		videoRenderingLogger.Logger.Error("Unable to load key ring at %s: %s", rootDir, err.Error())
		return nil, nil, err
	}
	keys, err := keyRing.List()
	if err != nil {
		videoRenderingLogger.Logger.Error("Error listing keys: %s", err.Error())
	} else {
		if len(keys) == 0 {
			videoRenderingLogger.Logger.Error("keyring doesn't contain any keys")
			return nil, nil, fmt.Errorf("keyring doesn't contain any keys")
		}
	}

	_, err = keyRing.Key(alias)
	if err != nil {
		videoRenderingLogger.Logger.Error("Key %s not found in keyring: %s", alias, err.Error())
		return nil, nil, err
	}

	hash := sha256.Sum256(message)
	signature, pubKey, err := keyRing.Sign(alias, hash[:], signing.SignMode_SIGN_MODE_DIRECT)
	if err != nil {
		videoRenderingLogger.Logger.Error("Error signing message: %s", err.Error())
		return nil, nil, err
	}
	return signature, pubKey, err

}

// checks if the signed message, correspond to the publick key
func VerifyMessage(pubKey cryptotypes.PubKey, message []byte, signature []byte) bool {
	return pubKey.VerifySignature(message, signature)
}
