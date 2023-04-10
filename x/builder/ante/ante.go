package ante

import (
	"bytes"
	"fmt"

	"cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/skip-mev/pob/mempool"
	"github.com/skip-mev/pob/x/builder/keeper"
)

var _ sdk.AnteDecorator = BuilderDecorator{}

type (
	BuilderDecorator struct {
		builderKeeper keeper.Keeper
		txDecoder     sdk.TxDecoder
		txEncoder     sdk.TxEncoder
		mempool       *mempool.AuctionMempool
	}

	TxWithTimeoutHeight interface {
		sdk.Tx

		GetTimeoutHeight() uint64
	}
)

func NewBuilderDecorator(ak keeper.Keeper, txDecoder sdk.TxDecoder, txEncoder sdk.TxEncoder, mempool *mempool.AuctionMempool) BuilderDecorator {
	return BuilderDecorator{
		builderKeeper: ak,
		txDecoder:     txDecoder,
		txEncoder:     txEncoder,
		mempool:       mempool,
	}
}

// AnteHandle validates that the auction bid is valid if one exists. If valid it will deduct the entrance fee from the
// bidder's account.
func (ad BuilderDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (sdk.Context, error) {
	// If comet is re-checking a transaction, we only need to check if the transaction is in the application-side mempool.
	if ctx.IsReCheckTx() {
		contains, err := ad.mempool.Contains(tx)
		if err != nil {
			return ctx, err
		}

		if !contains {
			return ctx, fmt.Errorf("transaction not found in application mempool")
		}
	}

	auctionMsg, err := mempool.GetMsgAuctionBidFromTx(tx)
	if err != nil {
		return ctx, err
	}

	// Validate the auction bid if one exists.
	if auctionMsg != nil {
		auctionTx, ok := tx.(TxWithTimeoutHeight)
		if !ok {
			return ctx, fmt.Errorf("transaction does not implement TxWithTimeoutHeight")
		}

		timeout := auctionTx.GetTimeoutHeight()
		if timeout == 0 {
			return ctx, fmt.Errorf("timeout height cannot be zero")
		}

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

		topBid := sdk.Coin{}

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

		if err := ad.builderKeeper.ValidateAuctionMsg(ctx, bidder, auctionMsg.Bid, topBid, transactions); err != nil {
			return ctx, errors.Wrap(err, "failed to validate auction bid")
		}
	}

	return next(ctx, tx, simulate)
}

// GetTopAuctionBid returns the highest auction bid if one exists.
func (ad BuilderDecorator) GetTopAuctionBid(ctx sdk.Context) (sdk.Coin, error) {
	auctionTx := ad.mempool.GetTopAuctionTx(ctx)
	if auctionTx == nil {
		return sdk.Coin{}, nil
	}

	msgAuctionBid, err := mempool.GetMsgAuctionBidFromTx(auctionTx)
	if err != nil {
		return sdk.Coin{}, err
	}

	return msgAuctionBid.Bid, nil
}

// IsTopBidTx returns true if the transaction inputted is the highest bidding auction transaction in the mempool.
func (ad BuilderDecorator) IsTopBidTx(ctx sdk.Context, tx sdk.Tx) (bool, error) {
	auctionTx := ad.mempool.GetTopAuctionTx(ctx)
	if auctionTx == nil {
		return false, nil
	}

	topBidBz, err := ad.txEncoder(auctionTx)
	if err != nil {
		return false, err
	}

	currentTxBz, err := ad.txEncoder(tx)
	if err != nil {
		return false, err
	}

	return bytes.Equal(topBidBz, currentTxBz), nil
}
