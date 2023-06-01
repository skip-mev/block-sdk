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
	cfg blockbuster.BaseLaneConfig
}

// NewDefaultLane returns a new default lane.
func NewDefaultLane(logger log.Logger, txDecoder sdk.TxDecoder, txEncoder sdk.TxEncoder, anteHandler sdk.AnteHandler) *DefaultLane {
	return &DefaultLane{
		Mempool: NewDefaultMempool(txEncoder),
		cfg:     blockbuster.NewBaseLaneConfig(logger, txEncoder, txDecoder, anteHandler),
	}
}

// Match returns true if the transaction belongs to this lane. Since
// this is the default lane, it always returns true. This means that
// any transaction can be included in this lane.
func (l *DefaultLane) Match(sdk.Tx) bool {
	return true
}

// Name returns the name of the lane.
func (l *DefaultLane) Name() string {
	return LaneName
}
