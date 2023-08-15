package blockbuster

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/skip-mev/pob/blockbuster/utils"
)

// DefaultPrepareLaneHandler returns a default implementation of the PrepareLaneHandler. It
// selects all transactions in the mempool that are valid and not already in the partial
// proposal. It will continue to reap transactions until the maximum block space for this
// lane has been reached. Additionally, any transactions that are invalid will be returned.
func (l *LaneConstructor) DefaultPrepareLaneHandler() PrepareLaneHandler {
	return func(ctx sdk.Context, proposal BlockProposal, maxTxBytes int64) ([][]byte, []sdk.Tx, error) {
		var (
			totalSize   int64
			txs         [][]byte
			txsToRemove []sdk.Tx
		)

		// Select all transactions in the mempool that are valid and not already in the
		// partial proposal.
		for iterator := l.Select(ctx, nil); iterator != nil; iterator = iterator.Next() {
			tx := iterator.Tx()

			txBytes, hash, err := utils.GetTxHashStr(l.TxEncoder(), tx)
			if err != nil {
				l.Logger().Info("failed to get hash of tx", "err", err)

				txsToRemove = append(txsToRemove, tx)
				continue
			}

			// Double check that the transaction belongs to this lane.
			if !l.Match(ctx, tx) {
				l.Logger().Info(
					"failed to select tx for lane; tx does not belong to lane",
					"tx_hash", hash,
					"lane", l.Name(),
				)

				txsToRemove = append(txsToRemove, tx)
				continue
			}

			// if the transaction is already in the (partial) block proposal, we skip it.
			if proposal.Contains(txBytes) {
				l.Logger().Info(
					"failed to select tx for lane; tx is already in proposal",
					"tx_hash", hash,
					"lane", l.Name(),
				)

				continue
			}

			// If the transaction is too large, we break and do not attempt to include more txs.
			txSize := int64(len(txBytes))
			if updatedSize := totalSize + txSize; updatedSize > maxTxBytes {
				l.Logger().Info(
					"tx bytes above the maximum allowed",
					"lane", l.Name(),
					"tx_size", txSize,
					"total_size", totalSize,
					"max_tx_bytes", maxTxBytes,
					"tx_hash", hash,
				)

				break
			}

			// Verify the transaction.
			if ctx, err = l.AnteVerifyTx(ctx, tx, false); err != nil {
				l.Logger().Info(
					"failed to verify tx",
					"tx_hash", hash,
					"err", err,
				)

				txsToRemove = append(txsToRemove, tx)
				continue
			}

			totalSize += txSize
			txs = append(txs, txBytes)
		}

		return txs, txsToRemove, nil
	}
}

// DefaultProcessLaneHandler returns a default implementation of the ProcessLaneHandler. It
// verifies all transactions in the lane that matches to the lane. If any transaction
// fails to verify, the entire proposal is rejected. If the handler comes across a transaction
// that does not match the lane's matcher, it will return the remaining transactions in the
// proposal.
func (l *LaneConstructor) DefaultProcessLaneHandler() ProcessLaneHandler {
	return func(ctx sdk.Context, txs []sdk.Tx) ([]sdk.Tx, error) {
		var err error

		// Process all transactions that match the lane's matcher.
		for index, tx := range txs {
			if l.Match(ctx, tx) {
				if ctx, err = l.AnteVerifyTx(ctx, tx, false); err != nil {
					return nil, fmt.Errorf("failed to verify tx: %w", err)
				}
			} else {
				return txs[index:], nil
			}
		}

		// This means we have processed all transactions in the proposal.
		return nil, nil
	}
}

// DefaultCheckOrderHandler returns a default implementation of the CheckOrderHandler. It
// ensures the following invariants:
//
//  1. All transactions that belong to this lane respect the ordering logic defined by the
//     lane.
//  2. Transactions that belong to other lanes cannot be interleaved with transactions that
//     belong to this lane.
func (l *LaneConstructor) DefaultCheckOrderHandler() CheckOrderHandler {
	return func(ctx sdk.Context, txs []sdk.Tx) error {
		seenOtherLaneTx := false

		for index, tx := range txs {
			if l.Match(ctx, tx) {
				if seenOtherLaneTx {
					return fmt.Errorf("the %s lane contains a transaction that belongs to another lane", l.Name())
				}

				// If the transactions do not respect the priority defined by the mempool, we consider the proposal
				// to be invalid
				if index > 0 && l.Compare(ctx, txs[index-1], tx) == -1 {
					return fmt.Errorf("transaction at index %d has a higher priority than %d", index, index-1)
				}
			} else {
				seenOtherLaneTx = true
			}
		}

		return nil
	}
}

// DefaultMatchHandler returns a default implementation of the MatchHandler. It matches all
// transactions.
func DefaultMatchHandler() MatchHandler {
	return func(ctx sdk.Context, tx sdk.Tx) bool {
		return true
	}
}
