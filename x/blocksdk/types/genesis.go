package types

import (
	"encoding/json"
	"github.com/cosmos/cosmos-sdk/codec"
)

// NewGenesisState creates a new GenesisState instance.
func NewGenesisState() *GenesisState {
	return &GenesisState{}
}

// DefaultGenesisState returns the default GenesisState instance.
func DefaultGenesisState() *GenesisState {
	return &GenesisState{}
}

// Validate performs basic validation of the blocksdk module genesis state.
func (gs GenesisState) Validate() error {
	return nil
}

// GetGenesisStateFromAppState returns x/blocksdk GenesisState given raw application
// genesis state.
func GetGenesisStateFromAppState(cdc codec.Codec, appState map[string]json.RawMessage) GenesisState {
	var genesisState GenesisState

	if appState[ModuleName] != nil {
		cdc.MustUnmarshalJSON(appState[ModuleName], &genesisState)
	}

	return genesisState
}
