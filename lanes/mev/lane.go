package mev

import (
	"github.com/skip-mev/block-sdk/v2/block/base"
)

const (
	// LaneName defines the name of the mev lane.
	LaneName = "mev"
)

// MEVLane defines a MEV (Maximal Extracted Value) auction lane. The MEV auction lane
// hosts transactions that want to bid for inclusion at the top of the next block.
// The MEV auction lane stores bid transactions that are sorted by their bid price.
// The highest valid bid transaction is selected for inclusion in the next block.
// The bundled transactions of the selected bid transaction are also included in the
// next block.
type (
	MEVLane struct { //nolint
		*base.BaseLane

		// Factory defines the API/functionality which is responsible for determining
		// if a transaction is a bid transaction and how to extract relevant
		// information from the transaction (bid, timeout, bidder, etc.).
		Factory
	}
)

// NewMEVLane returns a new TOB lane.
func NewMEVLane(
	cfg base.LaneConfig,
	factory Factory,
	matchHandler base.MatchHandler,
) *MEVLane {
	options := []base.LaneOption{
		base.WithMatchHandler(matchHandler),
		base.WithMempoolConfigs[string](cfg, TxPriority(factory)),
	}

	baseLane, err := base.NewBaseLane(
		cfg,
		LaneName,
		options...,
	)
	if err != nil {
		panic(err)
	}

	// Create the mev proposal handler.
	handler := NewProposalHandler(baseLane, factory)
	baseLane.WithOptions(
		base.WithPrepareLaneHandler(handler.PrepareLaneHandler()),
		base.WithProcessLaneHandler(handler.ProcessLaneHandler()),
	)

	return &MEVLane{
		BaseLane: baseLane,
		Factory:  factory,
	}
}
