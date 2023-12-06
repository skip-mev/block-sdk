package base

import (
	"github.com/skip-mev/block-sdk/block/base"
)

const (
	// LaneName defines the name of the default lane.
	LaneName = "default"
)

// NewDefaultLane returns a new default lane. The DefaultLane defines a default
// lane implementation. The default lane orders transactions by the transaction fees.
// The default lane accepts any transaction. The default lane builds and verifies blocks
// in a similar fashion to how the CometBFT/Tendermint consensus engine builds and verifies
// blocks pre SDK version 0.47.0.
func NewDefaultLane(cfg base.LaneConfig, matchHandler base.MatchHandler) *base.BaseLane {
	options := []base.LaneOption{
		base.SetMatchHandler(matchHandler),
	}

	lane, err := base.NewBaseLane(
		cfg,
		LaneName,
		options...,
	)
	if err != nil {
		panic(err)
	}

	return lane
}
