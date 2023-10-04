package types

import (
	context "context"

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
	SendCoins(ctx context.Context, fromAddr, toAddr sdk.AccAddress, amt sdk.Coins) error
	GetBalance(ctx context.Context, addr sdk.AccAddress, denom string) sdk.Coin
}

// DistributionKeeper defines the expected API contract for the x/distribution
// module.
//
//go:generate mockery --name DistributionKeeper --output ./mocks --outpkg mocks --case underscore
type DistributionKeeper interface {
	GetPreviousProposerConsAddr(ctx context.Context) (sdk.ConsAddress, error)
}

// StakingKeeper defines the expected API contract for the x/staking module.
//
//go:generate mockery --name StakingKeeper --output ./mocks --outpkg mocks --case underscore
type StakingKeeper interface {
	GetValidatorByConsAddr(context.Context, sdk.ConsAddress) (stakingtypes.Validator, error)
}

// RewardsAddressProvider is an interface that provides an address where proposer/subset of auction profits are sent.
//
//go:generate mockery --name RewardsAddressProvider --output ./mocks --outpkg mocks --case underscore
type RewardsAddressProvider interface {
	GetRewardsAddress(context sdk.Context) (sdk.AccAddress, error)
}
