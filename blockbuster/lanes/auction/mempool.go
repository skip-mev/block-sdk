package auction

import (
	"context"
	"errors"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkmempool "github.com/cosmos/cosmos-sdk/types/mempool"
	"github.com/skip-mev/pob/blockbuster"
	"github.com/skip-mev/pob/blockbuster/utils"
)

var _ Mempool = (*TOBMempool)(nil)

type (
	// Mempool defines the interface of the auction mempool.
	Mempool interface {
		sdkmempool.Mempool

		// GetTopAuctionTx returns the highest bidding transaction in the auction mempool.
		GetTopAuctionTx(ctx context.Context) sdk.Tx

		// Contains returns true if the transaction is contained in the mempool.
		Contains(tx sdk.Tx) bool
	}

	// TOBMempool defines an auction mempool. It can be seen as an extension of
	// an SDK PriorityNonceMempool, i.e. a mempool that supports <sender, nonce>
	// two-dimensional priority ordering, with the additional support of prioritizing
	// and indexing auction bids.
	TOBMempool struct {
		// index defines an index of auction bids.
		index sdkmempool.Mempool

		// txEncoder defines the sdk.Tx encoder that allows us to encode transactions
		// to bytes.
		txEncoder sdk.TxEncoder

		// txIndex is a map of all transactions in the mempool. It is used
		// to quickly check if a transaction is already in the mempool.
		txIndex map[string]struct{}

		// Factory implements the functionality required to process auction transactions.
		Factory
	}
)

// TxPriority returns a TxPriority over auction bid transactions only. It
// is to be used in the auction index only.
func TxPriority(config Factory) blockbuster.TxPriority[string] {
	return blockbuster.TxPriority[string]{
		GetTxPriority: func(goCtx context.Context, tx sdk.Tx) string {
			bidInfo, err := config.GetAuctionBidInfo(tx)
			if err != nil {
				panic(err)
			}

			return bidInfo.Bid.String()
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

// NewMempool returns a new auction mempool.
func NewMempool(txEncoder sdk.TxEncoder, maxTx int, config Factory) *TOBMempool {
	return &TOBMempool{
		index: blockbuster.NewPriorityMempool(
			blockbuster.PriorityNonceMempoolConfig[string]{
				TxPriority: TxPriority(config),
				MaxTx:      maxTx,
			},
		),
		txEncoder: txEncoder,
		txIndex:   make(map[string]struct{}),
		Factory:   config,
	}
}

// Insert inserts a transaction into the auction mempool.
func (am *TOBMempool) Insert(ctx context.Context, tx sdk.Tx) error {
	if err := am.index.Insert(ctx, tx); err != nil {
		return fmt.Errorf("failed to insert tx into auction index: %w", err)
	}

	_, txHashStr, err := utils.GetTxHashStr(am.txEncoder, tx)
	if err != nil {
		return err
	}

	am.txIndex[txHashStr] = struct{}{}

	return nil
}

// Remove removes a transaction from the mempool based.
func (am *TOBMempool) Remove(tx sdk.Tx) error {
	if err := am.index.Remove(tx); err != nil && !errors.Is(err, sdkmempool.ErrTxNotFound) {
		return fmt.Errorf("failed to remove invalid transaction from the mempool: %w", err)
	}

	_, txHashStr, err := utils.GetTxHashStr(am.txEncoder, tx)
	if err != nil {
		return fmt.Errorf("failed to get tx hash string: %w", err)
	}

	delete(am.txIndex, txHashStr)

	return nil
}

// GetTopAuctionTx returns the highest bidding transaction in the auction mempool.
func (am *TOBMempool) GetTopAuctionTx(ctx context.Context) sdk.Tx {
	iterator := am.index.Select(ctx, nil)
	if iterator == nil {
		return nil
	}

	return iterator.Tx()
}

func (am *TOBMempool) Select(ctx context.Context, txs [][]byte) sdkmempool.Iterator {
	return am.index.Select(ctx, txs)
}

func (am *TOBMempool) CountTx() int {
	return am.index.CountTx()
}

// Contains returns true if the transaction is contained in the mempool.
func (am *TOBMempool) Contains(tx sdk.Tx) bool {
	_, txHashStr, err := utils.GetTxHashStr(am.txEncoder, tx)
	if err != nil {
		return false
	}

	_, ok := am.txIndex[txHashStr]
	return ok
}
