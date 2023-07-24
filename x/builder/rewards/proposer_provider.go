package rewards

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/skip-mev/pob/x/builder/types"
)

// ProposerRewardsAddressProvider provides a portion of
// auction profits to the block proposer.
type ProposerRewardsAddressProvider struct {
	distrKeeper   types.DistributionKeeper
	stakingKeeper types.StakingKeeper
}

// NewProposerRewardsAddressProvider creates a reward provider for block proposers.
func NewProposerRewardsAddressProvider(
	distrKeeper types.DistributionKeeper,
	stakingKeeper types.StakingKeeper,
) types.RewardsAddressProvider {
	return &ProposerRewardsAddressProvider{
		distrKeeper:   distrKeeper,
		stakingKeeper: stakingKeeper,
	}
}

func (p *ProposerRewardsAddressProvider) GetRewardsAddress(ctx sdk.Context) sdk.AccAddress {
	prevPropConsAddr := p.distrKeeper.GetPreviousProposerConsAddr(ctx)
	prevProposer := p.stakingKeeper.ValidatorByConsAddr(ctx, prevPropConsAddr)

	return sdk.AccAddress(prevProposer.GetOperator())
}
