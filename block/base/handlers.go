package base

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/skip-mev/block-sdk/block"
)

// DefaultPrepareLaneHandler returns a default implementation of the PrepareLaneHandler. It
// selects all transactions in the mempool that are valid and not already in the partial
// proposal. It will continue to reap transactions until the maximum block space for this
// lane has been reached. Additionally, any transactions that are invalid will be returned.
func (l *BaseLane) DefaultPrepareLaneHandler() PrepareLaneHandler {
	return func(ctx sdk.Context, proposal block.BlockProposal, limit block.LaneLimits) ([]sdk.Tx, []sdk.Tx, error) {
		var (
			totalSize    int64
			totalGas     uint64
			txsToInclude []sdk.Tx
			txsToRemove  []sdk.Tx
		)

		// Select all transactions in the mempool that are valid and not already in the
		// partial proposal.
		for iterator := l.Select(ctx, nil); iterator != nil; iterator = iterator.Next() {
			tx := iterator.Tx()

			txInfo, err := block.GetTxInfo(l.TxEncoder(), tx)
			if err != nil {
				l.Logger().Info("failed to get hash of tx", "err", err)

				txsToRemove = append(txsToRemove, tx)
				continue
			}

			// Double check that the transaction belongs to this lane.
			if !l.Match(ctx, tx) {
				l.Logger().Info(
					"failed to select tx for lane; tx does not belong to lane",
					"tx_hash", txInfo.Hash,
					"lane", l.Name(),
				)

				txsToRemove = append(txsToRemove, tx)
				continue
			}

			// if the transaction is already in the (partial) block proposal, we skip it.
			if proposal.Contains(txInfo.Hash) {
				l.Logger().Info(
					"failed to select tx for lane; tx is already in proposal",
					"tx_hash", txInfo.Hash,
					"lane", l.Name(),
				)

				continue
			}

			// If the transaction is too large, we break and do not attempt to include more txs.
			if updatedSize := totalSize + txInfo.Size; updatedSize > limit.MaxTxBytes {
				l.Logger().Info(
					"failed to select tx for lane; tx bytes above the maximum allowed",
					"lane", l.Name(),
					"tx_size", txInfo.Size,
					"total_size", totalSize,
					"max_tx_bytes", limit.MaxTxBytes,
					"tx_hash", txInfo.Hash,
				)

				// TODO: Determine if there is any trade off with breaking or continuing here.
				continue
			}

			// If the gas limit of the transaction is too large, we break and do not attempt to include more txs.
			if updatedGas := totalGas + txInfo.GasLimit; updatedGas > limit.MaxGas {
				l.Logger().Info(
					"failed to select tx for lane; gas limit above the maximum allowed",
					"lane", l.Name(),
					"tx_gas", txInfo.GasLimit,
					"total_gas", totalGas,
					"max_gas", limit.MaxGas,
					"tx_hash", txInfo.Hash,
				)

				// TODO: Determine if there is any trade off with breaking or continuing here.
				continue
			}

			// Verify the transaction.
			if ctx, err = l.AnteVerifyTx(ctx, tx, false); err != nil {
				l.Logger().Info(
					"failed to verify tx",
					"tx_hash", txInfo.Hash,
					"err", err,
				)

				txsToRemove = append(txsToRemove, tx)
				continue
			}

			totalSize += txInfo.Size
			totalGas += txInfo.GasLimit
			txsToInclude = append(txsToInclude, tx)
		}

		return txsToInclude, txsToRemove, nil
	}
}

// DefaultProcessLaneHandler returns a default implementation of the ProcessLaneHandler. It
// verifies all transactions in the lane that matches to the lane. If any transaction
// fails to verify, the entire proposal is rejected. If the handler comes across a transaction
// that does not match the lane's matcher, it will return the remaining transactions in the
// proposal.
func (l *BaseLane) DefaultProcessLaneHandler() ProcessLaneHandler {
	return func(ctx sdk.Context, txs []sdk.Tx, limit block.LaneLimits) ([]sdk.Tx, error) {
		var (
			totalGas  uint64
			totalSize int64
		)

		// Process all transactions that match the lane's matcher.
		for index, tx := range txs {
			if l.Match(ctx, tx) {
				txInfo, err := block.GetTxInfo(l.TxEncoder(), tx)
				if err != nil {
					return nil, fmt.Errorf("failed to get info on tx: %w", err)
				}

				if ctx, err = l.AnteVerifyTx(ctx, tx, false); err != nil {
					return nil, fmt.Errorf("failed to verify tx: %w", err)
				}

				totalGas += txInfo.GasLimit
				totalSize += txInfo.Size

				// invariant: the transactions must consume less than the maximum allowed gas and size.
				if totalGas > limit.MaxGas || totalSize > limit.MaxTxBytes {
					return nil, fmt.Errorf(
						"lane %s has exceeded its gas or size limit; max_gas=%d, max_size=%d, total_gas=%d, total_size=%d",
						l.Name(),
						limit.MaxGas,
						limit.MaxTxBytes,
						totalGas,
						totalSize,
					)
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
func (l *BaseLane) DefaultCheckOrderHandler() CheckOrderHandler {
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
