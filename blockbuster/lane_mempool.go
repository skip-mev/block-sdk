package blockbuster

import (
	"context"
	"errors"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkmempool "github.com/cosmos/cosmos-sdk/types/mempool"
	"github.com/skip-mev/pob/blockbuster/utils"
)

type (
	// ConstructorMempool defines a mempool that orders transactions based on the
	// txPriority. The mempool is a wrapper on top of the SDK's Priority Nonce mempool.
	// It include's additional helper functions that allow users to determine if a
	// transaction is already in the mempool and to compare the priority of two
	// transactions.
	ConstructorMempool[C comparable] struct {
		// index defines an index of transactions.
		index sdkmempool.Mempool

		// txPriority defines the transaction priority function. It is used to
		// retrieve the priority of a given transaction and to compare the priority
		// of two transactions. The index utilizes this struct to order transactions
		// in the mempool.
		txPriority TxPriority[C]

		// txEncoder defines the sdk.Tx encoder that allows us to encode transactions
		// to bytes.
		txEncoder sdk.TxEncoder

		// txCache is a map of all transactions in the mempool. It is used
		// to quickly check if a transaction is already in the mempool.
		txCache map[string]struct{}
	}
)

// NewConstructorMempool returns a new ConstructorMempool.
func NewConstructorMempool[C comparable](txPriority TxPriority[C], txEncoder sdk.TxEncoder, maxTx int) *ConstructorMempool[C] {
	return &ConstructorMempool[C]{
		index: NewPriorityMempool(
			PriorityNonceMempoolConfig[C]{
				TxPriority: txPriority,
				MaxTx:      maxTx,
			},
		),
		txPriority: txPriority,
		txEncoder:  txEncoder,
		txCache:    make(map[string]struct{}),
	}
}

// Insert inserts a transaction into the mempool.
func (cm *ConstructorMempool[C]) Insert(ctx context.Context, tx sdk.Tx) error {
	if err := cm.index.Insert(ctx, tx); err != nil {
		return fmt.Errorf("failed to insert tx into auction index: %w", err)
	}

	_, txHashStr, err := utils.GetTxHashStr(cm.txEncoder, tx)
	if err != nil {
		cm.Remove(tx)
		return err
	}

	cm.txCache[txHashStr] = struct{}{}

	return nil
}

// Remove removes a transaction from the mempool.
func (cm *ConstructorMempool[C]) Remove(tx sdk.Tx) error {
	if err := cm.index.Remove(tx); err != nil && !errors.Is(err, sdkmempool.ErrTxNotFound) {
		return fmt.Errorf("failed to remove transaction from the mempool: %w", err)
	}

	_, txHashStr, err := utils.GetTxHashStr(cm.txEncoder, tx)
	if err != nil {
		return fmt.Errorf("failed to get tx hash string: %w", err)
	}

	delete(cm.txCache, txHashStr)

	return nil
}

// Select returns an iterator of all transactions in the mempool. NOTE: If you
// remove a transaction from the mempool while iterating over the transactions,
// the iterator will not be aware of the removal and will continue to iterate
// over the removed transaction. Be sure to reset the iterator if you remove a transaction.
func (cm *ConstructorMempool[C]) Select(ctx context.Context, txs [][]byte) sdkmempool.Iterator {
	return cm.index.Select(ctx, txs)
}

// CountTx returns the number of transactions in the mempool.
func (cm *ConstructorMempool[C]) CountTx() int {
	return cm.index.CountTx()
}

// Contains returns true if the transaction is contained in the mempool.
func (cm *ConstructorMempool[C]) Contains(tx sdk.Tx) bool {
	_, txHashStr, err := utils.GetTxHashStr(cm.txEncoder, tx)
	if err != nil {
		return false
	}

	_, ok := cm.txCache[txHashStr]
	return ok
}

// Compare determines the relative priority of two transactions belonging in the same lane.
func (cm *ConstructorMempool[C]) Compare(ctx sdk.Context, this sdk.Tx, other sdk.Tx) int {
	firstPriority := cm.txPriority.GetTxPriority(ctx, this)
	secondPriority := cm.txPriority.GetTxPriority(ctx, other)
	return cm.txPriority.Compare(firstPriority, secondPriority)
}
