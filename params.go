package videoRendering

import (
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// DefaultParams returns default module parameters.
func DefaultParams() Params {
	return Params{
		// Set default values here.
		MinWorkerStaking:    &sdk.Coin{Denom: "jct", Amount: math.NewInt(100)},
		MaxWorkersPerThread: 2,
	}
}

// Validate does the sanity check on the params.
func (p Params) Validate() error {
	// Sanity check goes here.
	return nil
}
