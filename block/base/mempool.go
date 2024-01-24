package base

import (
	"context"
	"errors"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkmempool "github.com/cosmos/cosmos-sdk/types/mempool"

	signer_extraction "github.com/skip-mev/block-sdk/v2/adapters/signer_extraction_adapter"
	"github.com/skip-mev/block-sdk/v2/block/utils"
)

type (
	// Mempool defines a mempool that orders transactions based on the
	// txPriority. The mempool is a wrapper on top of the SDK's Priority Nonce mempool.
	// It include's additional helper functions that allow users to determine if a
	// transaction is already in the mempool and to compare the priority of two
	// transactions.
	Mempool[C comparable] struct {
		// index defines an index of transactions.
		index sdkmempool.Mempool

		// signerExtractor defines the signer extraction adapter that allows us to
		// extract the signer from a transaction.
		extractor signer_extraction.Adapter

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

// NewMempool returns a new Mempool.
func NewMempool[C comparable](txPriority TxPriority[C], txEncoder sdk.TxEncoder, extractor signer_extraction.Adapter, maxTx int) *Mempool[C] {
	return &Mempool[C]{
		index: NewPriorityMempool(
			PriorityNonceMempoolConfig[C]{
				TxPriority: txPriority,
				MaxTx:      maxTx,
			},
			extractor,
		),
		extractor:  extractor,
		txPriority: txPriority,
		txEncoder:  txEncoder,
		txCache:    make(map[string]struct{}),
	}
}

// Priority returns the priority of the transaction.
func (cm *Mempool[C]) Priority(ctx sdk.Context, tx sdk.Tx) any {
	return cm.txPriority.GetTxPriority(ctx, tx)
}

// Insert inserts a transaction into the mempool.
func (cm *Mempool[C]) Insert(ctx context.Context, tx sdk.Tx) error {
	if err := cm.index.Insert(ctx, tx); err != nil {
		return fmt.Errorf("failed to insert tx into auction index: %w", err)
	}

	hash, err := utils.GetTxHash(cm.txEncoder, tx)
	if err != nil {
		cm.Remove(tx)
		return err
	}

	cm.txCache[hash] = struct{}{}

	return nil
}

// Remove removes a transaction from the mempool.
func (cm *Mempool[C]) Remove(tx sdk.Tx) error {
	if err := cm.index.Remove(tx); err != nil && !errors.Is(err, sdkmempool.ErrTxNotFound) {
		return fmt.Errorf("failed to remove transaction from the mempool: %w", err)
	}

	hash, err := utils.GetTxHash(cm.txEncoder, tx)
	if err != nil {
		return fmt.Errorf("failed to get tx hash string: %w", err)
	}

	delete(cm.txCache, hash)

	return nil
}

// Select returns an iterator of all transactions in the mempool. NOTE: If you
// remove a transaction from the mempool while iterating over the transactions,
// the iterator will not be aware of the removal and will continue to iterate
// over the removed transaction. Be sure to reset the iterator if you remove a transaction.
func (cm *Mempool[C]) Select(ctx context.Context, txs [][]byte) sdkmempool.Iterator {
	return cm.index.Select(ctx, txs)
}

// CountTx returns the number of transactions in the mempool.
func (cm *Mempool[C]) CountTx() int {
	return cm.index.CountTx()
}

// Contains returns true if the transaction is contained in the mempool.
func (cm *Mempool[C]) Contains(tx sdk.Tx) bool {
	hash, err := utils.GetTxHash(cm.txEncoder, tx)
	if err != nil {
		return false
	}

	_, ok := cm.txCache[hash]
	return ok
}

// Compare determines the relative priority of two transactions belonging in the same lane.
// There are two cases to consider:
//  1. The transactions have the same signer. In this case, we compare the sequence numbers.
//  2. The transactions have different signers. In this case, we compare the priorities of the
//     transactions.
//
// Compare will return -1 if this transaction has a lower priority than the other transaction, 0 if
// they have the same priority, and 1 if this transaction has a higher priority than the other transaction.
func (cm *Mempool[C]) Compare(ctx sdk.Context, this sdk.Tx, other sdk.Tx) (int, error) {
	signers, err := cm.extractor.GetSigners(this)
	if err != nil {
		return 0, err
	}
	if len(signers) == 0 {
		return 0, fmt.Errorf("expected one signer for the first transaction")
	}
	// The priority nonce mempool uses the first tx signer so this is a safe operation.
	thisSignerInfo := signers[0]

	signers, err = cm.extractor.GetSigners(other)
	if err != nil {
		return 0, err
	}
	if len(signers) == 0 {
		return 0, fmt.Errorf("expected one signer for the second transaction")
	}
	otherSignerInfo := signers[0]

	// If the signers are the same, we compare the sequence numbers.
	if thisSignerInfo.Signer.Equals(otherSignerInfo.Signer) {
		switch {
		case thisSignerInfo.Sequence < otherSignerInfo.Sequence:
			return 1, nil
		case thisSignerInfo.Sequence > otherSignerInfo.Sequence:
			return -1, nil
		default:
			// This case should never happen but we add in the case for completeness.
			return 0, fmt.Errorf("the two transactions have the same sequence number")
		}
	}

	// Determine the priority and compare the priorities.
	firstPriority := cm.txPriority.GetTxPriority(ctx, this)
	secondPriority := cm.txPriority.GetTxPriority(ctx, other)
	return cm.txPriority.Compare(firstPriority, secondPriority), nil
}
