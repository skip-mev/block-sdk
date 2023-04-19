package mempool

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// IsAuctionTx returns true if the transaction is a transaction that is attempting to
// bid to the auction.
func (am *AuctionMempool) IsAuctionTx(tx sdk.Tx) (bool, error) {
	return am.config.IsAuctionTx(tx)
}

// GetTransactionSigners returns the signers of the bundle transaction.
func (am *AuctionMempool) GetTransactionSigners(tx []byte) (map[string]struct{}, error) {
	return am.config.GetTransactionSigners(tx)
}

// WrapBundleTransaction wraps a bundle transaction into sdk.Tx transaction.
func (am *AuctionMempool) WrapBundleTransaction(tx []byte) (sdk.Tx, error) {
	return am.config.WrapBundleTransaction(tx)
}

// GetAuctionBidInfo returns the bid info from an auction transaction.
func (am *AuctionMempool) GetAuctionBidInfo(tx sdk.Tx) (AuctionBidInfo, error) {
	bidder, err := am.GetBidder(tx)
	if err != nil {
		return AuctionBidInfo{}, err
	}

	bid, err := am.GetBid(tx)
	if err != nil {
		return AuctionBidInfo{}, err
	}

	transactions, err := am.GetBundledTransactions(tx)
	if err != nil {
		return AuctionBidInfo{}, err
	}

	return AuctionBidInfo{
		Bidder:       bidder,
		Bid:          bid,
		Transactions: transactions,
	}, nil
}

// GetBidder returns the bidder from an auction transaction.
func (am *AuctionMempool) GetBidder(tx sdk.Tx) (sdk.AccAddress, error) {
	if isAuctionTx, err := am.IsAuctionTx(tx); err != nil || !isAuctionTx {
		return nil, fmt.Errorf("transaction is not an auction transaction")
	}

	return am.config.GetBidder(tx)
}

// GetBid returns the bid from an auction transaction.
func (am *AuctionMempool) GetBid(tx sdk.Tx) (sdk.Coin, error) {
	if isAuctionTx, err := am.IsAuctionTx(tx); err != nil || !isAuctionTx {
		return sdk.Coin{}, fmt.Errorf("transaction is not an auction transaction")
	}

	return am.config.GetBid(tx)
}

// GetBundledTransactions returns the transactions that are bundled in an auction transaction.
func (am *AuctionMempool) GetBundledTransactions(tx sdk.Tx) ([][]byte, error) {
	if isAuctionTx, err := am.IsAuctionTx(tx); err != nil || !isAuctionTx {
		return nil, fmt.Errorf("transaction is not an auction transaction")
	}

	return am.config.GetBundledTransactions(tx)
}

// GetBundleSigners returns all of the signers for each transaction in the bundle.
func (am *AuctionMempool) GetBundleSigners(txs [][]byte) ([]map[string]struct{}, error) {
	signers := make([]map[string]struct{}, len(txs))

	for index, tx := range txs {
		txSigners, err := am.GetTransactionSigners(tx)
		if err != nil {
			return nil, err
		}

		signers[index] = txSigners
	}

	return signers, nil
}
