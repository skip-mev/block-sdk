package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

// BankKeeper defines the expected API contract for the x/account module.
//
//go:generate mockery --name AccountKeeper --output ./mocks --outpkg mocks --case underscore
type AccountKeeper interface {
	GetModuleAddress(moduleName string) sdk.AccAddress
}

// BankKeeper defines the expected API contract for the x/bank module.
//
//go:generate mockery --name BankKeeper --output ./mocks --outpkg mocks --case underscore
type BankKeeper interface {
	SendCoins(ctx sdk.Context, fromAddr sdk.AccAddress, toAddr sdk.AccAddress, amt sdk.Coins) error
	GetBalance(ctx sdk.Context, addr sdk.AccAddress, denom string) sdk.Coin
}

// DistributionKeeper defines the expected API contract for the x/distribution
// module.
//
//go:generate mockery --name DistributionKeeper --output ./mocks --outpkg mocks --case underscore
type DistributionKeeper interface {
	GetPreviousProposerConsAddr(ctx sdk.Context) sdk.ConsAddress
}

// StakingKeeper defines the expected API contract for the x/staking module.
//
//go:generate mockery --name StakingKeeper --output ./mocks --outpkg mocks --case underscore
type StakingKeeper interface {
	ValidatorByConsAddr(sdk.Context, sdk.ConsAddress) stakingtypes.ValidatorI
}

// RewardsAddressProvider is an interface that provides an address where proposer/subset of auction profits are sent.
//
//go:generate mockery --name RewardsAddressProvider --output ./mocks --outpkg mocks --case underscore
type RewardsAddressProvider interface {
	GetRewardsAddress(context sdk.Context) (sdk.AccAddress, error)
}
