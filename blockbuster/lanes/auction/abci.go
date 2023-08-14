package auction

import (
	"bytes"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/skip-mev/pob/blockbuster"
	"github.com/skip-mev/pob/blockbuster/utils"
	"github.com/skip-mev/pob/x/builder/types"
)

// PrepareLaneHandler will attempt to select the highest bid transaction that is valid
// and whose bundled transactions are valid and include them in the proposal. It
// will return no transactions if no valid bids are found. If any of the bids are invalid,
// it will return them and will only remove the bids and not the bundled transactions.
func (l *TOBLane) PrepareLaneHandler() blockbuster.PrepareLaneHandler {
	return func(ctx sdk.Context, proposal blockbuster.BlockProposal, maxTxBytes int64) ([][]byte, []sdk.Tx, error) {
		// Define all of the info we need to select transactions for the partial proposal.
		var (
			txs         [][]byte
			txsToRemove []sdk.Tx
		)

		// Attempt to select the highest bid transaction that is valid and whose
		// bundled transactions are valid.
		bidTxIterator := l.Select(ctx, nil)
	selectBidTxLoop:
		for ; bidTxIterator != nil; bidTxIterator = bidTxIterator.Next() {
			cacheCtx, write := ctx.CacheContext()
			tmpBidTx := bidTxIterator.Tx()

			bidTxBz, hash, err := utils.GetTxHashStr(l.TxEncoder(), tmpBidTx)
			if err != nil {
				l.Logger().Info("failed to get hash of auction bid tx", "err", err)

				txsToRemove = append(txsToRemove, tmpBidTx)
				continue selectBidTxLoop
			}

			// if the transaction is already in the (partial) block proposal, we skip it.
			if proposal.Contains(bidTxBz) {
				l.Logger().Info(
					"failed to select auction bid tx for lane; tx is already in proposal",
					"tx_hash", hash,
				)

				continue selectBidTxLoop
			}

			bidTxSize := int64(len(bidTxBz))
			if bidTxSize <= maxTxBytes {
				// Build the partial proposal by selecting the bid transaction and all of
				// its bundled transactions.
				bidInfo, err := l.GetAuctionBidInfo(tmpBidTx)
				if err != nil {
					l.Logger().Info(
						"failed to get auction bid info",
						"tx_hash", hash,
						"err", err,
					)

					// Some transactions in the bundle may be malformed or invalid, so we
					// remove the bid transaction and try the next top bid.
					txsToRemove = append(txsToRemove, tmpBidTx)
					continue selectBidTxLoop
				}

				// Verify the bid transaction and all of its bundled transactions.
				if err := l.VerifyTx(cacheCtx, tmpBidTx, bidInfo); err != nil {
					l.Logger().Info(
						"failed to verify auction bid tx",
						"tx_hash", hash,
						"err", err,
					)

					txsToRemove = append(txsToRemove, tmpBidTx)
					continue selectBidTxLoop
				}

				// store the bytes of each ref tx as sdk.Tx bytes in order to build a valid proposal
				bundledTxBz := make([][]byte, len(bidInfo.Transactions))
				for index, rawRefTx := range bidInfo.Transactions {
					sdkTx, err := l.WrapBundleTransaction(rawRefTx)
					if err != nil {
						l.Logger().Info(
							"failed to wrap bundled tx",
							"tx_hash", hash,
							"err", err,
						)

						txsToRemove = append(txsToRemove, tmpBidTx)
						continue selectBidTxLoop
					}

					sdkTxBz, _, err := utils.GetTxHashStr(l.TxEncoder(), sdkTx)
					if err != nil {
						l.Logger().Info(
							"failed to get hash of bundled tx",
							"tx_hash", hash,
							"err", err,
						)

						txsToRemove = append(txsToRemove, tmpBidTx)
						continue selectBidTxLoop
					}

					// if the transaction is already in the (partial) block proposal, we skip it.
					if proposal.Contains(sdkTxBz) {
						l.Logger().Info(
							"failed to select auction bid tx for lane; tx is already in proposal",
							"tx_hash", hash,
						)

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
				txs = append(txs, bidTxBz)
				txs = append(txs, bundledTxBz...)

				// Write the cache context to the original context when we know we have a
				// valid top of block bundle.
				write()

				break selectBidTxLoop
			}

			l.Logger().Info(
				"failed to select auction bid tx for lane; tx size is too large",
				"tx_size", bidTxSize,
				"max_size", maxTxBytes,
			)
		}

		return txs, txsToRemove, nil
	}
}

// ProcessLaneHandler will ensure that block proposals that include transactions from
// the top-of-block auction lane are valid.
func (l *TOBLane) ProcessLaneHandler() blockbuster.ProcessLaneHandler {
	return func(ctx sdk.Context, txs []sdk.Tx) ([]sdk.Tx, error) {
		if len(txs) == 0 {
			return txs, nil
		}

		bidTx := txs[0]
		if !l.Match(ctx, bidTx) {
			return txs, nil
		}

		bidInfo, err := l.GetAuctionBidInfo(bidTx)
		if err != nil {
			return nil, fmt.Errorf("failed to get bid info for lane %s: %w", l.Name(), err)
		}

		if err := l.VerifyTx(ctx, bidTx, bidInfo); err != nil {
			return nil, fmt.Errorf("invalid bid tx: %w", err)
		}

		return txs[len(bidInfo.Transactions)+1:], nil
	}
}

// CheckOrderHandler ensures that if a bid transaction is present in a proposal,
//   - it is the first transaction in the partial proposal
//   - all of the bundled transactions are included after the bid transaction in the order
//     they were included in the bid transaction.
//   - there are no other bid transactions in the proposal
//   - transactions from other lanes are not interleaved with transactions from the bid
//     transaction.
func (l *TOBLane) CheckOrderHandler() blockbuster.CheckOrderHandler {
	return func(ctx sdk.Context, txs []sdk.Tx) error {
		if len(txs) == 0 {
			return nil
		}

		bidTx := txs[0]

		// If there is a bid transaction, it must be the first transaction in the block proposal.
		if !l.Match(ctx, bidTx) {
			for _, tx := range txs[1:] {
				if l.Match(ctx, tx) {
					return fmt.Errorf("misplaced bid transactions in lane %s", l.Name())
				}
			}

			return nil
		}

		bidInfo, err := l.GetAuctionBidInfo(bidTx)
		if err != nil {
			return fmt.Errorf("failed to get bid info for lane %s: %w", l.Name(), err)
		}

		if len(txs) < len(bidInfo.Transactions)+1 {
			return fmt.Errorf(
				"invalid number of transactions in lane %s; expected at least %d, got %d",
				l.Name(),
				len(bidInfo.Transactions)+1,
				len(txs),
			)
		}

		// Ensure that the order of transactions in the bundle is preserved.
		for i, bundleTx := range txs[1 : len(bidInfo.Transactions)+1] {
			if l.Match(ctx, bundleTx) {
				return fmt.Errorf("multiple bid transactions in lane %s", l.Name())
			}

			txBz, err := l.TxEncoder()(bundleTx)
			if err != nil {
				return fmt.Errorf("failed to encode bundled tx in lane %s: %w", l.Name(), err)
			}

			if !bytes.Equal(txBz, bidInfo.Transactions[i]) {
				return fmt.Errorf("invalid order of transactions in lane %s", l.Name())
			}
		}

		// Ensure that there are no more bid transactions in the block proposal.
		for _, tx := range txs[len(bidInfo.Transactions)+1:] {
			if l.Match(ctx, tx) {
				return fmt.Errorf("multiple bid transactions in lane %s", l.Name())
			}
		}

		return nil
	}
}

// VerifyTx will verify that the bid transaction and all of its bundled
// transactions are valid. It will return an error if any of the transactions
// are invalid.
func (l *TOBLane) VerifyTx(ctx sdk.Context, bidTx sdk.Tx, bidInfo *types.BidInfo) (err error) {
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
