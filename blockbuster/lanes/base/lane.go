package base

import (
	"github.com/skip-mev/pob/blockbuster"
)

const (
	// LaneName defines the name of the default lane.
	LaneName = "default"
)

var _ blockbuster.Lane = (*DefaultLane)(nil)

// DefaultLane defines a default lane implementation. The default lane orders
// transactions by the transaction fees. The default lane accepts any transaction
// that is should not be ignored (as defined by the IgnoreList in the LaneConfig).
// The default lane builds and verifies blocks in a similar fashion to how the
// CometBFT/Tendermint consensus engine builds and verifies blocks pre SDK version
// 0.47.0.
type DefaultLane struct {
	*blockbuster.LaneConstructor
}

// NewDefaultLane returns a new default lane.
func NewDefaultLane(cfg blockbuster.LaneConfig) *DefaultLane {
	lane := blockbuster.NewLaneConstructor(
		cfg,
		LaneName,
		blockbuster.NewConstructorMempool[string](
			blockbuster.DefaultTxPriority(),
			cfg.TxEncoder,
			cfg.MaxTxs,
		),
		blockbuster.DefaultMatchHandler(),
	)

	return &DefaultLane{
		LaneConstructor: lane,
	}
}
