package base

import (
	"github.com/cometbft/cometbft/libs/log"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/skip-mev/pob/blockbuster"
)

const (
	// LaneName defines the name of the default lane.
	LaneName = "default"
)

var _ blockbuster.Lane = (*DefaultLane)(nil)

// DefaultLane defines a default lane implementation. It contains a priority-nonce
// index along with core lane functionality.
type DefaultLane struct {
	// Mempool defines the mempool for the lane.
	Mempool

	// LaneConfig defines the base lane configuration.
	Cfg blockbuster.BaseLaneConfig
}

// NewDefaultLane returns a new default lane.
func NewDefaultLane(cfg blockbuster.BaseLaneConfig) *DefaultLane {
	if err := cfg.ValidateBasic(); err != nil {
		panic(err)
	}

	return &DefaultLane{
		Mempool: NewDefaultMempool(cfg.TxEncoder),
		Cfg:     cfg,
	}
}

// Match returns true if the transaction belongs to this lane. Since
// this is the default lane, it always returns true except for transactions
// that belong to lanes in the ignore list.
func (l *DefaultLane) Match(tx sdk.Tx) bool {
	for _, lane := range l.Cfg.IgnoreList {
		if lane.Match(tx) {
			return false
		}
	}

	return true
}

// Name returns the name of the lane.
func (l *DefaultLane) Name() string {
	return LaneName
}

// Logger returns the lane's logger.
func (l *DefaultLane) Logger() log.Logger {
	return l.Cfg.Logger
}

// SetAnteHandler sets the lane's antehandler.
func (l *DefaultLane) SetAnteHandler(anteHandler sdk.AnteHandler) {
	l.Cfg.AnteHandler = anteHandler
}

// GetMaxBlockSpace returns the maximum block space for the lane as a relative percentage.
func (l *DefaultLane) GetMaxBlockSpace() sdk.Dec {
	return l.Cfg.MaxBlockSpace
}

// GetIgnoreList returns the lane's ignore list.
func (l *DefaultLane) GetIgnoreList() []blockbuster.Lane {
	return l.Cfg.IgnoreList
}
