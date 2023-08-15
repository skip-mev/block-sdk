package base

import (
	"github.com/skip-mev/pob/block"
	"github.com/skip-mev/pob/block/constructor"
)

const (
	// LaneName defines the name of the default lane.
	LaneName = "default"
)

var _ block.Lane = (*DefaultLane)(nil)

// DefaultLane defines a default lane implementation. The default lane orders
// transactions by the transaction fees. The default lane accepts any transaction
// that is should not be ignored (as defined by the IgnoreList in the LaneConfig).
// The default lane builds and verifies blocks in a similiar fashion to how the
// CometBFT/Tendermint consensus engine builds and verifies blocks pre SDK version
// 0.47.0.
type DefaultLane struct {
	*constructor.LaneConstructor[string]
}

// NewDefaultLane returns a new default lane.
func NewDefaultLane(cfg block.LaneConfig) *DefaultLane {
	lane := constructor.NewLaneConstructor[string](
		cfg,
		LaneName,
		constructor.NewConstructorMempool[string](
			constructor.DefaultTxPriority(),
			cfg.TxEncoder,
			cfg.MaxTxs,
		),
		constructor.DefaultMatchHandler(),
	)

	return &DefaultLane{
		LaneConstructor: lane,
	}
}
