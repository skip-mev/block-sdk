package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/skip-mev/pob/x/builder/types"
)

// InitGenesis initializes the builder module's state from a given genesis state.
func (k Keeper) InitGenesis(ctx sdk.Context, gs types.GenesisState) {
	// Set the builder module's parameters.
	if err := k.SetParams(ctx, gs.Params); err != nil {
		panic(err)
	}
}

// ExportGenesis returns a GenesisState for a given context.
func (k Keeper) ExportGenesis(ctx sdk.Context) *types.GenesisState {
	// Get the builder module's parameters.
	params, err := k.GetParams(ctx)
	if err != nil {
		panic(err)
	}

	return types.NewGenesisState(params)
}
