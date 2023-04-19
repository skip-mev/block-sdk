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

	isAuctionTx, err := ad.mempool.IsAuctionTx(tx)
	if err != nil {
		return ctx, err
	}

	// Validate the auction bid if one exists.
	if isAuctionTx {
		// Auction transactions must have a timeout set to a valid block height.
		if err := ad.HasValidTimeout(ctx, tx); err != nil {
			return ctx, err
		}

		bidInfo, err := ad.mempool.GetAuctionBidInfo(tx)
		if err != nil {
			return ctx, err
		}

		// If the current transaction is the highest bidding transaction, then the highest bid is empty.
		topBid := sdk.Coin{}
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

		// Extract signers from bundle for verification.
		signers, err := ad.mempool.GetBundleSigners(bidInfo.Transactions)
		if err != nil {
			return ctx, errors.Wrap(err, "failed to get bundle signers")
		}

		if err := ad.builderKeeper.ValidateBidInfo(ctx, topBid, bidInfo, signers); err != nil {
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

	bid, err := ad.mempool.GetBid(auctionTx)
	if err != nil {
		return sdk.Coin{}, err
	}

	return bid, nil
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

// HasValidTimeout returns true if the transaction has a valid timeout height.
func (ad BuilderDecorator) HasValidTimeout(ctx sdk.Context, tx sdk.Tx) error {
	auctionTx, ok := tx.(TxWithTimeoutHeight)
	if !ok {
		return fmt.Errorf("transaction does not implement TxWithTimeoutHeight")
	}

	timeout := auctionTx.GetTimeoutHeight()
	if timeout == 0 {
		return fmt.Errorf("timeout height cannot be zero")
	}

	if timeout < uint64(ctx.BlockHeight()) {
		return fmt.Errorf("timeout height cannot be less than the current block height")
	}

	return nil
}
