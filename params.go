package videoRendering

// DefaultParams returns default module parameters.
func DefaultParams() Params {
	return Params{
		// Set default values here.
		VideoRenderingKeyName: "videoRenderingKey",
	}
}

// Validate does the sanity check on the params.
func (p Params) Validate() error {
	// Sanity check goes here.
	return nil
}
