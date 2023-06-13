package auction

import (
	"bytes"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/skip-mev/pob/blockbuster"
	"github.com/skip-mev/pob/blockbuster/utils"
)

// PrepareLane will attempt to select the highest bid transaction that is valid
// and whose bundled transactions are valid and include them in the proposal. It
// will return an empty partial proposal if no valid bids are found.
func (l *TOBLane) PrepareLane(
	ctx sdk.Context,
	proposal blockbuster.BlockProposal,
	maxTxBytes int64,
	next blockbuster.PrepareLanesHandler,
) (blockbuster.BlockProposal, error) {
	// Define all of the info we need to select transactions for the partial proposal.
	var (
		txs         [][]byte
		txsToRemove = make(map[sdk.Tx]struct{}, 0)
	)

	// Attempt to select the highest bid transaction that is valid and whose
	// bundled transactions are valid.
	bidTxIterator := l.Select(ctx, nil)
selectBidTxLoop:
	for ; bidTxIterator != nil; bidTxIterator = bidTxIterator.Next() {
		cacheCtx, write := ctx.CacheContext()
		tmpBidTx := bidTxIterator.Tx()

		bidTxBz, hash, err := utils.GetTxHashStr(l.Cfg.TxEncoder, tmpBidTx)
		if err != nil {
			txsToRemove[tmpBidTx] = struct{}{}
			continue selectBidTxLoop
		}

		// if the transaction is already in the (partial) block proposal, we skip it.
		if proposal.Contains(bidTxBz) {
			continue selectBidTxLoop
		}

		bidTxSize := int64(len(bidTxBz))
		if bidTxSize <= maxTxBytes {
			// Verify the bid transaction and all of its bundled transactions.
			if err := l.VerifyTx(cacheCtx, tmpBidTx); err != nil {
				l.Logger().Info(
					"failed to verify auction bid tx",
					"tx_hash", hash,
					"err", err,
				)
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

				sdkTxBz, _, err := utils.GetTxHashStr(l.Cfg.TxEncoder, sdkTx)
				if err != nil {
					txsToRemove[tmpBidTx] = struct{}{}
					continue selectBidTxLoop
				}

				// if the transaction is already in the (partial) block proposal, we skip it.
				if proposal.Contains(sdkTxBz) {
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

		l.Cfg.Logger.Info(
			"failed to select auction bid tx for lane; tx size is too large",
			"tx_size", bidTxSize,
			"max_size", maxTxBytes,
		)
	}

	// Remove all transactions that were invalid during the creation of the partial proposal.
	if err := utils.RemoveTxsFromLane(txsToRemove, l.Mempool); err != nil {
		return proposal, err
	}

	// Update the proposal with the selected transactions. This will only return an error
	// if the invarient checks are not passed. In the case when this errors, the original proposal
	// will be returned (without the selected transactions from this lane).
	if err := proposal.UpdateProposal(l, txs); err != nil {
		return proposal, err
	}

	return next(ctx, proposal)
}

// ProcessLane will ensure that block proposals that include transactions from
// the top-of-block auction lane are valid.
func (l *TOBLane) ProcessLane(ctx sdk.Context, txs []sdk.Tx, next blockbuster.ProcessLanesHandler) (sdk.Context, error) {
	bidTx := txs[0]

	if !l.Match(bidTx) {
		return next(ctx, txs)
	}

	bidInfo, err := l.GetAuctionBidInfo(bidTx)
	if err != nil {
		return ctx, fmt.Errorf("failed to get bid info for lane %s: %w", l.Name(), err)
	}

	if err := l.VerifyTx(ctx, bidTx); err != nil {
		return ctx, fmt.Errorf("invalid bid tx: %w", err)
	}

	return next(ctx, txs[len(bidInfo.Transactions)+1:])
}

// ProcessLaneBasic ensures that if a bid transaction is present in a proposal,
//   - it is the first transaction in the partial proposal
//   - all of the bundled transactions are included after the bid transaction in the order
//     they were included in the bid transaction.
//   - there are no other bid transactions in the proposal
func (l *TOBLane) ProcessLaneBasic(txs []sdk.Tx) error {
	bidTx := txs[0]

	// If there is a bid transaction, it must be the first transaction in the block proposal.
	if !l.Match(bidTx) {
		for _, tx := range txs[1:] {
			if l.Match(tx) {
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
		return fmt.Errorf("invalid number of transactions in lane %s; expected at least %d, got %d", l.Name(), len(bidInfo.Transactions)+1, len(txs))
	}

	// Ensure that the order of transactions in the bundle is preserved.
	for i, bundleTx := range txs[1 : len(bidInfo.Transactions)+1] {
		if l.Match(bundleTx) {
			return fmt.Errorf("multiple bid transactions in lane %s", l.Name())
		}

		txBz, err := l.Cfg.TxEncoder(bundleTx)
		if err != nil {
			return fmt.Errorf("failed to encode bundled tx in lane %s: %w", l.Name(), err)
		}

		if !bytes.Equal(txBz, bidInfo.Transactions[i]) {
			return fmt.Errorf("invalid order of transactions in lane %s", l.Name())
		}
	}

	// Ensure that there are no more bid transactions in the block proposal.
	for _, tx := range txs[len(bidInfo.Transactions)+1:] {
		if l.Match(tx) {
			return fmt.Errorf("multiple bid transactions in lane %s", l.Name())
		}
	}

	return nil
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
	if l.Cfg.AnteHandler != nil {
		newCtx, err := l.Cfg.AnteHandler(ctx, tx, false)
		return newCtx, err
	}

	return ctx, nil
}
