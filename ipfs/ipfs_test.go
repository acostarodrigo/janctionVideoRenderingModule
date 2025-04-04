package ipfs

import (
	"errors"
	"testing"

	"bou.ke/monkey"
	shell "github.com/ipfs/go-ipfs-api"
)

func TestIPFSGet_WithMonkeyPatch(t *testing.T) {
	// Patch para reemplazar shell.Shell.Get
	patch := monkey.Patch((*shell.Shell).Get, func(_ *shell.Shell, cid string, outDir string) error {
		if cid == "validCID" {
			return nil
		}
		return errors.New("Mock error")
	})
	defer patch.Unpatch()

	path := "/tmp/fakepath"
	tests := []struct {
		name            string
		cid             string
		expectedToError bool
	}{
		{
			name:            "valid",
			cid:             "validCID",
			expectedToError: false,
		},
		{
			name:            "invalid",
			cid:             "invalidCID",
			expectedToError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := IPFSGet(tt.cid, path)
			if (err != nil) != tt.expectedToError {
				t.Errorf("Test %q failed: expectedToError=%v, got err=%v", tt.name, tt.expectedToError, err)
			}
		})
	}
}
