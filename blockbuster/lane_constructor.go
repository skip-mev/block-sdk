package blockbuster

import (
	"fmt"

	"cosmossdk.io/log"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

var _ Lane = (*LaneConstructor)(nil)

// LaneConstructor is a generic implementation of a lane. It is meant to be used
// as a base for other lanes to be built on top of. It provides a default
// implementation of the MatchHandler, PrepareLaneHandler, ProcessLaneHandler,
// and CheckOrderHandler. To extend this lane, you must either utilize the default
// handlers or construct your own that you pass into the constructor/setters.
type LaneConstructor struct {
	// cfg stores functionality required to encode/decode transactions, maintains how
	// many transactions are allowed in this lane's mempool, and the amount of block
	// space this lane is allowed to consume.
	cfg LaneConfig

	// laneName is the name of the lane.
	laneName string

	// LaneMempool is the mempool that is responsible for storing transactions
	// that are waiting to be processed.
	LaneMempool

	// matchHandler is the function that determines whether or not a transaction
	// should be processed by this lane.
	matchHandler MatchHandler

	// prepareLaneHandler is the function that is called when a new proposal is being
	// requested and the lane needs to submit transactions it wants included in the block.
	prepareLaneHandler PrepareLaneHandler

	// checkOrderHandler is the function that is called when a new proposal is being
	// verified and the lane needs to verify that the transactions included in the proposal
	// respect the ordering rules of the lane and does not interleave transactions from other lanes.
	checkOrderHandler CheckOrderHandler

	// processLaneHandler is the function that is called when a new proposal is being
	// verified and the lane needs to verify that the transactions included in the proposal
	// are valid respecting the verification logic of the lane.
	processLaneHandler ProcessLaneHandler
}

// NewLaneConstructor returns a new lane constructor. When creating this lane, the type
// of the lane must be specified. The type of the lane is directly associated with the
// type of the mempool that is used to store transactions that are waiting to be processed.
func NewLaneConstructor(
	cfg LaneConfig,
	laneName string,
	laneMempool LaneMempool,
	matchHandlerFn MatchHandler,
) *LaneConstructor {
	lane := &LaneConstructor{
		cfg:          cfg,
		laneName:     laneName,
		LaneMempool:  laneMempool,
		matchHandler: matchHandlerFn,
	}

	if err := lane.ValidateBasic(); err != nil {
		panic(err)
	}

	return lane
}

// ValidateBasic ensures that the lane was constructed properly. In the case that
// the lane was not constructed with proper handlers, default handlers are set.
func (l *LaneConstructor) ValidateBasic() error {
	if err := l.cfg.ValidateBasic(); err != nil {
		return err
	}

	if l.laneName == "" {
		return fmt.Errorf("lane name cannot be empty")
	}

	if l.LaneMempool == nil {
		return fmt.Errorf("lane mempool cannot be nil")
	}

	if l.matchHandler == nil {
		return fmt.Errorf("match handler cannot be nil")
	}

	if l.prepareLaneHandler == nil {
		l.prepareLaneHandler = l.DefaultPrepareLaneHandler()
	}

	if l.processLaneHandler == nil {
		l.processLaneHandler = l.DefaultProcessLaneHandler()
	}

	if l.checkOrderHandler == nil {
		l.checkOrderHandler = l.DefaultCheckOrderHandler()
	}

	return nil
}

// SetPrepareLaneHandler sets the prepare lane handler for the lane. This handler
// is called when a new proposal is being requested and the lane needs to submit
// transactions it wants included in the block.
func (l *LaneConstructor) SetPrepareLaneHandler(prepareLaneHandler PrepareLaneHandler) {
	if prepareLaneHandler == nil {
		panic("prepare lane handler cannot be nil")
	}

	l.prepareLaneHandler = prepareLaneHandler
}

// SetProcessLaneHandler sets the process lane handler for the lane. This handler
// is called when a new proposal is being verified and the lane needs to verify
// that the transactions included in the proposal are valid respecting the verification
// logic of the lane.
func (l *LaneConstructor) SetProcessLaneHandler(processLaneHandler ProcessLaneHandler) {
	if processLaneHandler == nil {
		panic("process lane handler cannot be nil")
	}

	l.processLaneHandler = processLaneHandler
}

// SetCheckOrderHandler sets the check order handler for the lane. This handler
// is called when a new proposal is being verified and the lane needs to verify
// that the transactions included in the proposal respect the ordering rules of
// the lane and does not include transactions from other lanes.
func (l *LaneConstructor) SetCheckOrderHandler(checkOrderHandler CheckOrderHandler) {
	if checkOrderHandler == nil {
		panic("check order handler cannot be nil")
	}

	l.checkOrderHandler = checkOrderHandler
}

// Match returns true if the transaction should be processed by this lane. This
// function first determines if the transaction matches the lane and then checks
// if the transaction is on the ignore list. If the transaction is on the ignore
// list, it returns false.
func (l *LaneConstructor) Match(ctx sdk.Context, tx sdk.Tx) bool {
	return l.matchHandler(ctx, tx) && !l.CheckIgnoreList(ctx, tx)
}

// CheckIgnoreList returns true if the transaction is on the ignore list. The ignore
// list is utilized to prevent transactions that should be considered in other lanes
// from being considered from this lane.
func (l *LaneConstructor) CheckIgnoreList(ctx sdk.Context, tx sdk.Tx) bool {
	for _, lane := range l.cfg.IgnoreList {
		if lane.Match(ctx, tx) {
			return true
		}
	}

	return false
}

// Name returns the name of the lane.
func (l *LaneConstructor) Name() string {
	return l.laneName
}

// SetIgnoreList sets the ignore list for the lane. The ignore list is a list
// of lanes that the lane should ignore when processing transactions.
func (l *LaneConstructor) SetIgnoreList(lanes []Lane) {
	l.cfg.IgnoreList = lanes
}

// SetAnteHandler sets the ante handler for the lane.
func (l *LaneConstructor) SetAnteHandler(anteHandler sdk.AnteHandler) {
	l.cfg.AnteHandler = anteHandler
}

// Logger returns the logger for the lane.
func (l *LaneConstructor) Logger() log.Logger {
	return l.cfg.Logger
}

// TxDecoder returns the tx decoder for the lane.
func (l *LaneConstructor) TxDecoder() sdk.TxDecoder {
	return l.cfg.TxDecoder
}

// TxEncoder returns the tx encoder for the lane.
func (l *LaneConstructor) TxEncoder() sdk.TxEncoder {
	return l.cfg.TxEncoder
}

// GetMaxBlockSpace returns the maximum amount of block space that the lane is
// allowed to consume as a percentage of the total block space.
func (l *LaneConstructor) GetMaxBlockSpace() math.LegacyDec {
	return l.cfg.MaxBlockSpace
}
