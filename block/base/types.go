package base

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/skip-mev/block-sdk/block/proposals"
)

type (
	// MatchHandler is utilized to determine if a transaction should be included in the lane. This
	// function can be a stateless or stateful check on the transaction.
	MatchHandler func(ctx sdk.Context, tx sdk.Tx) bool

	// PrepareLaneHandler is responsible for preparing transactions to be included in the block from a
	// given lane. Given a lane, this function should return the transactions to include in the block,
	// the transactions that must be removed from the lane, and an error if one occurred.
	PrepareLaneHandler func(
		ctx sdk.Context,
		proposal proposals.Proposal,
		limit proposals.LaneLimits,
	) (txsToInclude []sdk.Tx, txsToRemove []sdk.Tx, err error)

	// ProcessLaneHandler is responsible for processing transactions that are included in a block and
	// belong to a given lane. This handler must return an error if the transactions are not correctly
	// ordered, do not belong to this lane, or any other relevant error.
	ProcessLaneHandler func(ctx sdk.Context, partialProposal []sdk.Tx) error
)

// NoOpPrepareLaneHandler returns a no-op prepare lane handler.
// This should only be used for testing.
func NoOpPrepareLaneHandler() PrepareLaneHandler {
	return func(sdk.Context, proposals.Proposal, proposals.LaneLimits) ([]sdk.Tx, []sdk.Tx, error) {
		return nil, nil, nil
	}
}

// PanicPrepareLaneHandler returns a prepare lane handler that panics.
// This should only be used for testing.
func PanicPrepareLaneHandler() PrepareLaneHandler {
	return func(sdk.Context, proposals.Proposal, proposals.LaneLimits) ([]sdk.Tx, []sdk.Tx, error) {
		panic("panic prepare lanes handler")
	}
}

// NoOpProcessLaneHandler returns a no-op process lane handler.
// This should only be used for testing.
func NoOpProcessLaneHandler() ProcessLaneHandler {
	return func(sdk.Context, []sdk.Tx) error {
		return nil
	}
}

// PanicProcessLanesHandler returns a process lanes handler that panics.
// This should only be used for testing.
func PanicProcessLaneHandler() ProcessLaneHandler {
	return func(sdk.Context, []sdk.Tx) error {
		panic("panic process lanes handler")
	}
}
