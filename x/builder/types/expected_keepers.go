package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

// BankKeeper defines the expected API contract for the x/auth module.
type AccountKeeper interface {
	GetModuleAddress(moduleName string) sdk.AccAddress
}

// BankKeeper defines the expected API contract for the x/bank module.
type BankKeeper interface {
	SendCoins(ctx sdk.Context, fromAddr sdk.AccAddress, toAddr sdk.AccAddress, amt sdk.Coins) error
	GetBalance(ctx sdk.Context, addr sdk.AccAddress, denom string) sdk.Coin
}

// DistributionKeeper defines the expected API contract for the x/distribution
// module.
type DistributionKeeper interface {
	GetPreviousProposerConsAddr(ctx sdk.Context) sdk.ConsAddress
}

// StakingKeeper defines the expected API contract for the x/staking module.
type StakingKeeper interface {
	ValidatorByConsAddr(sdk.Context, sdk.ConsAddress) stakingtypes.ValidatorI
}

// RewardsAddressProvider is an interface that provides an address where proposer/subset of auction profits are sent.
type RewardsAddressProvider interface {
	GetRewardsAddress(context sdk.Context) (sdk.AccAddress, error)
}
