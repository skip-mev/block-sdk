package mempool

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkmempool "github.com/cosmos/cosmos-sdk/types/mempool"
)

var _ sdkmempool.Mempool = (*AuctionMempool)(nil)

// AuctionMempool defines an auction mempool. It can be seen as an extension of
// an SDK PriorityNonceMempool, i.e. a mempool that supports <sender, nonce>
// two-dimensional priority ordering, with the additional support of prioritizing
// and indexing auction bids.
type AuctionMempool struct {
	// globalIndex defines the index of all transactions in the mempool. It uses
	// the SDK's builtin PriorityNonceMempool. Once a bid is selected for top-of-block,
	// all subsequent transactions in the mempool will be selected from this index.
	globalIndex sdkmempool.Mempool

	// auctionIndex defines an index of auction bids.
	auctionIndex sdkmempool.Mempool

	// txDecoder defines the sdk.Tx decoder that allows us to decode transactions
	// and construct sdk.Txs from the bundled transactions.
	txDecoder sdk.TxDecoder

	// txEncoder defines the sdk.Tx encoder that allows us to encode transactions
	// to bytes.
	txEncoder sdk.TxEncoder

	// txIndex is a map of all transactions in the mempool. It is used
	// to quickly check if a transaction is already in the mempool.
	txIndex map[string]struct{}

	// config defines the transaction configuration for processing auction transactions.
	config Config
}

// AuctionTxPriority returns a TxPriority over auction bid transactions only. It
// is to be used in the auction index only.
func AuctionTxPriority() TxPriority[string] {
	return TxPriority[string]{
		GetTxPriority: func(goCtx context.Context, tx sdk.Tx) string {
			msgAuctionBid, err := GetMsgAuctionBidFromTx(tx)
			if err != nil {
				panic(err)
			}

			return msgAuctionBid.Bid.String()
		},
		Compare: func(a, b string) int {
			aCoins, _ := sdk.ParseCoinsNormalized(a)
			bCoins, _ := sdk.ParseCoinsNormalized(b)

			switch {
			case aCoins == nil && bCoins == nil:
				return 0

			case aCoins == nil:
				return -1

			case bCoins == nil:
				return 1

			default:
				switch {
				case aCoins.IsAllGT(bCoins):
					return 1

				case aCoins.IsAllLT(bCoins):
					return -1

				default:
					return 0
				}
			}
		},
		MinValue: "",
	}
}

func NewAuctionMempool(txDecoder sdk.TxDecoder, txEncoder sdk.TxEncoder, maxTx int, config Config) *AuctionMempool {
	return &AuctionMempool{
		globalIndex: NewPriorityMempool(
			PriorityNonceMempoolConfig[int64]{
				TxPriority: NewDefaultTxPriority(),
				MaxTx:      maxTx,
			},
		),
		auctionIndex: NewPriorityMempool(
			PriorityNonceMempoolConfig[string]{
				TxPriority: AuctionTxPriority(),
				MaxTx:      maxTx,
			},
		),
		txDecoder: txDecoder,
		txEncoder: txEncoder,
		txIndex:   make(map[string]struct{}),
		config:    config,
	}
}

// Insert inserts a transaction into the mempool. If the transaction is a special
// auction tx (tx that contains a single MsgAuctionBid), it will also insert the
// transaction into the auction index.
func (am *AuctionMempool) Insert(ctx context.Context, tx sdk.Tx) error {
	isAuctionTx, err := am.IsAuctionTx(tx)
	if err != nil {
		return err
	}

	// Insert the transactions into the appropriate index.
	switch {
	case !isAuctionTx:
		if err := am.globalIndex.Insert(ctx, tx); err != nil {
			return fmt.Errorf("failed to insert tx into global index: %w", err)
		}
	case isAuctionTx:
		if err := am.auctionIndex.Insert(ctx, tx); err != nil {
			return fmt.Errorf("failed to insert tx into auction index: %w", err)
		}
	}

	txHashStr, err := am.getTxHashStr(tx)
	if err != nil {
		return err
	}

	am.txIndex[txHashStr] = struct{}{}

	return nil
}

// Remove removes a transaction from the mempool. If the transaction is a special
// auction tx (tx that contains a single MsgAuctionBid), it will also remove all
// referenced transactions from the global mempool.
func (am *AuctionMempool) Remove(tx sdk.Tx) error {
	isAuctionTx, err := am.IsAuctionTx(tx)
	if err != nil {
		return err
	}

	// Remove the transactions from the appropriate index.
	switch {
	case !isAuctionTx:
		am.removeTx(am.globalIndex, tx)
	case isAuctionTx:
		am.removeTx(am.auctionIndex, tx)

		// Remove all referenced transactions from the global mempool.
		bundleTxs, err := am.GetBundledTransactions(tx)
		if err != nil {
			return err
		}

		for _, refTx := range bundleTxs {
			wrappedRefTx, err := am.WrapBundleTransaction(refTx)
			if err != nil {
				return err
			}

			am.removeTx(am.globalIndex, wrappedRefTx)
		}
	}

	return nil
}

// RemoveWithoutRefTx removes a transaction from the mempool without removing
// any referenced transactions. Referenced transactions only exist in special
// auction transactions (txs that only include a single MsgAuctionBid). This
// API is used to ensure that searchers are unable to remove valid transactions
// from the global mempool.
func (am *AuctionMempool) RemoveWithoutRefTx(tx sdk.Tx) error {
	isAuctionTx, err := am.IsAuctionTx(tx)
	if err != nil {
		return err
	}

	if isAuctionTx {
		am.removeTx(am.auctionIndex, tx)
	}

	return nil
}

// GetTopAuctionTx returns the highest bidding transaction in the auction mempool.
func (am *AuctionMempool) GetTopAuctionTx(ctx context.Context) sdk.Tx {
	iterator := am.auctionIndex.Select(ctx, nil)
	if iterator == nil {
		return nil
	}

	return iterator.Tx()
}

// AuctionBidSelect returns an iterator over auction bids transactions only.
func (am *AuctionMempool) AuctionBidSelect(ctx context.Context) sdkmempool.Iterator {
	return am.auctionIndex.Select(ctx, nil)
}

func (am *AuctionMempool) Select(ctx context.Context, txs [][]byte) sdkmempool.Iterator {
	return am.globalIndex.Select(ctx, txs)
}

func (am *AuctionMempool) CountAuctionTx() int {
	return am.auctionIndex.CountTx()
}

func (am *AuctionMempool) CountTx() int {
	return am.globalIndex.CountTx()
}

// Contains returns true if the transaction is contained in the mempool.
func (am *AuctionMempool) Contains(tx sdk.Tx) (bool, error) {
	txHashStr, err := am.getTxHashStr(tx)
	if err != nil {
		return false, fmt.Errorf("failed to get tx hash string: %w", err)
	}

	_, ok := am.txIndex[txHashStr]
	return ok, nil
}

func (am *AuctionMempool) removeTx(mp sdkmempool.Mempool, tx sdk.Tx) {
	err := mp.Remove(tx)
	if err != nil && !errors.Is(err, sdkmempool.ErrTxNotFound) {
		panic(fmt.Errorf("failed to remove invalid transaction from the mempool: %w", err))
	}

	txHashStr, err := am.getTxHashStr(tx)
	if err != nil {
		panic(fmt.Errorf("failed to get tx hash string: %w", err))
	}

	delete(am.txIndex, txHashStr)
}

// getTxHashStr returns the transaction hash string for a given transaction.
func (am *AuctionMempool) getTxHashStr(tx sdk.Tx) (string, error) {
	txBz, err := am.txEncoder(tx)
	if err != nil {
		return "", fmt.Errorf("failed to encode transaction: %w", err)
	}

	txHash := sha256.Sum256(txBz)
	txHashStr := hex.EncodeToString(txHash[:])

	return txHashStr, nil
}
