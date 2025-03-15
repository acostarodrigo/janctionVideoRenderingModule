package videoRenderingCrypto

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	"github.com/cosmos/cosmos-sdk/crypto/types"
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
		videoRenderingLogger.Logger.Info("No keys found in keyring at dir %s", rootDir)
	} else {
		videoRenderingLogger.Logger.Info("Loaded %v keys succesfully", len(keys))
	}

	return kr, nil
}

func GetPublicKey(rootDir, alias string, codec codec.Codec) (types.PubKey, error) {
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

func SignMessage(rootDir, alias string, message []byte, codec codec.Codec) ([]byte, types.PubKey, error) {
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
func VerifyMessage(pubKey types.PubKey, message []byte, signature []byte) bool {
	return pubKey.VerifySignature(message, signature)
}

// extract public key for the specified alias from the Key ring
func ExtractPublicKey(rootDir, alias string, codec codec.Codec) (types.PubKey, error) {
	kr, err := getKeyRing(rootDir, codec)
	if err != nil {
		videoRenderingLogger.Logger.Error("ExtractPublicKey rootDir: %s, alias %s", rootDir, alias)
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

// Generate the message to sign
type SignableMessage struct {
	Cid           string `json:"cid"`
	WorkerAddress string `json:"worker_address"`
}

func GenerateSignableMessage(cid, workerAddr string) ([]byte, error) {
	msg := SignableMessage{WorkerAddress: workerAddr, Cid: cid}

	// Serialize the message using Protobuf
	msgBytes, err := json.Marshal(msg)
	if err != nil {
		return nil, err
	}

	// Hash the serialized message
	hash := sha256.Sum256(msgBytes)
	return hash[:], nil
}

// Convert signature bytes to a Base64 string for CLI usage
func EncodeSignatureForCLI(signature []byte) string {
	return base64.StdEncoding.EncodeToString(signature)
}

// Decode Base64 string back to signature bytes after CLI submission
func DecodeSignatureFromCLI(encodedSig string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(encodedSig)
}

func EncodePublicKeyForCLI(publicKey types.PubKey) string {
	return base64.StdEncoding.EncodeToString(publicKey.Bytes())
}

func DecodePublicKeyFromCLI(encodedPubKey string) (types.PubKey, error) {
	decoded, err := base64.StdEncoding.DecodeString(encodedPubKey)
	if err != nil {
		return nil, fmt.Errorf("failed to decode base64 public key: %w", err)
	}
	return fromBytes(decoded)
}

// FromBytes converts a byte slice back to a types.PubKey
func fromBytes(pubKeyBytes []byte) (types.PubKey, error) {
	// Create the pubKey from the byte slice (secp256k1 in this case)
	var _ types.PubKey = (*secp256k1.PubKey)(nil)
	pubKey := &secp256k1.PubKey{Key: pubKeyBytes}
	return pubKey, nil
}
