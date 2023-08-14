package blockbuster

import (
	"fmt"

	"cosmossdk.io/log"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
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
		proposal BlockProposal,
		maxTxBytes int64,
	) (txsToInclude [][]byte, txsToRemove []sdk.Tx, err error)

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

	// PrepareLanesHandler wraps all of the lanes' PrepareLane function into a single chained
	// function. You can think of it like an AnteHandler, but for preparing proposals in the
	// context of lanes instead of modules.
	PrepareLanesHandler func(ctx sdk.Context, proposal BlockProposal) (BlockProposal, error)

	// ProcessLanesHandler wraps all of the lanes' ProcessLane functions into a single chained
	// function. You can think of it like an AnteHandler, but for processing proposals in the
	// context of lanes instead of modules.
	ProcessLanesHandler func(ctx sdk.Context, txs []sdk.Tx) (sdk.Context, error)

	// LaneConfig defines the basic functionality needed for a lane.
	LaneConfig struct {
		Logger      log.Logger
		TxEncoder   sdk.TxEncoder
		TxDecoder   sdk.TxDecoder
		AnteHandler sdk.AnteHandler

		// MaxBlockSpace defines the relative percentage of block space that can be
		// used by this lane. NOTE: If this is set to zero, then there is no limit
		// on the number of transactions that can be included in the block for this
		// lane (up to maxTxBytes as provided by the request). This is useful for the default lane.
		MaxBlockSpace math.LegacyDec

		// IgnoreList defines the list of lanes to ignore when processing transactions. This
		// is useful for when you want lanes to exist after the default lane. For example,
		// say there are two lanes: default and free. The free lane should be processed after
		// the default lane. In this case, the free lane should be added to the ignore list
		// of the default lane. Otherwise, the transactions that belong to the free lane
		// will be processed by the default lane (which accepts all transactions by default).
		IgnoreList []Lane

		// MaxTxs sets the maximum number of transactions allowed in the mempool with
		// the semantics:
		// - if MaxTx == 0, there is no cap on the number of transactions in the mempool
		// - if MaxTx > 0, the mempool will cap the number of transactions it stores,
		//   and will prioritize transactions by their priority and sender-nonce
		//   (sequence number) when evicting transactions.
		// - if MaxTx < 0, `Insert` is a no-op.
		MaxTxs int
	}
)

// NewLaneConfig returns a new LaneConfig. This will be embedded in a lane.
func NewBaseLaneConfig(
	logger log.Logger,
	txEncoder sdk.TxEncoder,
	txDecoder sdk.TxDecoder,
	anteHandler sdk.AnteHandler,
	maxBlockSpace math.LegacyDec,
) LaneConfig {
	return LaneConfig{
		Logger:        logger,
		TxEncoder:     txEncoder,
		TxDecoder:     txDecoder,
		AnteHandler:   anteHandler,
		MaxBlockSpace: maxBlockSpace,
	}
}

// ValidateBasic validates the lane configuration.
func (c *LaneConfig) ValidateBasic() error {
	if c.Logger == nil {
		return fmt.Errorf("logger cannot be nil")
	}

	if c.TxEncoder == nil {
		return fmt.Errorf("tx encoder cannot be nil")
	}

	if c.TxDecoder == nil {
		return fmt.Errorf("tx decoder cannot be nil")
	}

	if c.MaxBlockSpace.IsNil() || c.MaxBlockSpace.IsNegative() || c.MaxBlockSpace.GT(math.LegacyOneDec()) {
		return fmt.Errorf("max block space must be set to a value between 0 and 1")
	}

	return nil
}

// NoOpPrepareLanesHandler returns a no-op prepare lanes handler.
// This should only be used for testing.
func NoOpPrepareLanesHandler() PrepareLanesHandler {
	return func(ctx sdk.Context, proposal BlockProposal) (BlockProposal, error) {
		return proposal, nil
	}
}

// NoOpPrepareLaneHandler returns a no-op prepare lane handler.
// This should only be used for testing.
func NoOpPrepareLaneHandler() PrepareLaneHandler {
	return func(ctx sdk.Context, proposal BlockProposal, maxTxBytes int64) (txsToInclude [][]byte, txsToRemove []sdk.Tx, err error) {
		return nil, nil, nil
	}
}

// PanicPrepareLaneHandler returns a prepare lane handler that panics.
// This should only be used for testing.
func PanicPrepareLaneHandler() PrepareLaneHandler {
	return func(sdk.Context, BlockProposal, int64) (txsToInclude [][]byte, txsToRemove []sdk.Tx, err error) {
		panic("panic prepare lanes handler")
	}
}

// NoOpProcessLanesHandler returns a no-op process lanes handler.
// This should only be used for testing.
func NoOpProcessLanesHandler() ProcessLanesHandler {
	return func(ctx sdk.Context, txs []sdk.Tx) (sdk.Context, error) {
		return ctx, nil
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
