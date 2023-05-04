package ante

import (
	"bytes"
	"context"
	"fmt"

	"cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/skip-mev/pob/mempool"
	"github.com/skip-mev/pob/x/builder/keeper"
)

var _ sdk.AnteDecorator = BuilderDecorator{}

type (
	Mempool interface {
		Contains(tx sdk.Tx) (bool, error)
		GetAuctionBidInfo(tx sdk.Tx) (*mempool.AuctionBidInfo, error)
		GetTopAuctionTx(ctx context.Context) sdk.Tx
	}

	BuilderDecorator struct {
		builderKeeper keeper.Keeper
		txDecoder     sdk.TxDecoder
		txEncoder     sdk.TxEncoder
		mempool       Mempool
	}
)

func NewBuilderDecorator(ak keeper.Keeper, txDecoder sdk.TxDecoder, txEncoder sdk.TxEncoder, mempool Mempool) BuilderDecorator {
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

	bidInfo, err := ad.mempool.GetAuctionBidInfo(tx)
	if err != nil {
		return ctx, err
	}

	// Validate the auction bid if one exists.
	if bidInfo != nil {
		// Auction transactions must have a timeout set to a valid block height.
		if int64(bidInfo.Timeout) < ctx.BlockHeight() {
			return ctx, fmt.Errorf("timeout height cannot be less than the current block height")
		}

		// We only need to verify the auction bid relative to the local validator's mempool if the mode
		// is checkTx or recheckTx. Otherwise, the ABCI handlers (VerifyVoteExtension, ExtendVoteExtension, etc.)
		// will always compare the auction bid to the highest bidding transaction in the mempool leading to
		// poor liveness guarantees.
		topBid := sdk.Coin{}
		if ctx.IsCheckTx() || ctx.IsReCheckTx() {
			if topBidTx := ad.mempool.GetTopAuctionTx(ctx); topBidTx != nil {
				topBidBz, err := ad.txEncoder(topBidTx)
				if err != nil {
					return ctx, err
				}

				currentTxBz, err := ad.txEncoder(tx)
				if err != nil {
					return ctx, err
				}

				// Compare the bytes to see if the current transaction is the highest bidding transaction.
				if !bytes.Equal(topBidBz, currentTxBz) {
					topBidInfo, err := ad.mempool.GetAuctionBidInfo(topBidTx)
					if err != nil {
						return ctx, err
					}

					topBid = topBidInfo.Bid
				}
			}
		}

		if err := ad.builderKeeper.ValidateBidInfo(ctx, topBid, bidInfo); err != nil {
			return ctx, errors.Wrap(err, "failed to validate auction bid")
		}
	}

	return next(ctx, tx, simulate)
}
