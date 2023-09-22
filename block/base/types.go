package base

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/skip-mev/block-sdk/block"
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
		proposal block.BlockProposal,
		limit block.LaneLimits,
	) (txsToInclude []sdk.Tx, txsToRemove []sdk.Tx, err error)

	// ProcessLaneHandler is responsible for processing transactions that are included in a block and
	// belong to a given lane. ProcessLaneHandler is executed after CheckOrderHandler so the transactions
	// passed into this function SHOULD already be in order respecting the ordering rules of the lane and
	// respecting the ordering rules of mempool relative to the lanes it has.
	ProcessLaneHandler func(ctx sdk.Context, txs []sdk.Tx) ([]sdk.Tx, error)

	// CheckOrderHandler is responsible for checking the order of transactions that belong to a given
	// lane. This handler should be used to verify that the ordering of transactions passed into the
	// function respect the ordering logic of the lane (if any transactions from the lane are included).
	// This function should also ensure that transactions that belong to this lane are contiguous and do
	// not have any transactions from other lanes in between them.
	CheckOrderHandler func(ctx sdk.Context, txs []sdk.Tx) error
)

// NoOpPrepareLaneHandler returns a no-op prepare lane handler.
// This should only be used for testing.
func NoOpPrepareLaneHandler() PrepareLaneHandler {
	return func(sdk.Context, block.BlockProposal, block.LaneLimits) ([]sdk.Tx, []sdk.Tx, error) {
		return nil, nil, nil
	}
}

// PanicPrepareLaneHandler returns a prepare lane handler that panics.
// This should only be used for testing.
func PanicPrepareLaneHandler() PrepareLaneHandler {
	return func(sdk.Context, block.BlockProposal, block.LaneLimits) ([]sdk.Tx, []sdk.Tx, error) {
		panic("panic prepare lanes handler")
	}
}

// NoOpProcessLaneHandler returns a no-op process lane handler.
// This should only be used for testing.
func NoOpProcessLaneHandler() ProcessLaneHandler {
	return func(ctx sdk.Context, txs []sdk.Tx) ([]sdk.Tx, error) {
		return txs, nil
	}
}

// PanicProcessLanesHandler returns a process lanes handler that panics.
// This should only be used for testing.
func PanicProcessLaneHandler() ProcessLaneHandler {
	return func(sdk.Context, []sdk.Tx) ([]sdk.Tx, error) {
		panic("panic process lanes handler")
	}
}
