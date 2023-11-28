package base

import (
	"github.com/skip-mev/block-sdk/block"
	"github.com/skip-mev/block-sdk/block/base"
)

const (
	// LaneName defines the name of the default lane.
	LaneName = "default"
)

var _ block.Lane = (*DefaultLane)(nil)

// DefaultLane defines a default lane implementation. The default lane orders
// transactions by the transaction fees. The default lane accepts any transaction
// that should not be ignored (as defined by the IgnoreList in the LaneConfig).
// The default lane builds and verifies blocks in a similar fashion to how the
// CometBFT/Tendermint consensus engine builds and verifies blocks pre SDK version
// 0.47.0.
type DefaultLane struct {
	*base.BaseLane
}

// NewDefaultLane returns a new default lane.
func NewDefaultLane(cfg base.LaneConfig, matchHandler base.MatchHandler) *DefaultLane {
	lane := base.NewBaseLane(
		cfg,
		LaneName,
		base.NewMempool[string](
			base.DefaultTxPriority(),
			cfg.TxEncoder,
			cfg.SignerExtractor,
			cfg.MaxTxs,
		),
		matchHandler,
	)

	if err := lane.ValidateBasic(); err != nil {
		panic(err)
	}

	return &DefaultLane{
		BaseLane: lane,
	}
}
