package ante

import (
	"cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/skip-mev/pob/mempool"
	"github.com/skip-mev/pob/x/auction/keeper"
)

var _ sdk.AnteDecorator = AuctionDecorator{}

type AuctionDecorator struct {
	auctionKeeper keeper.Keeper
	txDecoder     sdk.TxDecoder
}

func NewAuctionDecorator(ak keeper.Keeper, txDecoder sdk.TxDecoder) AuctionDecorator {
	return AuctionDecorator{
		auctionKeeper: ak,
		txDecoder:     txDecoder,
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

		if err := ad.auctionKeeper.ValidateAuctionMsg(ctx, bidder, auctionMsg.Bid, transactions); err != nil {
			return ctx, errors.Wrap(err, "failed to validate auction bid")
		}
	}

	return next(ctx, tx, simulate)
}
