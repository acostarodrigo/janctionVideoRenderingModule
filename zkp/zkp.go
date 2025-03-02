package zkp

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"math/big"
	"os"
	"path/filepath"

	"github.com/consensys/gnark-crypto/ecc"
	"github.com/consensys/gnark/backend/groth16"
	"github.com/consensys/gnark/frontend"
	"github.com/consensys/gnark/frontend/cs/r1cs"
)

// Circuit defining the proof logic
type FrameProofCircuit struct {
	ProvidedCID frontend.Variable `gnark:",public"` // CID provided as public input
	NodeAddress frontend.Variable `gnark:",public"` // Node address as public input
	ComputedCID frontend.Variable // CID computed from the file hash
}

func (c *FrameProofCircuit) Define(api frontend.API) error {
	// Ensure the computed hash matches the provided one
	api.AssertIsEqual(c.ComputedCID, c.ProvidedCID)
	return nil
}

// InitGnark generates proving and verification keys and returns the proving key path
func InitGnark(path string) (string, string, error) {
	r1cs, err := frontend.Compile(ecc.BN254.ScalarField(), r1cs.NewBuilder, &FrameProofCircuit{})
	if err != nil {
		return "", "", fmt.Errorf("failed to compile circuit: %w", err)
	}

	provingKey, verifyingKey, err := groth16.Setup(r1cs)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate keys: %w", err)
	}

	provingKeyPath := filepath.Join(path, "proving_key.pk")
	verifyingKeyPath := filepath.Join(path, "verifying_key.vk")

	pkFile, err := os.Create(provingKeyPath)
	if err != nil {
		return "", "", fmt.Errorf("failed to create proving key file: %w", err)
	}
	defer pkFile.Close()

	vkFile, err := os.Create(verifyingKeyPath)
	if err != nil {
		return "", "", fmt.Errorf("failed to create verifying key file: %w", err)
	}
	defer vkFile.Close()

	if _, err := provingKey.WriteTo(pkFile); err != nil {
		return "", "", fmt.Errorf("failed to write proving key: %w", err)
	}

	if _, err := verifyingKey.WriteTo(vkFile); err != nil {
		return "", "", fmt.Errorf("failed to write verifying key: %w", err)
	}

	return provingKeyPath, verifyingKeyPath, nil
}

// Function to generate ZK proof after rendering a frame
func GenerateFrameProof(cid string, nodeAddr string, provingKeyPath string) (string, error) {
	// Convert CID and Node Address hex string to big.Int
	cidBigInt := new(big.Int)
	cidBigInt.SetString(cid, 16)

	nodeAddrBigInt := new(big.Int)
	nodeAddrBigInt.SetString(nodeAddr, 16)

	// Step 1: Build the witness
	assignment := FrameProofCircuit{
		ProvidedCID: frontend.Variable(cidBigInt),
		NodeAddress: frontend.Variable(nodeAddrBigInt),
		ComputedCID: frontend.Variable(cidBigInt), // Simulating computed hash
	}

	// Step 2: Compile circuit
	r1cs, err := frontend.Compile(ecc.BN254.ScalarField(), r1cs.NewBuilder, &FrameProofCircuit{})
	if err != nil {
		return "", fmt.Errorf("failed to compile circuit: %w", err)
	}

	// Step 3: Load the proving key
	provingKeyFile, err := os.Open(provingKeyPath)
	if err != nil {
		return "", fmt.Errorf("failed to open proving key file: %w", err)
	}
	defer provingKeyFile.Close()

	provingKey := groth16.NewProvingKey(ecc.BN254)
	if _, err := provingKey.ReadFrom(provingKeyFile); err != nil {
		return "", fmt.Errorf("failed to read proving key: %w", err)
	}

	// Step 4: Create witness
	witness, err := frontend.NewWitness(&assignment, ecc.BN254.ScalarField())
	if err != nil {
		return "", fmt.Errorf("failed to create witness: %w", err)
	}

	// Step 5: Generate proof
	proof, err := groth16.Prove(r1cs, provingKey, witness)
	if err != nil {
		return "", fmt.Errorf("failed to generate proof: %w", err)
	}

	// Step 6: Serialize proof using WriteTo
	buf := new(bytes.Buffer)
	if _, err := proof.WriteTo(buf); err != nil {
		return "", fmt.Errorf("failed to serialize proof: %w", err)
	}

	proofHex := hex.EncodeToString(buf.Bytes())

	return proofHex, nil
}

// Function to verify the proof
func VerifyFrameProof(proof string, verifyingKeyPath string, fakeCID string, fakeNodeAddr string) error {
	// Convert CID and Node Address hex string to big.Int
	cidBigInt := new(big.Int)
	cidBigInt.SetString(fakeCID, 16)

	nodeAddrBigInt := new(big.Int)
	nodeAddrBigInt.SetString(fakeNodeAddr, 16)

	verifyingKeyFile, err := os.Open(verifyingKeyPath)
	if err != nil {
		return fmt.Errorf("failed to open verifying key file: %w", err)
	}
	defer verifyingKeyFile.Close()

	verifyingKey := groth16.NewVerifyingKey(ecc.BN254)
	if _, err := verifyingKey.ReadFrom(verifyingKeyFile); err != nil {
		return fmt.Errorf("failed to read verifying key: %w", err)
	}

	publicWitness, err := frontend.NewWitness(&FrameProofCircuit{
		ProvidedCID: frontend.Variable(cidBigInt),
		NodeAddress: frontend.Variable(nodeAddrBigInt),
	}, ecc.BN254.ScalarField(), frontend.PublicOnly())
	if err != nil {
		return fmt.Errorf("failed to create public witness: %w", err)
	}

	proofBytes, err := hex.DecodeString(proof)
	if err != nil {
		return fmt.Errorf("failed to decode proof: %w", err)
	}

	proofReader := bytes.NewReader(proofBytes)
	proofStruct := groth16.NewProof(ecc.BN254)
	if _, err := proofStruct.ReadFrom(proofReader); err != nil {
		return fmt.Errorf("failed to read proof: %w", err)
	}

	if err := groth16.Verify(proofStruct, verifyingKey, publicWitness); err != nil {
		return fmt.Errorf("proof verification failed: %w", err)
	}

	return nil
}
