package block

import (
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkmempool "github.com/cosmos/cosmos-sdk/types/mempool"

	"github.com/skip-mev/block-sdk/block/proposals"
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
		partialProposal [][]byte,
		next ProcessLanesHandler,
	) (proposals.Proposal, error)

	// GetMaxBlockSpace returns the max block space for the lane as a relative percentage.
	GetMaxBlockSpace() math.LegacyDec

	// SetMaxBlockSpace sets the max block space for the lane as a relative percentage.
	SetMaxBlockSpace(math.LegacyDec)

	// Name returns the name of the lane.
	Name() string

	// SetAnteHandler sets the lane's antehandler.
	SetAnteHandler(antehander sdk.AnteHandler)

	// SetIgnoreList sets the lanes that should be ignored by this lane.
	SetIgnoreList(ignoreList []Lane)

	// Match determines if a transaction belongs to this lane.
	Match(ctx sdk.Context, tx sdk.Tx) bool
}

// FindLane finds a Lanes from in an array of Lanes and returns it and its index if found.
// Returns nil, 0 and false if not found.
func FindLane(lanes []Lane, name string) (Lane, int, bool) {
	for i, lane := range lanes {
		if lane.Name() == name {
			return lane, i, true
		}
	}

	return nil, 0, false
}
