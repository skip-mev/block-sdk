package types

import sdk "github.com/cosmos/cosmos-sdk/types"

// BidInfo defines the information about a bid to the auction house.
type BidInfo struct {
	// Bidder is the address of the bidder.
	Bidder sdk.AccAddress
	// Bid is the amount of coins that the bidder is bidding.
	Bid sdk.Coin
	// Transactions is the bundle of transactions that the bidder is committing to.
	Transactions [][]byte
	// Timeout is the block height at which the bid transaction will be executed. This must be the next block height.
	Timeout uint64
	// Signers is the list of signers for each transaction in the bundle.
	Signers []map[string]struct{}
	// TransactionTimeouts is the list of timeouts for each transaction in the bundle.
	TransactionTimeouts []uint64
}
