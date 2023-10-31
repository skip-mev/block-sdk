package keeper

import (
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/codec"
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
