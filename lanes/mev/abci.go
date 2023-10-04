package mev

import (
	"bytes"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/skip-mev/block-sdk/block/base"
	"github.com/skip-mev/block-sdk/block/proposals"
	"github.com/skip-mev/block-sdk/block/utils"
	"github.com/skip-mev/block-sdk/x/auction/types"
)

// PrepareLaneHandler will attempt to select the highest bid transaction that is valid
// and whose bundled transactions are valid and include them in the proposal. It
// will return no transactions if no valid bids are found. If any of the bids are invalid,
// it will return them and will only remove the bids and not the bundled transactions.
func (l *MEVLane) PrepareLaneHandler() base.PrepareLaneHandler {
	return func(ctx sdk.Context, proposal proposals.Proposal, limit proposals.LaneLimits) ([]sdk.Tx, []sdk.Tx, error) {
		// Define all of the info we need to select transactions for the partial proposal.
		var (
			txsToInclude []sdk.Tx
			txsToRemove  []sdk.Tx
		)

		// Attempt to select the highest bid transaction that is valid and whose
		// bundled transactions are valid.
		bidTxIterator := l.Select(ctx, nil)
	selectBidTxLoop:
		for ; bidTxIterator != nil; bidTxIterator = bidTxIterator.Next() {
			cacheCtx, write := ctx.CacheContext()
			bidTx := bidTxIterator.Tx()

			txInfo, err := utils.GetTxInfo(l.TxEncoder(), bidTx)
			if err != nil {
				l.Logger().Info("failed to get hash of auction bid tx", "err", err)

				txsToRemove = append(txsToRemove, bidTx)
				continue selectBidTxLoop
			}

			// if the transaction is already in the (partial) block proposal, we skip it.
			//
			// TODO: Should we really be panic'ing here?
			if proposal.Contains(txInfo.Hash) {
				l.Logger().Info(
					"failed to select auction bid tx for lane; tx is already in proposal",
					"tx_hash", txInfo.Hash,
				)

				continue selectBidTxLoop
			}

			if txInfo.Size > limit.MaxTxBytes {
				l.Logger().Info(
					"failed to select auction bid tx for lane; tx size is too large",
					"tx_size", txInfo.Size,
					"max_size", limit.MaxTxBytes,
					"tx_hash", txInfo.Hash,
				)

				txsToRemove = append(txsToRemove, bidTx)
				continue selectBidTxLoop
			}

			if txInfo.GasLimit > limit.MaxGasLimit {
				l.Logger().Info(
					"failed to select auction bid tx for lane; tx gas limit is too large",
					"tx_gas_limit", txInfo.GasLimit,
					"max_gas_limit", limit.MaxGasLimit,
					"tx_hash", txInfo.Hash,
				)

				txsToRemove = append(txsToRemove, bidTx)
				continue selectBidTxLoop
			}

			// Build the partial proposal by selecting the bid transaction and all of
			// its bundled transactions.
			bidInfo, err := l.GetAuctionBidInfo(bidTx)
			if err != nil {
				l.Logger().Info(
					"failed to get auction bid info",
					"tx_hash", txInfo.Hash,
					"err", err,
				)

				// Some transactions in the bundle may be malformed or invalid, so we
				// remove the bid transaction and try the next top bid.
				txsToRemove = append(txsToRemove, bidTx)
				continue selectBidTxLoop
			}

			// Verify the bid transaction and all of its bundled transactions.
			if err := l.VerifyTx(cacheCtx, bidTx, bidInfo); err != nil {
				l.Logger().Info(
					"failed to verify auction bid tx",
					"tx_hash", txInfo.Hash,
					"err", err,
				)

				txsToRemove = append(txsToRemove, bidTx)
				continue selectBidTxLoop
			}

			// store the bytes of each ref tx as sdk.Tx bytes in order to build a valid proposal
			gasLimitSum := txInfo.GasLimit
			bundledTxs := make([]sdk.Tx, len(bidInfo.Transactions))
			for index, bundledTxBz := range bidInfo.Transactions {
				bundleTx, err := l.WrapBundleTransaction(bundledTxBz)
				if err != nil {
					l.Logger().Info(
						"failed to wrap bundled tx",
						"tx_hash", txInfo.Hash,
						"err", err,
					)

					txsToRemove = append(txsToRemove, bidTx)
					continue selectBidTxLoop
				}

				bundledTxInfo, err := utils.GetTxInfo(l.TxEncoder(), bundleTx)
				if err != nil {
					l.Logger().Info(
						"failed to get hash of bundled tx",
						"tx_hash", txInfo.Hash,
						"err", err,
					)

					txsToRemove = append(txsToRemove, bidTx)
					continue selectBidTxLoop
				}

				// if the transaction is already in the (partial) block proposal, we skip it.
				if proposal.Contains(bundledTxInfo.Hash) {
					l.Logger().Info(
						"failed to select auction bid tx for lane; tx is already in proposal",
						"tx_hash", bundledTxInfo.Hash,
					)

					continue selectBidTxLoop
				}

				// If the bundled transaction is a bid transaction, we skip it.
				if l.Match(ctx, bundleTx) {
					l.Logger().Info(
						"failed to select auction bid tx for lane; bundled tx is another bid transaction",
						"tx_hash", bundledTxInfo.Hash,
					)

					txsToRemove = append(txsToRemove, bidTx)
					continue selectBidTxLoop
				}

				if gasLimitSum += bundledTxInfo.GasLimit; gasLimitSum > limit.MaxGasLimit {
					l.Logger().Info(
						"failed to select auction bid tx for lane; tx gas limit is too large",
						"tx_gas_limit", gasLimitSum,
						"max_gas_limit", limit.MaxGasLimit,
						"tx_hash", txInfo.Hash,
					)

					txsToRemove = append(txsToRemove, bidTx)
					continue selectBidTxLoop
				}

				bundleTxBz := make([]byte, bundledTxInfo.Size)
				copy(bundleTxBz, bundledTxInfo.TxBytes)
				bundledTxs[index] = bundleTx
			}

			// At this point, both the bid transaction itself and all the bundled
			// transactions are valid. So we select the bid transaction along with
			// all the bundled transactions. We also mark these transactions as seen and
			// update the total size selected thus far.
			txsToInclude = append(txsToInclude, bidTx)
			txsToInclude = append(txsToInclude, bundledTxs...)

			// Write the cache context to the original context when we know we have a
			// valid bundle.
			write()

			break selectBidTxLoop
		}

		return txsToInclude, txsToRemove, nil
	}
}

// ProcessLaneHandler will ensure that block proposals that include transactions from
// the mev lane are valid. In particular, the invariant checks that we perform are:
//  1. The first transaction in the partial block proposal must be a bid transaction.
//  2. The bid transaction must be valid.
//  3. The bundled transactions must be valid.
//  4. The bundled transactions must match the transactions in the block proposal in the
//     same order they were defined in the bid transaction.
//  5. The bundled transactions must not be bid transactions.
func (l *MEVLane) ProcessLaneHandler() base.ProcessLaneHandler {
	return func(ctx sdk.Context, partialProposal []sdk.Tx) error {
		if len(partialProposal) == 0 {
			return nil
		}

		// If the first transaction does not match the lane, then we return an error.
		bidTx := partialProposal[0]
		if !l.Match(ctx, bidTx) {
			return fmt.Errorf("expected first transaction in lane %s to be a bid transaction", l.Name())
		}

		bidInfo, err := l.GetAuctionBidInfo(bidTx)
		if err != nil {
			return fmt.Errorf("failed to get bid info from auction bid tx for lane %s: %w", l.Name(), err)
		}

		// Check that all bundled transactions were included.
		if len(bidInfo.Transactions)+1 != len(partialProposal) {
			return fmt.Errorf(
				"expected %d transactions in lane %s but got %d",
				len(bidInfo.Transactions)+1,
				l.Name(),
				len(partialProposal),
			)
		}

		// Verify the top-level bid transaction.
		if ctx, err = l.AnteVerifyTx(ctx, bidTx, false); err != nil {
			return fmt.Errorf("invalid bid tx; failed to execute ante handler: %w", err)
		}

		// Verify all of the bundled transactions.
		for index, bundledTxBz := range bidInfo.Transactions {
			bundledTx, err := l.WrapBundleTransaction(bundledTxBz)
			if err != nil {
				return fmt.Errorf("invalid bid tx; failed to decode bundled tx: %w", err)
			}

			txBz, err := l.TxEncoder()(partialProposal[index+1])
			if err != nil {
				return fmt.Errorf("invalid bid tx; failed to encode tx: %w", err)
			}

			// Verify that the bundled transaction matches the transaction in the block proposal.
			if !bytes.Equal(bundledTxBz, txBz) {
				return fmt.Errorf("invalid bid tx; bundled tx does not match tx in block proposal")
			}

			// Verify this is not another bid transaction.
			if l.Match(ctx, bundledTx) {
				return fmt.Errorf("invalid bid tx; bundled tx is another bid transaction")
			}

			if ctx, err = l.AnteVerifyTx(ctx, bundledTx, false); err != nil {
				return fmt.Errorf("invalid bid tx; failed to execute bundled transaction: %w", err)
			}
		}

		return nil
	}
}

// VerifyTx will verify that the bid transaction and all of its bundled
// transactions are valid. It will return an error if any of the transactions
// are invalid.
func (l *MEVLane) VerifyTx(ctx sdk.Context, bidTx sdk.Tx, bidInfo *types.BidInfo) (err error) {
	if bidInfo == nil {
		return fmt.Errorf("bid info is nil")
	}

	// verify the top-level bid transaction
	if ctx, err = l.AnteVerifyTx(ctx, bidTx, false); err != nil {
		return fmt.Errorf("invalid bid tx; failed to execute ante handler: %w", err)
	}

	// verify all of the bundled transactions
	for _, tx := range bidInfo.Transactions {
		bundledTx, err := l.WrapBundleTransaction(tx)
		if err != nil {
			return fmt.Errorf("invalid bid tx; failed to decode bundled tx: %w", err)
		}

		if ctx, err = l.AnteVerifyTx(ctx, bundledTx, false); err != nil {
			return fmt.Errorf("invalid bid tx; failed to execute bundled transaction: %w", err)
		}
	}

	return nil
}
