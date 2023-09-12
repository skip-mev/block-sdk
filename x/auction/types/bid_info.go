package types

import sdk "github.com/cosmos/cosmos-sdk/types"

// BidInfo defines the information about a bid to the auction house.
type BidInfo struct {
	Bidder       sdk.AccAddress
	Bid          sdk.Coin
	Transactions [][]byte
	Timeout      uint64
	Signers      []map[string]struct{}
}
