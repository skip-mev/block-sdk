package blockbuster

import (
	"fmt"

	"cosmossdk.io/log"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkmempool "github.com/cosmos/cosmos-sdk/types/mempool"
)

type (
	// PrepareLanesHandler wraps all of the lanes Prepare function into a single chained
	// function. You can think of it like an AnteHandler, but for preparing proposals in the
	// context of lanes instead of modules.
	PrepareLanesHandler func(ctx sdk.Context, proposal BlockProposal) (BlockProposal, error)

	// ProcessLanesHandler wraps all of the lanes Process functions into a single chained
	// function. You can think of it like an AnteHandler, but for processing proposals in the
	// context of lanes instead of modules.
	ProcessLanesHandler func(ctx sdk.Context, txs []sdk.Tx) (sdk.Context, error)

	// BaseLaneConfig defines the basic functionality needed for a lane.
	BaseLaneConfig struct {
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
		// will be processed by the default lane.
		IgnoreList []Lane
	}

	// Lane defines an interface used for block construction
	Lane interface {
		sdkmempool.Mempool

		// Name returns the name of the lane.
		Name() string

		// Match determines if a transaction belongs to this lane.
		Match(ctx sdk.Context, tx sdk.Tx) bool

		// VerifyTx verifies the transaction belonging to this lane.
		VerifyTx(ctx sdk.Context, tx sdk.Tx) error

		// Contains returns true if the mempool/lane contains the given transaction.
		Contains(tx sdk.Tx) bool

		// PrepareLane builds a portion of the block. It inputs the maxTxBytes that can be
		// included in the proposal for the given lane, the partial proposal, and a function
		// to call the next lane in the chain. The next lane in the chain will be called with
		// the updated proposal and context.
		PrepareLane(ctx sdk.Context, proposal BlockProposal, maxTxBytes int64, next PrepareLanesHandler) (BlockProposal, error)

		// ProcessLaneBasic validates that transactions belonging to this lane are not misplaced
		// in the block proposal.
		ProcessLaneBasic(ctx sdk.Context, txs []sdk.Tx) error

		// ProcessLane verifies this lane's portion of a proposed block. It inputs the transactions
		// that may belong to this lane and a function to call the next lane in the chain. The next
		// lane in the chain will be called with the updated context and filtered down transactions.
		ProcessLane(ctx sdk.Context, proposalTxs []sdk.Tx, next ProcessLanesHandler) (sdk.Context, error)

		// SetAnteHandler sets the lane's antehandler.
		SetAnteHandler(antehander sdk.AnteHandler)

		// Logger returns the lane's logger.
		Logger() log.Logger

		// GetMaxBlockSpace returns the max block space for the lane as a relative percentage.
		GetMaxBlockSpace() math.LegacyDec
	}
)

// NewLaneConfig returns a new LaneConfig. This will be embedded in a lane.
func NewBaseLaneConfig(
	logger log.Logger,
	txEncoder sdk.TxEncoder,
	txDecoder sdk.TxDecoder,
	anteHandler sdk.AnteHandler,
	maxBlockSpace math.LegacyDec,
) BaseLaneConfig {
	return BaseLaneConfig{
		Logger:        logger,
		TxEncoder:     txEncoder,
		TxDecoder:     txDecoder,
		AnteHandler:   anteHandler,
		MaxBlockSpace: maxBlockSpace,
	}
}

// ValidateBasic validates the lane configuration.
func (c *BaseLaneConfig) ValidateBasic() error {
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
