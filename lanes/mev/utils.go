package mev

import (
	"errors"

	sdk "github.com/cosmos/cosmos-sdk/types"
<<<<<<< HEAD
	buildertypes "github.com/skip-mev/block-sdk/x/builder/types"
=======

	auctiontypes "github.com/skip-mev/block-sdk/x/auction/types"
>>>>>>> 3c6f319 (feat(docs): rename x/builder -> x/auction (#55))
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
		// a transaction with at at least one MsgAuctionBid message
		return nil, errors.New("invalid MsgAuctionBid transaction")
	}
}
