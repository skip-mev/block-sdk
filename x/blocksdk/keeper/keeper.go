package keeper

import (
	"fmt"

	"cosmossdk.io/log"
	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/skip-mev/block-sdk/v2/x/blocksdk/types"
)

type Keeper struct {
	cdc      codec.BinaryCodec
	storeKey storetypes.StoreKey

	// The address that is capable of executing a message.
	// Typically, this will be the governance module's address.
	authority string
}

// NewKeeper creates a new x/blocksdk keeper.
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
func (k *Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", "x/"+types.ModuleName)
}

// GetAuthority returns the address that is capable of executing a MsgUpdateParams message.
func (k *Keeper) GetAuthority() string {
	return k.authority
}

// AddLane calls SetLane and provides additional stateful checks that the new
// set of lanes will be valid.
func (k *Keeper) AddLane(ctx sdk.Context, lane types.Lane) error {
	currentLanes := k.GetLanes(ctx)
	currentLanes = append(currentLanes, lane)

	// validate new set of lanes
	if err := types.Lanes(currentLanes).ValidateBasic(); err != nil {
		return fmt.Errorf("new lane creates invalid lane configuration: %w", err)
	}

	k.setLane(ctx, lane)
	return nil
}

// setLane sets a lane in the store.
func (k *Keeper) setLane(ctx sdk.Context, lane types.Lane) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyLanes)
	b := k.cdc.MustMarshal(&lane)
	store.Set([]byte(lane.Id), b)
}

// GetLane returns a lane by its id.
func (k *Keeper) GetLane(ctx sdk.Context, id string) (lane types.Lane, err error) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyLanes)
	b := store.Get([]byte(id))
	if b == nil {
		return lane, fmt.Errorf("lane not found for ID %s", id)
	}
	k.cdc.MustUnmarshal(b, &lane)
	return lane, nil
}

// GetLanes returns all lanes.
func (k *Keeper) GetLanes(ctx sdk.Context) (lanes []types.Lane) {
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

// DeleteLane deletes an Lane.
func (k *Keeper) DeleteLane(ctx sdk.Context, id string) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyLanes)
	// Delete the Lane
	store.Delete([]byte(id))
}

// GetParams returns the blocksdk module's parameters.
func (k *Keeper) GetParams(ctx sdk.Context) (types.Params, error) {
	store := ctx.KVStore(k.storeKey)

	key := types.KeyParams
	bz := store.Get(key)

	if len(bz) == 0 {
		return types.Params{}, nil
	}

	params := types.Params{}
	if err := params.Unmarshal(bz); err != nil {
		return types.Params{}, err
	}

	return params, nil
}

// SetParams sets the blocksdk module's parameters.
func (k *Keeper) SetParams(ctx sdk.Context, params types.Params) error {
	store := ctx.KVStore(k.storeKey)

	bz, err := params.Marshal()
	if err != nil {
		return err
	}

	store.Set(types.KeyParams, bz)

	return nil
}
