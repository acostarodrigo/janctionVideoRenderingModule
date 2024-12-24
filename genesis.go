package videoRendering

// NewGenesisState creates a new genesis state with default values.
func NewGenesisState() *GenesisState {
	return &GenesisState{
		Params:                 DefaultParams(),
		VideoRenderingTaskList: GetEmptyVideoRenderingTaskList(),
		VideoRenderingTaskInfo: &VideoRenderingTaskInfo{NextId: 1},
	}
}

// Validate performs basic genesis state validation returning an error upon any
func (gs *GenesisState) Validate() error {
	if err := gs.Params.Validate(); err != nil {
		return err
	}

	return nil
}
