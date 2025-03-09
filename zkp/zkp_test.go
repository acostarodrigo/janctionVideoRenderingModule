package zkp_test

import (
	"testing"

	"github.com/janction/videoRendering/zkp"
)

// Test function for GenerateFrameProof
// Test function for GenerateFrameProof
func TestGenerateFrameProof(t *testing.T) {
	fakeCID := "aabbccddeeff00112233445566778899" // Example invented CID
	fakeAddress := "cosmosAddress"                // Example invented CID
	err := zkp.InitGnark("./")
	if err != nil {
		t.Fatalf("Failed to initialize gnark: %v", err)
	}

	proof, err := zkp.GenerateFrameProof(fakeCID, fakeAddress, "./proving_key.pk")
	if err != nil {
		t.Fatalf("Failed to generate proof: %v", err)
	}

	if err := zkp.VerifyFrameProof(proof, "./verifying_key.vk", fakeCID, fakeAddress); err != nil {
		t.Fatalf("Proof verification failed: %v", err)
	}

	t.Logf("Generated proof: %s", proof)
}
