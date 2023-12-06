package base

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/skip-mev/block-sdk/block"
)

// LaneOption defines a function that can be used to set options on a lane.
type LaneOption func(*BaseLane)

// SetAnteHandler sets the ante handler for the lane.
func SetAnteHandler(anteHandler sdk.AnteHandler) LaneOption {
	return func(l *BaseLane) { l.cfg.AnteHandler = anteHandler }
}

// SetPrepareLaneHandler sets the prepare lane handler for the lane. This handler
// is called when a new proposal is being requested and the lane needs to submit
// transactions it wants included in the block.
func SetPrepareLaneHandler(prepareLaneHandler PrepareLaneHandler) LaneOption {
	return func(l *BaseLane) {
		if prepareLaneHandler == nil {
			panic("prepare lane handler cannot be nil")
		}

		l.prepareLaneHandler = prepareLaneHandler
	}
}

// SetProcessLaneHandler sets the process lane handler for the lane. This handler
// is called when a new proposal is being verified and the lane needs to verify
// that the transactions included in the proposal are valid respecting the verification
// logic of the lane.
func SetProcessLaneHandler(processLaneHandler ProcessLaneHandler) LaneOption {
	return func(l *BaseLane) {
		if processLaneHandler == nil {
			panic("process lane handler cannot be nil")
		}

		l.processLaneHandler = processLaneHandler
	}
}

// SetMatchHandler sets the match handler for the lane. This handler is called
// when a new transaction is being submitted to the lane and the lane needs to
// determine if the transaction should be processed by the lane.
func SetMatchHandler(matchHandler MatchHandler) LaneOption {
	return func(l *BaseLane) {
		if matchHandler == nil {
			panic("match handler cannot be nil")
		}

		l.matchHandler = matchHandler
	}
}

// SetMempool sets the mempool for the lane. This mempool is used to store
// transactions that are waiting to be processed.
func SetMempool(mempool block.LaneMempool) LaneOption {
	return func(l *BaseLane) {
		if mempool == nil {
			panic("mempool cannot be nil")
		}

		l.LaneMempool = mempool
	}
}

// SetMempoolWithConfigs sets the mempool for the lane with the given lane config
// and TxPriority struct. This mempool is used to store transactions that are waiting
// to be processed.
func SetMempoolWithConfigs[C comparable](cfg LaneConfig, txPriority TxPriority[C]) LaneOption {
	return func(l *BaseLane) {
		l.LaneMempool = NewMempool(
			txPriority,
			cfg.TxEncoder,
			cfg.SignerExtractor,
			cfg.MaxTxs,
		)
	}
}
