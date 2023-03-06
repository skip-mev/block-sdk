package mempool

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/huandu/skiplist"
)

type (
	// AuctionBidList defines a list of WrappedBidTx objects, sorted by their bids.

	AuctionBidList struct {
		list *skiplist.SkipList
	}

	auctionBidListKey struct {
		bid  sdk.Coins
		hash []byte
	}
)

func NewAuctionBidList() *AuctionBidList {
	return &AuctionBidList{
		list: skiplist.New(skiplist.GreaterThanFunc(func(lhs, rhs any) int {
			bidA := lhs.(auctionBidListKey)
			bidB := rhs.(auctionBidListKey)

			switch {
			case bidA.bid.IsAllGT(bidB.bid):
				return 1

			case bidA.bid.IsAllLT(bidB.bid):
				return -1

			default:
				// in case of a tie in bid, sort by hash
				return skiplist.ByteAsc.Compare(bidA.hash, bidB.hash)
			}
		})),
	}
}

// TopBid returns the WrappedBidTx with the highest bid.
func (abl *AuctionBidList) TopBid() *WrappedBidTx {
	n := abl.list.Back()
	if n == nil {
		return nil
	}

	return n.Value.(*WrappedBidTx)
}

func (abl *AuctionBidList) Insert(wBidTx *WrappedBidTx) {
	abl.list.Set(auctionBidListKey{bid: wBidTx.bid, hash: wBidTx.hash[:]}, wBidTx)
}

func (abl *AuctionBidList) Remove(wBidTx *WrappedBidTx) {
	abl.list.Remove(auctionBidListKey{bid: wBidTx.bid, hash: wBidTx.hash[:]})
}
