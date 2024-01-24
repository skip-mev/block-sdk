package block

import (
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkmempool "github.com/cosmos/cosmos-sdk/types/mempool"

	"github.com/skip-mev/block-sdk/v2/block/proposals"
	"github.com/skip-mev/block-sdk/v2/block/utils"
)

// LaneMempool defines the interface a lane's mempool should implement. The basic API
// is the same as the sdk.Mempool, but it also includes a Compare function that is used
// to determine the relative priority of two transactions belonging in the same lane.
//
//go:generate mockery --name LaneMempool --output ./mocks --outpkg mocks --case underscore
type LaneMempool interface {
	sdkmempool.Mempool

	// Compare determines the relative priority of two transactions belonging in the same lane. Compare
	// will return -1 if this transaction has a lower priority than the other transaction, 0 if they have
	// the same priority, and 1 if this transaction has a higher priority than the other transaction.
	Compare(ctx sdk.Context, this, other sdk.Tx) (int, error)

	// Contains returns true if the transaction is contained in the mempool.
	Contains(tx sdk.Tx) bool

	// Priority returns the priority of a transaction that belongs to this lane.
	Priority(ctx sdk.Context, tx sdk.Tx) any
}

// Lane defines an interface used for matching transactions to lanes, storing transactions,
// and constructing partial blocks.
//
//go:generate mockery --name Lane --output ./mocks --outpkg mocks --case underscore
type Lane interface {
	LaneMempool

	// PrepareLane builds a portion of the block. It inputs the current context, proposal, and a
	// function to call the next lane in the chain. This handler should update the context as needed
	// and add transactions to the proposal. Note, the lane should only add transactions up to the
	// max block space for the lane.
	PrepareLane(
		ctx sdk.Context,
		proposal proposals.Proposal,
		next PrepareLanesHandler,
	) (proposals.Proposal, error)

	// ProcessLane verifies this lane's portion of a proposed block. It inputs the current context,
	// proposal, transactions that belong to this lane, and a function to call the next lane in the
	// chain. This handler should update the context as needed and add transactions to the proposal.
	// The entire process lane chain should end up constructing the same proposal as the prepare lane
	// chain.
	ProcessLane(
		ctx sdk.Context,
		proposal proposals.Proposal,
		txs []sdk.Tx,
		next ProcessLanesHandler,
	) (proposals.Proposal, error)

	// GetMaxBlockSpace returns the max block space for the lane as a relative percentage.
	GetMaxBlockSpace() math.LegacyDec

	// Name returns the name of the lane.
	Name() string

	// Match determines if a transaction belongs to this lane.
	Match(ctx sdk.Context, tx sdk.Tx) bool

	// GetTxInfo returns various information about the transaction that
	// belongs to the lane including its priority, signer's, sequence number,
	// size and more.
	GetTxInfo(ctx sdk.Context, tx sdk.Tx) (utils.TxWithInfo, error)
}
