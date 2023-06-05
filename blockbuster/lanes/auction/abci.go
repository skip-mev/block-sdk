package auction

import (
	"bytes"
	"errors"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/skip-mev/pob/blockbuster"
)

// PrepareLane will attempt to select the highest bid transaction that is valid
// and whose bundled transactions are valid and include them in the proposal. It
// will return an empty partial proposal if no valid bids are found.
func (l *TOBLane) PrepareLane(ctx sdk.Context, maxTxBytes int64, selectedTxs map[string][]byte) ([][]byte, error) {
	var tmpSelectedTxs [][]byte

	bidTxIterator := l.Select(ctx, nil)
	txsToRemove := make(map[sdk.Tx]struct{}, 0)

	// Attempt to select the highest bid transaction that is valid and whose
	// bundled transactions are valid.
selectBidTxLoop:
	for ; bidTxIterator != nil; bidTxIterator = bidTxIterator.Next() {
		cacheCtx, write := ctx.CacheContext()
		tmpBidTx := bidTxIterator.Tx()

		// if the transaction is already in the (partial) block proposal, we skip it.
		txHash, err := blockbuster.GetTxHashStr(l.cfg.TxEncoder, tmpBidTx)
		if err != nil {
			return nil, fmt.Errorf("failed to get bid tx hash: %w", err)
		}
		if _, ok := selectedTxs[txHash]; ok {
			continue selectBidTxLoop
		}

		bidTxBz, err := l.cfg.TxEncoder(tmpBidTx)
		if err != nil {
			txsToRemove[tmpBidTx] = struct{}{}
			continue selectBidTxLoop
		}

		bidTxSize := int64(len(bidTxBz))
		if bidTxSize <= maxTxBytes {
			// Verify the bid transaction and all of its bundled transactions.
			if err := l.VerifyTx(cacheCtx, tmpBidTx); err != nil {
				txsToRemove[tmpBidTx] = struct{}{}
				continue selectBidTxLoop
			}

			// Build the partial proposal by selecting the bid transaction and all of
			// its bundled transactions.
			bidInfo, err := l.GetAuctionBidInfo(tmpBidTx)
			if bidInfo == nil || err != nil {
				// Some transactions in the bundle may be malformed or invalid, so we
				// remove the bid transaction and try the next top bid.
				txsToRemove[tmpBidTx] = struct{}{}
				continue selectBidTxLoop
			}

			// store the bytes of each ref tx as sdk.Tx bytes in order to build a valid proposal
			bundledTxBz := make([][]byte, len(bidInfo.Transactions))
			for index, rawRefTx := range bidInfo.Transactions {
				sdkTx, err := l.WrapBundleTransaction(rawRefTx)
				if err != nil {
					txsToRemove[tmpBidTx] = struct{}{}
					continue selectBidTxLoop
				}

				sdkTxBz, err := l.cfg.TxEncoder(sdkTx)
				if err != nil {
					txsToRemove[tmpBidTx] = struct{}{}
					continue selectBidTxLoop
				}

				hash, err := blockbuster.GetTxHashStr(l.cfg.TxEncoder, sdkTx)
				if err != nil {
					txsToRemove[tmpBidTx] = struct{}{}
					continue selectBidTxLoop
				}

				// if the transaction is already in the (partial) block proposal, we skip it.
				if _, ok := selectedTxs[hash]; ok {
					continue selectBidTxLoop
				}

				bundleTxBz := make([]byte, len(sdkTxBz))
				copy(bundleTxBz, sdkTxBz)
				bundledTxBz[index] = sdkTxBz
			}

			// At this point, both the bid transaction itself and all the bundled
			// transactions are valid. So we select the bid transaction along with
			// all the bundled transactions. We also mark these transactions as seen and
			// update the total size selected thus far.
			tmpSelectedTxs = append(tmpSelectedTxs, bidTxBz)
			tmpSelectedTxs = append(tmpSelectedTxs, bundledTxBz...)

			// Write the cache context to the original context when we know we have a
			// valid top of block bundle.
			write()

			break selectBidTxLoop
		}

		txsToRemove[tmpBidTx] = struct{}{}
		l.cfg.Logger.Info(
			"failed to select auction bid tx; tx size is too large",
			"tx_size", bidTxSize,
			"max_size", maxTxBytes,
		)
	}

	// remove all invalid transactions from the mempool
	for tx := range txsToRemove {
		if err := l.Remove(tx); err != nil {
			return nil, err
		}
	}

	return tmpSelectedTxs, nil
}

// ProcessLane will ensure that block proposals that include transactions from
// the top-of-block auction lane are valid. It will return an error if the
// block proposal is invalid. The block proposal is invalid if it does not
// respect the ordering of transactions in the bid transaction or if the bid/bundled
// transactions are invalid.
func (l *TOBLane) ProcessLane(ctx sdk.Context, proposalTxs [][]byte, next blockbuster.ProcessLanesHandler) (sdk.Context, error) {
	// Track the index of the first transaction that does not belong to this lane.
	endIndex := 0

	for index, txBz := range proposalTxs {
		tx, err := l.cfg.TxDecoder(txBz)
		if err != nil {
			return ctx, err
		}

		bidInfo, err := l.GetAuctionBidInfo(tx)
		if err != nil {
			return ctx, fmt.Errorf("failed to get auction bid info for tx %w", err)
		}

		// If the transaction is an auction bid, then we need to ensure that it is
		// the first transaction in the block proposal and that the order of
		// transactions in the block proposal follows the order of transactions in
		// the bid.
		if bidInfo != nil {
			if index != 0 {
				return ctx, fmt.Errorf("block proposal did not place auction bid transaction at the top of the lane: %d", index)
			}

			bundledTransactions := bidInfo.Transactions
			if len(proposalTxs) < len(bundledTransactions)+1 {
				return ctx, errors.New("block proposal does not contain enough transactions to match the bundled transactions in the auction bid")
			}

			for i, refTxRaw := range bundledTransactions {
				// Wrap and then encode the bundled transaction to ensure that the underlying
				// reference transaction can be processed as an sdk.Tx.
				wrappedTx, err := l.WrapBundleTransaction(refTxRaw)
				if err != nil {
					return ctx, err
				}

				refTxBz, err := l.cfg.TxEncoder(wrappedTx)
				if err != nil {
					return ctx, err
				}

				if !bytes.Equal(refTxBz, proposalTxs[i+1]) {
					return ctx, errors.New("block proposal does not match the bundled transactions in the auction bid")
				}
			}

			// Verify the bid transaction.
			if err = l.VerifyTx(ctx, tx); err != nil {
				return ctx, err
			}

			endIndex += len(bundledTransactions) + 1
		}
	}

	return next(ctx, proposalTxs[endIndex:])
}

// VerifyTx will verify that the bid transaction and all of its bundled
// transactions are valid. It will return an error if any of the transactions
// are invalid.
func (l *TOBLane) VerifyTx(ctx sdk.Context, bidTx sdk.Tx) error {
	bidInfo, err := l.GetAuctionBidInfo(bidTx)
	if err != nil {
		return fmt.Errorf("failed to get auction bid info: %w", err)
	}

	// verify the top-level bid transaction
	ctx, err = l.verifyTx(ctx, bidTx)
	if err != nil {
		return fmt.Errorf("invalid bid tx; failed to execute ante handler: %w", err)
	}

	// verify all of the bundled transactions
	for _, tx := range bidInfo.Transactions {
		bundledTx, err := l.WrapBundleTransaction(tx)
		if err != nil {
			return fmt.Errorf("invalid bid tx; failed to decode bundled tx: %w", err)
		}

		// bid txs cannot be included in bundled txs
		bidInfo, _ := l.GetAuctionBidInfo(bundledTx)
		if bidInfo != nil {
			return fmt.Errorf("invalid bid tx; bundled tx cannot be a bid tx")
		}

		if ctx, err = l.verifyTx(ctx, bundledTx); err != nil {
			return fmt.Errorf("invalid bid tx; failed to execute bundled transaction: %w", err)
		}
	}

	return nil
}

// verifyTx will execute the ante handler on the transaction and return the
// resulting context and error.
func (l *TOBLane) verifyTx(ctx sdk.Context, tx sdk.Tx) (sdk.Context, error) {
	if l.cfg.AnteHandler != nil {
		newCtx, err := l.cfg.AnteHandler(ctx, tx, false)
		return newCtx, err
	}

	return ctx, nil
}
