package videoRendering

import (
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// DefaultParams returns default module parameters.
func DefaultParams() Params {
	return Params{
		// Set default values here.
		MinWorkerStaking:    &sdk.Coin{Denom: "jct", Amount: math.NewInt(1000000)},
		MaxWorkersPerThread: 2,
		MinValidators:       2,
	}
}

// Validate does the sanity check on the params.
func (p Params) Validate() error {
	// Sanity check goes here.

	// We can't have more validators that the amount of workers allowed per thread
	if p.MinValidators > p.MaxWorkersPerThread {
		// error
	}

	// if any of the values is zero thats another mistake
	return nil
}
