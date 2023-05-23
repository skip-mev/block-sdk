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
	// Mempool is an interface that defines the methods required to interact with the application-side mempool.
	Mempool interface {
		Contains(tx sdk.Tx) (bool, error)
		GetAuctionBidInfo(tx sdk.Tx) (*mempool.AuctionBidInfo, error)
		GetTopAuctionTx(ctx context.Context) sdk.Tx
	}

	// BuilderDecorator is an AnteDecorator that validates the auction bid and bundled transactions.
	BuilderDecorator struct {
		builderKeeper keeper.Keeper
		txEncoder     sdk.TxEncoder
		mempool       Mempool
	}
)

func NewBuilderDecorator(ak keeper.Keeper, txEncoder sdk.TxEncoder, mempool Mempool) BuilderDecorator {
	return BuilderDecorator{
		builderKeeper: ak,
		txEncoder:     txEncoder,
		mempool:       mempool,
	}
}

// AnteHandle validates that the auction bid is valid if one exists. If valid it will deduct the entrance fee from the
// bidder's account.
func (bd BuilderDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (sdk.Context, error) {
	// If comet is re-checking a transaction, we only need to check if the transaction is in the application-side mempool.
	if ctx.IsReCheckTx() {
		contains, err := bd.mempool.Contains(tx)
		if err != nil {
			return ctx, err
		}

		if !contains {
			return ctx, fmt.Errorf("transaction not found in application-side mempool")
		}
	}

	bidInfo, err := bd.mempool.GetAuctionBidInfo(tx)
	if err != nil {
		return ctx, err
	}

	// Validate the auction bid if one exists.
	if bidInfo != nil {
		// Auction transactions must have a timeout set to a valid block height.
		if err := bd.ValidateTimeout(ctx, int64(bidInfo.Timeout)); err != nil {
			return ctx, err
		}

		// We only need to verify the auction bid relative to the local validator's mempool if the mode
		// is checkTx or recheckTx. Otherwise, the ABCI handlers (VerifyVoteExtension, ExtendVoteExtension, etc.)
		// will always compare the auction bid to the highest bidding transaction in the mempool leading to
		// poor liveness guarantees.
		topBid := sdk.Coin{}
		if ctx.IsCheckTx() || ctx.IsReCheckTx() {
			if topBidTx := bd.mempool.GetTopAuctionTx(ctx); topBidTx != nil {
				topBidBz, err := bd.txEncoder(topBidTx)
				if err != nil {
					return ctx, err
				}

				currentTxBz, err := bd.txEncoder(tx)
				if err != nil {
					return ctx, err
				}

				// Compare the bytes to see if the current transaction is the highest bidding transaction.
				if !bytes.Equal(topBidBz, currentTxBz) {
					topBidInfo, err := bd.mempool.GetAuctionBidInfo(topBidTx)
					if err != nil {
						return ctx, err
					}

					topBid = topBidInfo.Bid
				}
			}
		}

		if err := bd.builderKeeper.ValidateBidInfo(ctx, topBid, bidInfo); err != nil {
			return ctx, errors.Wrap(err, "failed to validate auction bid")
		}
	}

	return next(ctx, tx, simulate)
}

// ValidateTimeout validates that the timeout is greater than or equal to the expected block height
// the bid transaction will be executed in.
//
// TODO: This will be deprecated in favor of the pre-commit hook once this available on the SDK
// https://github.com/skip-mev/pob/issues/147
func (bd BuilderDecorator) ValidateTimeout(ctx sdk.Context, timeout int64) error {
	currentBlockHeight := ctx.BlockHeight()

	// If the mode is CheckTx or ReCheckTx, we increment the current block height by one to
	// account for the fact that the transaction will be executed in the next block.
	if ctx.IsCheckTx() || ctx.IsReCheckTx() {
		currentBlockHeight++
	}

	if timeout < currentBlockHeight {
		return fmt.Errorf("timeout height cannot be less than the current block height")
	}

	return nil
}
