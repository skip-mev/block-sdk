package mempool

import (
	"context"
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
	globalIndex *PriorityNonceMempool[int64]

	// auctionIndex defines an index of auction bids.
	auctionIndex *PriorityNonceMempool[string]

	// txDecoder defines the sdk.Tx decoder that allows us to decode transactions
	// and construct sdk.Txs from the bundled transactions.
	txDecoder sdk.TxDecoder
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

func NewAuctionMempool(txDecoder sdk.TxDecoder, maxTx int) *AuctionMempool {
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
	}
}

// Insert inserts a transaction into the mempool. If the transaction is a special
// auction tx (tx that contains a single MsgAuctionBid), it will also insert the
// transaction into the auction index.
func (am *AuctionMempool) Insert(ctx context.Context, tx sdk.Tx) error {
	if err := am.globalIndex.Insert(ctx, tx); err != nil {
		return fmt.Errorf("failed to insert tx into global index: %w", err)
	}

	msg, err := GetMsgAuctionBidFromTx(tx)
	if err != nil {
		return err
	}

	if msg != nil {
		if err := am.auctionIndex.Insert(ctx, tx); err != nil {
			removeTx(am.globalIndex, tx)
			return fmt.Errorf("failed to insert tx into auction index: %w", err)
		}
	}

	return nil
}

// Remove removes a transaction from the mempool. If the transaction is a special
// auction tx (tx that contains a single MsgAuctionBid), it will also remove all
// referenced transactions from the global mempool.
func (am *AuctionMempool) Remove(tx sdk.Tx) error {
	// 1. Remove the tx from the global index
	removeTx(am.globalIndex, tx)

	msg, err := GetMsgAuctionBidFromTx(tx)
	if err != nil {
		return err
	}

	// 2. Remove the bid from the auction index (if applicable). In addition, we
	// remove all referenced transactions from the global mempool.
	if msg != nil {
		removeTx(am.auctionIndex, tx)

		for _, refRawTx := range msg.GetTransactions() {
			refTx, err := am.txDecoder(refRawTx)
			if err != nil {
				return fmt.Errorf("failed to decode referenced tx: %w", err)
			}

			removeTx(am.globalIndex, refTx)
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
	removeTx(am.globalIndex, tx)

	msg, err := GetMsgAuctionBidFromTx(tx)
	if err != nil {
		return err
	}

	if msg != nil {
		removeTx(am.auctionIndex, tx)
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

func removeTx(mp sdkmempool.Mempool, tx sdk.Tx) {
	err := mp.Remove(tx)
	if err != nil && !errors.Is(err, sdkmempool.ErrTxNotFound) {
		panic(fmt.Errorf("failed to remove invalid transaction from the mempool: %w", err))
	}
}
