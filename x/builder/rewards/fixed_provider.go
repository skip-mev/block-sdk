package rewards

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/skip-mev/pob/x/builder/types"
)

var _ types.RewardsAddressProvider = (*FixedAddressRewardsAddressProvider)(nil)

// FixedAddressRewardsAddressProvider provides a portion of
// auction profits to a fixed address (i.e. the proposer portion).
// This is useful for chains that do not have a distribution module.
type FixedAddressRewardsAddressProvider struct {
	rewardsAddress sdk.AccAddress
}

// NewFixedAddressRewardsAddressProvider creates a reward provider for a fixed address.
func NewFixedAddressRewardsAddressProvider(
	rewardsAddress sdk.AccAddress,
) *FixedAddressRewardsAddressProvider {
	return &FixedAddressRewardsAddressProvider{
		rewardsAddress: rewardsAddress,
	}
}

func (p *FixedAddressRewardsAddressProvider) GetRewardsAddress(_ sdk.Context) (sdk.AccAddress, error) {
	if p.rewardsAddress.Empty() {
		return nil, fmt.Errorf("rewards address is empty")
	}

	return p.rewardsAddress, nil
}
