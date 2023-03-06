package mempool_test

import (
	"math/rand"
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/skip-mev/pob/mempool"
	"github.com/stretchr/testify/require"
)

var emptyHash = [32]byte{}

func TestAuctionBidList(t *testing.T) {
	abl := mempool.NewAuctionBidList()

	require.Nil(t, abl.TopBid())

	// insert a bid which should be the head and tail
	bid1 := sdk.NewCoins(sdk.NewInt64Coin("foo", 100))
	abl.Insert(mempool.NewWrappedBidTx(nil, emptyHash, bid1))
	require.Equal(t, bid1, abl.TopBid().GetBid())

	// insert 500 random bids between [100, 1000)
	var currTopBid sdk.Coins
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	for i := 0; i < 500; i++ {
		randomBid := rng.Int63n(1000-100) + 100

		bid := sdk.NewCoins(sdk.NewInt64Coin("foo", randomBid))
		abl.Insert(mempool.NewWrappedBidTx(nil, emptyHash, bid))

		currTopBid = abl.TopBid().GetBid()
	}

	// insert a bid which should be the new tail, thus the highest bid
	bid2 := sdk.NewCoins(sdk.NewInt64Coin("foo", 1000))
	abl.Insert(mempool.NewWrappedBidTx(nil, emptyHash, bid2))
	require.Equal(t, bid2, abl.TopBid().GetBid())

	// remove the top bid and ensure the new top bid is the previous top bid
	abl.Remove(mempool.NewWrappedBidTx(nil, emptyHash, bid2))
	require.Equal(t, currTopBid, abl.TopBid().GetBid())
}
