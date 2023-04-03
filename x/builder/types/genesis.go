package types

// NewGenesisState creates a new GenesisState instance.
func NewGenesisState(params Params) *GenesisState {
	return &GenesisState{
		Params: params,
	}
}

// DefaultGenesisState returns the default GenesisState instance.
func DefaultGenesisState() *GenesisState {
	return &GenesisState{
		Params: DefaultParams(),
	}
}

// Validate performs basic validation of the builder module genesis state.
func (gs GenesisState) Validate() error {
	return gs.Params.Validate()
}
