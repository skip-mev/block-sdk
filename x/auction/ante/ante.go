package ante

import (
	"bytes"

	"cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/skip-mev/pob/mempool"
	"github.com/skip-mev/pob/x/auction/keeper"
)

var _ sdk.AnteDecorator = AuctionDecorator{}

type AuctionDecorator struct {
	auctionKeeper keeper.Keeper
	txDecoder     sdk.TxDecoder
	txEncoder     sdk.TxEncoder
	mempool       *mempool.AuctionMempool
}

func NewAuctionDecorator(ak keeper.Keeper, txDecoder sdk.TxDecoder, txEncoder sdk.TxEncoder, mempool *mempool.AuctionMempool) AuctionDecorator {
	return AuctionDecorator{
		auctionKeeper: ak,
		txDecoder:     txDecoder,
		txEncoder:     txEncoder,
		mempool:       mempool,
	}
}

// AnteHandle validates that the auction bid is valid if one exists. If valid it will deduct the entrance fee from the
// bidder's account.
func (ad AuctionDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (sdk.Context, error) {
	auctionMsg, err := mempool.GetMsgAuctionBidFromTx(tx)
	if err != nil {
		return ctx, err
	}

	// Validate the auction bid if one exists.
	if auctionMsg != nil {
		bidder, err := sdk.AccAddressFromBech32(auctionMsg.Bidder)
		if err != nil {
			return ctx, errors.Wrapf(err, "invalid bidder address (%s)", auctionMsg.Bidder)
		}

		transactions := make([]sdk.Tx, len(auctionMsg.Transactions))
		for i, tx := range auctionMsg.Transactions {
			decodedTx, err := ad.txDecoder(tx)
			if err != nil {
				return ctx, errors.Wrapf(err, "failed to decode transaction (%s)", tx)
			}

			transactions[i] = decodedTx
		}

		topBid := sdk.NewCoins()

		// If the current transaction is the highest bidding transaction, then the highest bid is empty.
		isTopBidTx, err := ad.IsTopBidTx(ctx, tx)
		if err != nil {
			return ctx, errors.Wrap(err, "failed to check if current transaction is highest bidding transaction")
		}

		if !isTopBidTx {
			// Set the top bid to the highest bidding transaction.
			topBid, err = ad.GetTopAuctionBid(ctx)
			if err != nil {
				return ctx, errors.Wrap(err, "failed to get highest auction bid")
			}
		}

		if err := ad.auctionKeeper.ValidateAuctionMsg(ctx, bidder, auctionMsg.Bid, topBid, transactions); err != nil {
			return ctx, errors.Wrap(err, "failed to validate auction bid")
		}
	}

	return next(ctx, tx, simulate)
}

// GetTopAuctionBid returns the highest auction bid if one exists.
func (ad AuctionDecorator) GetTopAuctionBid(ctx sdk.Context) (sdk.Coins, error) {
	auctionTx := ad.mempool.GetTopAuctionTx(ctx)
	if auctionTx == nil {
		return sdk.NewCoins(), nil
	}

	return auctionTx.(*mempool.WrappedBidTx).GetBid(), nil
}

// IsTopBidTx returns true if the transaction inputted is the highest bidding auction transaction in the mempool.
func (ad AuctionDecorator) IsTopBidTx(ctx sdk.Context, tx sdk.Tx) (bool, error) {
	auctionTx := ad.mempool.GetTopAuctionTx(ctx)
	if auctionTx == nil {
		return false, nil
	}

	topBidTx := mempool.UnwrapBidTx(auctionTx)
	topBidBz, err := ad.txEncoder(topBidTx)
	if err != nil {
		return false, err
	}

	currentTxBz, err := ad.txEncoder(tx)
	if err != nil {
		return false, err
	}

	return bytes.Equal(topBidBz, currentTxBz), nil
}
