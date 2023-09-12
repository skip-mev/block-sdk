package ante

import (
	"bytes"
	"context"
	"fmt"

	"cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
<<<<<<< HEAD:x/builder/ante/ante.go
	"github.com/skip-mev/block-sdk/x/builder/keeper"
	"github.com/skip-mev/block-sdk/x/builder/types"
=======

	"github.com/skip-mev/block-sdk/x/auction/keeper"
	"github.com/skip-mev/block-sdk/x/auction/types"
>>>>>>> 3c6f319 (feat(docs): rename x/builder -> x/auction (#55)):x/auction/ante/ante.go
)

var _ sdk.AnteDecorator = AuctionDecorator{}

type (
	// MEVLane is an interface that defines the methods required to interact with the MEV
	// lane.
	MEVLane interface {
		GetAuctionBidInfo(tx sdk.Tx) (*types.BidInfo, error)
		GetTopAuctionTx(ctx context.Context) sdk.Tx
	}

	// Mempool is an interface that defines the methods required to interact with the application-side mempool.
	Mempool interface {
		Contains(tx sdk.Tx) bool
	}

	// AuctionDecorator is an AnteDecorator that validates the auction bid and bundled transactions.
	AuctionDecorator struct {
		auctionkeeper keeper.Keeper
		txEncoder     sdk.TxEncoder
		lane          MEVLane
		mempool       Mempool
	}
)

func NewAuctionDecorator(ak keeper.Keeper, txEncoder sdk.TxEncoder, lane MEVLane, mempool Mempool) AuctionDecorator {
	return AuctionDecorator{
		auctionkeeper: ak,
		txEncoder:     txEncoder,
		lane:          lane,
		mempool:       mempool,
	}
}

// AnteHandle validates that the auction bid is valid if one exists. If valid it will deduct the entrance fee from the
// bidder's account.
func (ad AuctionDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (sdk.Context, error) {
	// If comet is re-checking a transaction, we only need to check if the transaction is in the application-side mempool.
	if ctx.IsReCheckTx() {
		if !ad.mempool.Contains(tx) {
			return ctx, fmt.Errorf("transaction not found in application-side mempool")
		}
	}

	bidInfo, err := ad.lane.GetAuctionBidInfo(tx)
	if err != nil {
		return ctx, err
	}

	// Validate the auction bid if one exists.
	if bidInfo != nil {
		// Auction transactions must have a timeout set to a valid block height.
		if err := ad.ValidateTimeout(ctx, int64(bidInfo.Timeout)); err != nil {
			return ctx, err
		}

		// We only need to verify the auction bid relative to the local validator's mempool if the mode
		// is checkTx or recheckTx. Otherwise, the ABCI handlers (VerifyVoteExtension, ExtendVoteExtension, etc.)
		// will always compare the auction bid to the highest bidding transaction in the mempool leading to
		// poor liveness guarantees.
		topBid := sdk.Coin{}
		if ctx.IsCheckTx() || ctx.IsReCheckTx() {
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

// ValidateTimeout validates that the timeout is greater than or equal to the expected block height
// the bid transaction will be executed in.
func (ad AuctionDecorator) ValidateTimeout(ctx sdk.Context, timeout int64) error {
	currentBlockHeight := ctx.BlockHeight()

	// If the mode is CheckTx or ReCheckTx, we increment the current block height by one to
	// account for the fact that the transaction will be executed in the next block.
	if ctx.IsCheckTx() || ctx.IsReCheckTx() {
		currentBlockHeight++
	}

	if timeout < currentBlockHeight {
		return fmt.Errorf(
			"timeout height cannot be less than the current block height (timeout: %d, current block height: %d)",
			timeout,
			currentBlockHeight,
		)
	}

	return nil
}
