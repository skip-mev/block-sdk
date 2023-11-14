package mev

import (
	"bytes"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/skip-mev/block-sdk/block/base"
	"github.com/skip-mev/block-sdk/block/proposals"
	"github.com/skip-mev/block-sdk/block/utils"
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
		for iterator := l.Select(ctx, nil); iterator != nil; iterator = iterator.Next() {
			bidTx := iterator.Tx()

			if !l.Match(ctx, bidTx) {
				l.Logger().Info("failed to select auction bid tx for lane; tx does not match lane")

				txsToRemove = append(txsToRemove, bidTx)
				continue
			}

			bundle, err := l.VerifyBidBasic(bidTx, proposal, limit)
			if err != nil {
				l.Logger().Info(
					"failed to select auction bid tx for lane; tx is invalid",
					"err", err,
				)

				txsToRemove = append(txsToRemove, bidTx)
				continue
			}

			cacheCtx, write := ctx.CacheContext()
			if err := l.VerifyBidTx(cacheCtx, bidTx, bundle); err != nil {
				l.Logger().Info(
					"failed to select auction bid tx for lane; tx is invalid",
					"err", err,
				)

				txsToRemove = append(txsToRemove, bidTx)
				continue
			}

			// At this point, both the bid transaction itself and all the bundled
			// transactions are valid. So we select the bid transaction along with
			// all the bundled transactions. We also mark these transactions as seen and
			// update the total size selected thus far.
			txsToInclude = append(txsToInclude, bidTx)
			txsToInclude = append(txsToInclude, bundle...)

			// Write the cache context to the original context when we know we have a
			// valid bundle.
			write()

			break
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

		if bidInfo == nil {
			return fmt.Errorf("bid info is nil")
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

		// Ensure the transactions in the proposal match the bundled transactions in the bid transaction.
		bundle := partialProposal[1:]
		for index, bundledTxBz := range bidInfo.Transactions {
			bundledTx, err := l.WrapBundleTransaction(bundledTxBz)
			if err != nil {
				return fmt.Errorf("invalid bid tx; failed to decode bundled tx: %w", err)
			}

			expectedTxBz, err := l.TxEncoder()(bundledTx)
			if err != nil {
				return fmt.Errorf("invalid bid tx; failed to encode bundled tx: %w", err)
			}

			actualTxBz, err := l.TxEncoder()(bundle[index])
			if err != nil {
				return fmt.Errorf("invalid bid tx; failed to encode tx: %w", err)
			}

			// Verify that the bundled transaction matches the transaction in the block proposal.
			if !bytes.Equal(actualTxBz, expectedTxBz) {
				return fmt.Errorf("invalid bid tx; bundled tx does not match tx in block proposal")
			}
		}

		// Verify the top-level bid transaction.
		if err := l.VerifyBidTx(ctx, bidTx, bundle); err != nil {
			return fmt.Errorf("invalid bid tx; failed to verify bid tx: %w", err)
		}

		return nil
	}
}

// VerifyBidBasic will verify that the bid transaction and all of its bundled
// transactions respect the basic invariants of the lane (e.g. size, gas limit).
func (l *MEVLane) VerifyBidBasic(
	bidTx sdk.Tx,
	proposal proposals.Proposal,
	limit proposals.LaneLimits,
) ([]sdk.Tx, error) {
	// Verify the transaction is a bid transaction.
	bidInfo, err := l.GetAuctionBidInfo(bidTx)
	if err != nil {
		return nil, fmt.Errorf("failed to get bid info from auction bid tx for lane %s: %w", l.Name(), err)
	}

	if bidInfo == nil {
		return nil, fmt.Errorf("bid info is nil")
	}

	txInfo, err := utils.GetTxInfo(l.TxEncoder(), bidTx)
	if err != nil {
		return nil, fmt.Errorf("err retrieving transaction info: %s", err)
	}

	// This should never happen, but we check just in case.
	if proposal.Contains(txInfo.Hash) {
		return nil, fmt.Errorf("invalid bid tx; bid tx is already in the proposal")
	}

	totalSize := txInfo.Size
	totalGasLimit := txInfo.GasLimit
	bundle := make([]sdk.Tx, len(bidInfo.Transactions))

	// Verify size and gas limit of the bundled transactions.
	for index, bundledTxBz := range bidInfo.Transactions {
		bundledTx, err := l.WrapBundleTransaction(bundledTxBz)
		if err != nil {
			return nil, fmt.Errorf("invalid bid tx; failed to decode bundled tx: %w", err)
		}

		bundledTxInfo, err := utils.GetTxInfo(l.TxEncoder(), bundledTx)
		if err != nil {
			return nil, fmt.Errorf("err retrieving transaction info: %s", err)
		}

		if proposal.Contains(bundledTxInfo.Hash) {
			return nil, fmt.Errorf("invalid bid tx; bundled tx is already in the proposal")
		}

		totalSize += bundledTxInfo.Size
		totalGasLimit += bundledTxInfo.GasLimit
		bundle[index] = bundledTx
	}

	if totalSize > limit.MaxTxBytes {
		return nil, fmt.Errorf(
			"partial proposal is too large: %d > %d",
			totalSize,
			limit.MaxTxBytes,
		)
	}

	if totalGasLimit > limit.MaxGasLimit {
		return nil, fmt.Errorf(
			"partial proposal consumes too much gas: %d > %d",
			totalGasLimit,
			limit.MaxGasLimit,
		)
	}

	return bundle, nil
}

// VerifyBidTx will verify that the bid transaction and all of its bundled
// transactions are valid.
func (l *MEVLane) VerifyBidTx(ctx sdk.Context, bidTx sdk.Tx, bundle []sdk.Tx) error {
	bidInfo, err := l.GetAuctionBidInfo(bidTx)
	if err != nil {
		return fmt.Errorf("failed to get bid info from auction bid tx for lane %s: %w", l.Name(), err)
	}

	if bidInfo == nil {
		return fmt.Errorf("bid info is nil")
	}

	// verify the top-level bid transaction
	if err = l.VerifyTx(ctx, bidTx, false); err != nil {
		return fmt.Errorf("invalid bid tx; failed to execute ante handler: %w", err)
	}

	// verify all of the bundled transactions
	for _, bundledTx := range bundle {
		if l.Match(ctx, bundledTx) {
			return fmt.Errorf("invalid bid tx; bundled tx is another bid transaction")
		}

		if err = l.VerifyTx(ctx, bundledTx, false); err != nil {
			return fmt.Errorf("invalid bid tx; failed to execute bundled transaction: %w", err)
		}
	}

	return nil
}
