package auction

import (
	"github.com/cometbft/cometbft/libs/log"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/skip-mev/pob/blockbuster"
)

const (
	// LaneName defines the name of the top-of-block auction lane.
	LaneName = "tob"
)

var _ blockbuster.Lane = (*TOBLane)(nil)

// TOBLane defines a top-of-block auction lane. The top of block auction lane
// hosts transactions that want to bid for inclusion at the top of the next block.
// The top of block auction lane stores bid transactions that are sorted by
// their bid price. The highest valid bid transaction is selected for inclusion in the
// next block. The bundled transactions of the selected bid transaction are also
// included in the next block.
type TOBLane struct {
	// Mempool defines the mempool for the lane.
	Mempool

	// LaneConfig defines the base lane configuration.
	cfg blockbuster.BaseLaneConfig

	// Factory defines the API/functionality which is responsible for determining
	// if a transaction is a bid transaction and how to extract relevant
	// information from the transaction (bid, timeout, bidder, etc.).
	Factory
}

// NewTOBLane returns a new TOB lane.
func NewTOBLane(
	logger log.Logger,
	txDecoder sdk.TxDecoder,
	txEncoder sdk.TxEncoder,
	maxTx int,
	anteHandler sdk.AnteHandler,
	af Factory,
	maxBlockSpace sdk.Dec,
) *TOBLane {
	return &TOBLane{
		Mempool: NewMempool(txEncoder, maxTx, af),
		cfg:     blockbuster.NewBaseLaneConfig(logger, txEncoder, txDecoder, anteHandler, maxBlockSpace),
		Factory: af,
	}
}

// Match returns true if the transaction is a bid transaction. This is determined
// by the AuctionFactory.
func (l *TOBLane) Match(tx sdk.Tx) bool {
	bidInfo, err := l.GetAuctionBidInfo(tx)
	return bidInfo != nil && err == nil
}

// Name returns the name of the lane.
func (l *TOBLane) Name() string {
	return LaneName
}
