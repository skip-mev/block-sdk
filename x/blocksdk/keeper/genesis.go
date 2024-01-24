package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/skip-mev/block-sdk/v2/x/blocksdk/types"
)

// InitGenesis initializes the blocksdk module's state from a given genesis state.
func (k *Keeper) InitGenesis(ctx sdk.Context, gs types.GenesisState) {
	for _, lane := range gs.Lanes {
		k.setLane(ctx, lane)
	}
}

// ExportGenesis returns a GenesisState for a given context.
func (k *Keeper) ExportGenesis(ctx sdk.Context) *types.GenesisState {
	lanes := k.GetLanes(ctx)

	return &types.GenesisState{Lanes: lanes}
}
