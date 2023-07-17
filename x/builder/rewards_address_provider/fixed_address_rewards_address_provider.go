package rewardsaddressprovider

import (
	"cosmossdk.io/depinject"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/skip-mev/pob/x/builder/types"
)

// FixedAddressRewardsAddressProvider provides auction profits to a fixed address
type FixedAddressRewardsAddressProvider struct {
	rewardsAddress sdk.AccAddress
}

// NewFixedAddressRewardsAddressProvider creates a reward provider for a fixed address.
func NewFixedAddressRewardsAddressProvider(
	rewardsAddress sdk.AccAddress,
) types.RewardsAddressProvider {
	return &FixedAddressRewardsAddressProvider{
		rewardsAddress: rewardsAddress,
	}
}

func (p *FixedAddressRewardsAddressProvider) GetRewardsAddress(_ sdk.Context) sdk.AccAddress {
	return p.rewardsAddress
}

// Dependency injection

type FixedAddressDepInjectInput struct {
	depinject.In

	AccountKeeper types.AccountKeeper
}

type FixedAddressDepInjectOutput struct {
	depinject.Out

	RewardsAddressProvider types.RewardsAddressProvider
}

func ProvideModuleAddress(in FixedAddressDepInjectInput) FixedAddressDepInjectOutput {
	rewardAddressProvider := NewFixedAddressRewardsAddressProvider(
		in.AccountKeeper.GetModuleAddress(types.ModuleName),
	)

	return FixedAddressDepInjectOutput{RewardsAddressProvider: rewardAddressProvider}
}
