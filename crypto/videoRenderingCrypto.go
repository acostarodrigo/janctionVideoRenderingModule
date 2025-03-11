package videoRenderingCrypto

import (
	"fmt"

	protocrypto "github.com/cometbft/cometbft/proto/tendermint/crypto"
	"github.com/cosmos/cosmos-sdk/codec"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
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
		videoRenderingLogger.Logger.Info("Loaded %v keys succesfully", len(keys))
	}

	return kr, nil
}

func GetPublicKey(rootDir, alias string, codec codec.Codec) (cryptotypes.PubKey, error) {
	keyRing, err := getKeyRing(rootDir, codec)
	if err != nil {
		videoRenderingLogger.Logger.Error("Unable to load key ring at %s: %s", rootDir, err.Error())
		return nil, err
	}

	k, err := keyRing.Key(alias)
	if err != nil {
		videoRenderingLogger.Logger.Error("unable to load key for %s: %s", alias, err.Error())
		return nil, err
	}
	pk, _ := k.GetPubKey()
	return pk, nil
}

func SignMessage(rootDir, alias string, message []byte, codec codec.Codec) ([]byte, cryptotypes.PubKey, error) {
	keyRing, err := getKeyRing(rootDir, codec)
	if err != nil {
		videoRenderingLogger.Logger.Error("Unable to load key ring at %s: %s", rootDir, err.Error())
		return nil, nil, err
	}

	_, err = keyRing.Key(alias)

	if err != nil {
		videoRenderingLogger.Logger.Error("Key %s not found in keyring: %s", alias, err.Error())
		return nil, nil, err
	}

	signature, pubKey, err := keyRing.Sign(alias, message, signing.SignMode_SIGN_MODE_DIRECT)
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

// extract public key for the specified alias from the Key ring
func ExtractPublicKey(rootDir, alias string, codec codec.Codec) (cryptotypes.PubKey, error) {
	kr, err := getKeyRing(rootDir, codec)
	if err != nil {
		return nil, err
	}

	// / Find the key
	keyInfo, err := kr.Key(alias)
	if err != nil {
		return nil, fmt.Errorf("failed to find key %s: %w", alias, err)
	}

	// Extract public key
	pubKey, err := keyInfo.GetPubKey()
	if err != nil {
		return nil, fmt.Errorf("failed to get public key: %w", err)
	}
	return pubKey, nil
}

func ProtoToPubKey(protoPubKey protocrypto.PublicKey) (cryptotypes.PubKey, error) {
	// Correct way to unmarshal
	pubKey, err := cryptocodec.FromCmtProtoPublicKey(protoPubKey)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal public key: %w", err)
	}
	return pubKey, nil
}

func PubKeyToProto(pk cryptotypes.PubKey) (*protocrypto.PublicKey, error) {
	pubKey, err := cryptocodec.ToCmtProtoPublicKey(pk)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal public key: %w", err)
	}

	return &pubKey, nil
}
