package mempool

import (
	"errors"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth/signing"
	auctiontypes "github.com/skip-mev/pob/x/auction/types"
)

// WrappedBidTx defines a wrapper around an sdk.Tx that contains a single
// MsgAuctionBid message with additional metadata.
type WrappedBidTx struct {
	signing.Tx

	bid sdk.Coins
}

func NewWrappedBidTx(tx sdk.Tx, bid sdk.Coins) *WrappedBidTx {
	return &WrappedBidTx{
		Tx:  tx.(signing.Tx),
		bid: bid,
	}
}

func (wbtx *WrappedBidTx) GetBid() sdk.Coins { return wbtx.bid }

// GetMsgAuctionBidFromTx attempts to retrieve a MsgAuctionBid from an sdk.Tx if
// one exists. If a MsgAuctionBid does exist and other messages are also present,
// an error is returned. If no MsgAuctionBid is present, <nil, nil> is returned.
func GetMsgAuctionBidFromTx(tx sdk.Tx) (*auctiontypes.MsgAuctionBid, error) {
	auctionBidMsgs := make([]*auctiontypes.MsgAuctionBid, 0)
	for _, msg := range tx.GetMsgs() {
		t, ok := msg.(*auctiontypes.MsgAuctionBid)
		if ok {
			auctionBidMsgs = append(auctionBidMsgs, t)
		}
	}

	switch {
	case len(auctionBidMsgs) == 0:
		// a normal transaction without a MsgAuctionBid message
		return nil, nil

	case len(auctionBidMsgs) == 1 && len(tx.GetMsgs()) == 1:
		// a single MsgAuctionBid message transaction
		return auctionBidMsgs[0], nil

	default:
		// a transaction with at at least one MsgAuctionBid message
		return nil, errors.New("invalid MsgAuctionBid transaction")
	}
}

// UnwrapBidTx attempts to unwrap a WrappedBidTx from an sdk.Tx if one exists.
func UnwrapBidTx(tx sdk.Tx) sdk.Tx {
	if tx == nil {
		return nil
	}

	wTx, ok := tx.(*WrappedBidTx)
	if ok {
		return wTx.Tx
	}

	return tx
}
