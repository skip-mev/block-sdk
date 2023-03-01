package mempool

import (
	"errors"

	sdk "github.com/cosmos/cosmos-sdk/types"
	auctiontypes "github.com/skip-mev/pob/x/auction/types"
)

type (
	// WrappedTx defines a wrapper around an sdk.Tx with additional metadata.
	WrappedTx struct {
		sdk.Tx

		hash [32]byte
	}

	// WrappedBidTx defines a wrapper around an sdk.Tx that contains a single
	// MsgAuctionBid message with additional metadata.
	WrappedBidTx struct {
		sdk.Tx

		hash [32]byte
		bid  sdk.Coins
	}
)

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
		// A transaction with at at least one MsgAuctionBid message and some other
		// message.
		return nil, errors.New("invalid MsgAuctionBid transaction")
	}
}
