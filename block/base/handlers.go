package base

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/skip-mev/block-sdk/block/proposals"
	"github.com/skip-mev/block-sdk/block/utils"
)

// DefaultPrepareLaneHandler returns a default implementation of the PrepareLaneHandler. It
// selects all transactions in the mempool that are valid and not already in the partial
// proposal. It will continue to reap transactions until the maximum block space for this
// lane has been reached. Additionally, any transactions that are invalid will be returned.
func (l *BaseLane) DefaultPrepareLaneHandler() PrepareLaneHandler {
	return func(ctx sdk.Context, proposal proposals.Proposal, limit proposals.LaneLimits) ([]sdk.Tx, []sdk.Tx, error) {
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

			txInfo, err := utils.GetTxInfo(l.TxEncoder(), tx)
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
			if updatedGas := totalGas + txInfo.GasLimit; updatedGas > limit.MaxGasLimit {
				l.Logger().Info(
					"failed to select tx for lane; gas limit above the maximum allowed",
					"lane", l.Name(),
					"tx_gas", txInfo.GasLimit,
					"total_gas", totalGas,
					"max_gas", limit.MaxGasLimit,
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

// DefaultProcessLaneHandler returns a default implementation of the ProcessLaneHandler. It verifies
// the following invariants:
// 1. All transactions belong to this lane.
// 2. All transactions respect the priority defined by the mempool.
// 3. All transactions are valid respecting the verification logic of the lane.
func (l *BaseLane) DefaultProcessLaneHandler() ProcessLaneHandler {
	return func(ctx sdk.Context, partialProposal []sdk.Tx) error {
		// Process all transactions that match the lane's matcher.
		for index, tx := range partialProposal {
			if !l.Match(ctx, tx) {
				return fmt.Errorf("the %s lane contains a transaction that belongs to another lane", l.Name())
			}

			// If the transactions do not respect the priority defined by the mempool, we consider the proposal
			// to be invalid
			if index > 0 && l.Compare(ctx, partialProposal[index-1], tx) == -1 {
				return fmt.Errorf("transaction at index %d has a higher priority than %d", index, index-1)
			}

			if _, err := l.AnteVerifyTx(ctx, tx, false); err != nil {
				return fmt.Errorf("failed to verify tx: %w", err)
			}
		}

		// This means we have processed all transactions in the proposal.
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
