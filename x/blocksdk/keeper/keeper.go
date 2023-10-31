package keeper

import (
	"fmt"

	"cosmossdk.io/log"
	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/skip-mev/block-sdk/x/blocksdk/types"
)

type Keeper struct {
	cdc      codec.BinaryCodec
	storeKey storetypes.StoreKey

	// The address that is capable of executing a MsgUpdateParams message.
	// Typically this will be the governance module's address.
	authority string
}

// NewKeeper is a wrapper around NewKeeperWithRewardsAddressProvider for backwards compatibility.
func NewKeeper(
	cdc codec.BinaryCodec,
	storeKey storetypes.StoreKey,

	authority string,
) Keeper {
	return Keeper{
		cdc:      cdc,
		storeKey: storeKey,

		authority: authority,
	}
}

// Logger returns a blocksdk module-specific logger.
func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", "x/"+types.ModuleName)
}

// GetAuthority returns the address that is capable of executing a MsgUpdateParams message.
func (k Keeper) GetAuthority() string {
	return k.authority
}

// SetLane sets a lane in the store.
func (k Keeper) SetLane(ctx sdk.Context, lane types.Lane) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyLanes)
	b := k.cdc.MustMarshal(&lane)
	store.Set([]byte(lane.Id), b)
}

// GetLane returns a lane by its id.
func (k Keeper) GetLane(ctx sdk.Context, id string) (lane types.Lane, err error) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyLanes)
	b := store.Get([]byte(id))
	if b == nil {
		return lane, fmt.Errorf("lane not found for ID %s", id)
	}
	k.cdc.MustUnmarshal(b, &lane)
	return lane, nil
}

// GetLanes returns all lanes.
func (k Keeper) GetLanes(ctx sdk.Context) (lanes []types.Lane) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyLanes)
	iterator := storetypes.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var lane types.Lane
		k.cdc.MustUnmarshal(iterator.Value(), &lane)
		lanes = append(lanes, lane)
	}

	return
}
