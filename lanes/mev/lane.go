package mev

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/skip-mev/block-sdk/block"
	"github.com/skip-mev/block-sdk/block/base"
)

const (
	// LaneName defines the name of the mev lane.
	LaneName = "mev"
)

var _ MEVLaneI = (*MEVLane)(nil)

// MEVLane defines a MEV (Maximal Extracted Value) auction lane. The MEV auction lane
// hosts transactions that want to bid for inclusion at the top of the next block.
// The MEV auction lane stores bid transactions that are sorted by their bid price.
// The highest valid bid transaction is selected for inclusion in the next block.
// The bundled transactions of the selected bid transaction are also included in the
// next block.
type (
	// MEVLaneI defines the interface for the mev auction lane. This interface
	// is utilized by both the x/auction module and the checkTx handler.
	MEVLaneI interface { //nolint
		block.Lane
		Factory
		GetTopAuctionTx(ctx context.Context) sdk.Tx
	}

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
) *MEVLane {
	lane := &MEVLane{
		BaseLane: base.NewBaseLane(
			cfg,
			LaneName,
			base.NewMempool[string](
				TxPriority(factory),
				cfg.TxEncoder,
				cfg.MaxTxs,
			),
			factory.MatchHandler(),
		),
		Factory: factory,
	}

	// Set the prepare lane handler to the TOB one
	lane.SetPrepareLaneHandler(lane.PrepareLaneHandler())

	// Set the process lane handler to the TOB one
	lane.SetProcessLaneHandler(lane.ProcessLaneHandler())

	// Set the check order handler to the TOB one
	lane.SetCheckOrderHandler(lane.CheckOrderHandler())

	if err := lane.ValidateBasic(); err != nil {
		panic(err)
	}

	return lane
}
