package rewards

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/skip-mev/pob/x/builder/types"
)

var _ types.RewardsAddressProvider = (*ProposerRewardsAddressProvider)(nil)

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
) *ProposerRewardsAddressProvider {
	return &ProposerRewardsAddressProvider{
		distrKeeper:   distrKeeper,
		stakingKeeper: stakingKeeper,
	}
}

func (p *ProposerRewardsAddressProvider) GetRewardsAddress(ctx sdk.Context) (sdk.AccAddress, error) {
	prevPropConsAddr, err := p.distrKeeper.GetPreviousProposerConsAddr(ctx)
	if err != nil {
		return nil, err
	}

	prevProposer, err := p.stakingKeeper.GetValidatorByConsAddr(ctx, prevPropConsAddr)
	if err != nil {
		return nil, err
	}

	return sdk.AccAddress(prevProposer.GetOperator()), nil
}
