package abci

import (
	cometabci "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// ValidateVoteExtensionsFn defines the function for validating vote extensions. This
// function is not explicitly used to validate the oracle data but rather that
// the signed vote extensions included in the proposal are valid and provide
// a supermajority of vote extensions for the current block. This method is
// expected to be used in ProcessProposal, the expected ctx is the ProcessProposalState's ctx.
type ValidateVoteExtensionsFn func(ctx sdk.Context, currentHeight int64, extendedCommitInfo cometabci.ExtendedCommitInfo) error

// NoOpValidateVoteExtensionsFn returns a ValidateVoteExtensionsFn that does nothing. This should NOT
// be used in production.
func NoOpValidateVoteExtensionsFn() ValidateVoteExtensionsFn {
	return func(_ sdk.Context, _ int64, _ cometabci.ExtendedCommitInfo) error {
		return nil
	}
}
