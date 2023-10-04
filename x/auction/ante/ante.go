package ante

import (
	"bytes"

	"cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

var _ sdk.AnteDecorator = AuctionDecorator{}

type (
	// AuctionDecorator is an AnteDecorator that validates the auction bid and bundled transactions.
	AuctionDecorator struct {
		lane          MEVLane
		auctionkeeper AuctionKeeper
		txEncoder     sdk.TxEncoder
	}
)

// NewAuctionDecorator returns a new AuctionDecorator.
func NewAuctionDecorator(ak AuctionKeeper, txEncoder sdk.TxEncoder, lane MEVLane) AuctionDecorator {
	return AuctionDecorator{
		auctionkeeper: ak,
		txEncoder:     txEncoder,
		lane:          lane,
	}
}

// AnteHandle validates that the auction bid is valid if one exists. If valid it will deduct the entrance fee from the
// bidder's account.
func (ad AuctionDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (sdk.Context, error) {
	bidInfo, err := ad.lane.GetAuctionBidInfo(tx)
	if err != nil {
		return ctx, err
	}

	// Validate the auction bid if one exists.
	if bidInfo != nil {
		// Auction transactions must have a timeout set to a valid block height.
		if err := ValidateTimeout(ctx, int64(bidInfo.Timeout)); err != nil {
			return ctx, err
		}

		// Only compare the bid to the top bid if necessary.
		topBid := sdk.Coin{}
		if _, ok := nextHeightExecModes[ctx.ExecMode()]; ok {
			if topBidTx := ad.lane.GetTopAuctionTx(ctx); topBidTx != nil {
				topBidBz, err := ad.txEncoder(topBidTx)
				if err != nil {
					return ctx, err
				}

				currentTxBz, err := ad.txEncoder(tx)
				if err != nil {
					return ctx, err
				}

				// Compare the bytes to see if the current transaction is the highest bidding transaction.
				// If it is the same transaction, we do not need to compare the bids as the bid check will
				// fail.
				if !bytes.Equal(topBidBz, currentTxBz) {
					topBidInfo, err := ad.lane.GetAuctionBidInfo(topBidTx)
					if err != nil {
						return ctx, err
					}

					topBid = topBidInfo.Bid
				}
			}
		}

		if err := ad.auctionkeeper.ValidateBidInfo(ctx, topBid, bidInfo); err != nil {
			return ctx, errors.Wrap(err, "failed to validate auction bid")
		}
	}

	return next(ctx, tx, simulate)
}
