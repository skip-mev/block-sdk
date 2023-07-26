package keeper

import (
	"fmt"

	"cosmossdk.io/log"
	"cosmossdk.io/math"

	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/skip-mev/pob/x/builder/rewards"
	"github.com/skip-mev/pob/x/builder/types"
)

type Keeper struct {
	cdc      codec.BinaryCodec
	storeKey storetypes.StoreKey

	bankKeeper             types.BankKeeper
	rewardsAddressProvider types.RewardsAddressProvider

	// The address that is capable of executing a MsgUpdateParams message.
	// Typically this will be the governance module's address.
	authority string
}

// NewKeeper is a wrapper around NewKeeperWithRewardsAddressProvider for backwards compatibility.
func NewKeeper(
	cdc codec.BinaryCodec,
	storeKey storetypes.StoreKey,
	accountKeeper types.AccountKeeper,
	bankKeeper types.BankKeeper,
	distrKeeper types.DistributionKeeper,
	stakingKeeper types.StakingKeeper,
	authority string,
) Keeper {
	// Build a rewards address provider
	rewardsAddressProvider := rewards.NewProposerRewardsAddressProvider(
		distrKeeper,
		stakingKeeper,
	)

	return NewKeeperWithRewardsAddressProvider(
		cdc,
		storeKey,
		accountKeeper,
		bankKeeper,
		rewardsAddressProvider,
		authority,
	)
}

func NewKeeperWithRewardsAddressProvider(
	cdc codec.BinaryCodec,
	storeKey storetypes.StoreKey,
	accountKeeper types.AccountKeeper,
	bankKeeper types.BankKeeper,
	rewardsAddressProvider types.RewardsAddressProvider,
	authority string,
) Keeper {
	// Ensure that the authority address is valid.
	if _, err := sdk.AccAddressFromBech32(authority); err != nil {
		panic(err)
	}

	// Ensure that the builder module account exists.
	if accountKeeper.GetModuleAddress(types.ModuleName) == nil {
		panic("builder module account has not been set")
	}

	return Keeper{
		cdc:                    cdc,
		storeKey:               storeKey,
		bankKeeper:             bankKeeper,
		rewardsAddressProvider: rewardsAddressProvider,
		authority:              authority,
	}
}

// Logger returns a builder module-specific logger.
func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", "x/"+types.ModuleName)
}

// GetAuthority returns the address that is capable of executing a MsgUpdateParams message.
func (k Keeper) GetAuthority() string {
	return k.authority
}

// GetParams returns the builder module's parameters.
func (k Keeper) GetParams(ctx sdk.Context) (types.Params, error) {
	store := ctx.KVStore(k.storeKey)

	key := types.KeyParams
	bz := store.Get(key)

	if len(bz) == 0 {
		return types.Params{}, fmt.Errorf("no params found for the builder module")
	}

	params := types.Params{}
	if err := params.Unmarshal(bz); err != nil {
		return types.Params{}, err
	}

	return params, nil
}

// SetParams sets the builder module's parameters.
func (k Keeper) SetParams(ctx sdk.Context, params types.Params) error {
	store := ctx.KVStore(k.storeKey)

	bz, err := params.Marshal()
	if err != nil {
		return err
	}

	store.Set(types.KeyParams, bz)

	return nil
}

// GetMaxBundleSize returns the maximum number of transactions that can be included in a bundle.
func (k Keeper) GetMaxBundleSize(ctx sdk.Context) (uint32, error) {
	params, err := k.GetParams(ctx)
	if err != nil {
		return 0, err
	}

	return params.MaxBundleSize, nil
}

// GetEscrowAccount returns the builder module's escrow account.
func (k Keeper) GetEscrowAccount(ctx sdk.Context) (sdk.AccAddress, error) {
	params, err := k.GetParams(ctx)
	if err != nil {
		return nil, err
	}

	return params.EscrowAccountAddress, nil
}

// GetReserveFee returns the reserve fee of the builder module.
func (k Keeper) GetReserveFee(ctx sdk.Context) (sdk.Coin, error) {
	params, err := k.GetParams(ctx)
	if err != nil {
		return sdk.Coin{}, err
	}

	return params.ReserveFee, nil
}

// GetMinBidIncrement returns the minimum bid increment for the builder.
func (k Keeper) GetMinBidIncrement(ctx sdk.Context) (sdk.Coin, error) {
	params, err := k.GetParams(ctx)
	if err != nil {
		return sdk.Coin{}, err
	}

	return params.MinBidIncrement, nil
}

// GetProposerFee returns the proposer fee for the builder module.
func (k Keeper) GetProposerFee(ctx sdk.Context) (math.LegacyDec, error) {
	params, err := k.GetParams(ctx)
	if err != nil {
		return math.LegacyZeroDec(), err
	}

	return params.ProposerFee, nil
}

// FrontRunningProtectionEnabled returns true if front-running protection is enabled.
func (k Keeper) FrontRunningProtectionEnabled(ctx sdk.Context) (bool, error) {
	params, err := k.GetParams(ctx)
	if err != nil {
		return false, err
	}

	return params.FrontRunningProtection, nil
}
