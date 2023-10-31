package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/skip-mev/block-sdk/x/blocksdk/types"
)

// InitGenesis initializes the auction module's state from a given genesis state.
func (k Keeper) InitGenesis(ctx sdk.Context, gs types.GenesisState) {

}

// ExportGenesis returns a GenesisState for a given context.
func (k Keeper) ExportGenesis(ctx sdk.Context) *types.GenesisState {
	return types.NewGenesisState()
}
