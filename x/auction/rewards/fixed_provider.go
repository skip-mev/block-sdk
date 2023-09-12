package rewards

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
<<<<<<< HEAD:x/builder/rewards/fixed_provider.go
	"github.com/skip-mev/block-sdk/x/builder/types"
=======

	"github.com/skip-mev/block-sdk/x/auction/types"
>>>>>>> 3c6f319 (feat(docs): rename x/builder -> x/auction (#55)):x/auction/rewards/fixed_provider.go
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
