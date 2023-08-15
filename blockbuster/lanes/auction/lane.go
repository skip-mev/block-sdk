package auction

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/skip-mev/pob/blockbuster"
)

const (
	// LaneName defines the name of the top-of-block auction lane.
	LaneName = "top-of-block"
)

var _ TOBLaneI = (*TOBLane)(nil)

// TOBLane defines a top-of-block auction lane. The top of block auction lane
// hosts transactions that want to bid for inclusion at the top of the next block.
// The top of block auction lane stores bid transactions that are sorted by
// their bid price. The highest valid bid transaction is selected for inclusion in the
// next block. The bundled transactions of the selected bid transaction are also
// included in the next block.
type (
	// TOBLaneI defines the interface for the top-of-block auction lane. This interface
	// is utilized by both the x/builder module and the checkTx handler.
	TOBLaneI interface {
		blockbuster.Lane
		Factory
		GetTopAuctionTx(ctx context.Context) sdk.Tx
	}

	TOBLane struct {
		// LaneConfig defines the base lane configuration.
		*blockbuster.LaneConstructor

		// Factory defines the API/functionality which is responsible for determining
		// if a transaction is a bid transaction and how to extract relevant
		// information from the transaction (bid, timeout, bidder, etc.).
		Factory
	}
)

// NewTOBLane returns a new TOB lane.
func NewTOBLane(
	cfg blockbuster.LaneConfig,
	factory Factory,
) *TOBLane {
	lane := &TOBLane{
		LaneConstructor: blockbuster.NewLaneConstructor(
			cfg,
			LaneName,
			blockbuster.NewConstructorMempool[string](
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

	return lane
}
