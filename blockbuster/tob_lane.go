package blockbuster

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/cometbft/cometbft/libs/log"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkmempool "github.com/cosmos/cosmos-sdk/types/mempool"
	"github.com/skip-mev/pob/mempool"
)

const (
	// LaneNameTOB defines the name of the top-of-block auction lane.
	LaneNameTOB = "tob"
)

type (
	// AuctionBidInfo defines the information about a bid to the auction house.
	AuctionBidInfo struct {
		Bidder       sdk.AccAddress
		Bid          sdk.Coin
		Transactions [][]byte
		Timeout      uint64
		Signers      []map[string]struct{}
	}

	// AuctionFactory defines the interface for processing auction transactions. It is
	// a wrapper around all of the functionality that each application chain must implement
	// in order for auction processing to work.
	AuctionFactory interface {
		// WrapBundleTransaction defines a function that wraps a bundle transaction into a sdk.Tx. Since
		// this is a potentially expensive operation, we allow each application chain to define how
		// they want to wrap the transaction such that it is only called when necessary (i.e. when the
		// transaction is being considered in the proposal handlers).
		WrapBundleTransaction(tx []byte) (sdk.Tx, error)

		// GetAuctionBidInfo defines a function that returns the bid info from an auction transaction.
		GetAuctionBidInfo(tx sdk.Tx) (*AuctionBidInfo, error)
	}
)

var _ Lane = (*TOBLane)(nil)

type TOBLane struct {
	logger      log.Logger
	index       sdkmempool.Mempool
	af          AuctionFactory
	txEncoder   sdk.TxEncoder
	txDecoder   sdk.TxDecoder
	anteHandler sdk.AnteHandler

	// txIndex is a map of all transactions in the mempool. It is used
	// to quickly check if a transaction is already in the mempool.
	txIndex map[string]struct{}
}

func NewTOBLane(logger log.Logger, txDecoder sdk.TxDecoder, txEncoder sdk.TxEncoder, maxTx int, af AuctionFactory, anteHandler sdk.AnteHandler) *TOBLane {
	return &TOBLane{
		logger: logger,
		index: mempool.NewPriorityMempool(
			mempool.PriorityNonceMempoolConfig[int64]{
				TxPriority: mempool.NewDefaultTxPriority(),
				MaxTx:      maxTx,
			},
		),
		af:          af,
		txEncoder:   txEncoder,
		txDecoder:   txDecoder,
		anteHandler: anteHandler,
		txIndex:     make(map[string]struct{}),
	}
}

func (l *TOBLane) Name() string {
	return LaneNameTOB
}

func (l *TOBLane) Match(tx sdk.Tx) bool {
	bidInfo, err := l.af.GetAuctionBidInfo(tx)
	return bidInfo != nil && err == nil
}

func (l *TOBLane) Contains(tx sdk.Tx) (bool, error) {
	txHashStr, err := l.getTxHashStr(tx)
	if err != nil {
		return false, fmt.Errorf("failed to get tx hash string: %w", err)
	}

	_, ok := l.txIndex[txHashStr]
	return ok, nil
}

func (l *TOBLane) VerifyTx(ctx sdk.Context, bidTx sdk.Tx) error {
	bidInfo, err := l.af.GetAuctionBidInfo(bidTx)
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
		bundledTx, err := l.af.WrapBundleTransaction(tx)
		if err != nil {
			return fmt.Errorf("invalid bid tx; failed to decode bundled tx: %w", err)
		}

		// bid txs cannot be included in bundled txs
		bidInfo, _ := l.af.GetAuctionBidInfo(bundledTx)
		if bidInfo != nil {
			return fmt.Errorf("invalid bid tx; bundled tx cannot be a bid tx")
		}

		if ctx, err = l.verifyTx(ctx, bundledTx); err != nil {
			return fmt.Errorf("invalid bid tx; failed to execute bundled transaction: %w", err)
		}
	}

	return nil
}

// PrepareLane which builds a portion of the block. Inputs a cache of transactions
// that have already been included by a previous lane.
func (l *TOBLane) PrepareLane(ctx sdk.Context, maxTxBytes int64, selectedTxs map[string][]byte) ([][]byte, error) {
	var tmpSelectedTxs [][]byte

	bidTxIterator := l.index.Select(ctx, nil)
	txsToRemove := make(map[sdk.Tx]struct{}, 0)

	// Attempt to select the highest bid transaction that is valid and whose
	// bundled transactions are valid.
selectBidTxLoop:
	for ; bidTxIterator != nil; bidTxIterator = bidTxIterator.Next() {
		cacheCtx, write := ctx.CacheContext()
		tmpBidTx := bidTxIterator.Tx()

		// if the transaction is already in the (partial) block proposal, we skip it
		txHash, err := l.getTxHashStr(tmpBidTx)
		if err != nil {
			return nil, fmt.Errorf("failed to get bid tx hash: %w", err)
		}
		if _, ok := selectedTxs[txHash]; ok {
			continue selectBidTxLoop
		}

		bidTxBz, err := l.txEncoder(tmpBidTx)
		if err != nil {
			txsToRemove[tmpBidTx] = struct{}{}
			continue selectBidTxLoop
		}

		bidTxSize := int64(len(bidTxBz))
		if bidTxSize <= maxTxBytes {
			if err := l.VerifyTx(cacheCtx, tmpBidTx); err != nil {
				// Some transactions in the bundle may be malformed or invalid, so we
				// remove the bid transaction and try the next top bid.
				txsToRemove[tmpBidTx] = struct{}{}
				continue selectBidTxLoop
			}

			bidInfo, err := l.af.GetAuctionBidInfo(tmpBidTx)
			if bidInfo == nil || err != nil {
				// Some transactions in the bundle may be malformed or invalid, so we
				// remove the bid transaction and try the next top bid.
				txsToRemove[tmpBidTx] = struct{}{}
				continue selectBidTxLoop
			}

			// store the bytes of each ref tx as sdk.Tx bytes in order to build a valid proposal
			bundledTxBz := make([][]byte, len(bidInfo.Transactions))
			for index, rawRefTx := range bidInfo.Transactions {
				bundleTxBz := make([]byte, len(rawRefTx))
				copy(bundleTxBz, rawRefTx)
				bundledTxBz[index] = rawRefTx
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
		l.logger.Info(
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

// ProcessLane which verifies the lane's portion of a proposed block.
func (l *TOBLane) ProcessLane(ctx sdk.Context, proposalTxs [][]byte) error {
	for index, txBz := range proposalTxs {
		tx, err := l.txDecoder(txBz)
		if err != nil {
			return err
		}

		// skip transaction if it does not match this lane
		if !l.Match(tx) {
			continue
		}

		_, err = l.processProposalVerifyTx(ctx, txBz)
		if err != nil {
			return err
		}

		bidInfo, err := l.af.GetAuctionBidInfo(tx)
		if err != nil {
			return err
		}

		// If the transaction is an auction bid, then we need to ensure that it is
		// the first transaction in the block proposal and that the order of
		// transactions in the block proposal follows the order of transactions in
		// the bid.
		if bidInfo != nil {
			if index != 0 {
				return errors.New("auction bid must be the first transaction in the block proposal")
			}

			bundledTransactions := bidInfo.Transactions
			if len(proposalTxs) < len(bundledTransactions)+1 {
				return errors.New("block proposal does not contain enough transactions to match the bundled transactions in the auction bid")
			}

			for i, refTxRaw := range bundledTransactions {
				// Wrap and then encode the bundled transaction to ensure that the underlying
				// reference transaction can be processed as an sdk.Tx.
				wrappedTx, err := l.af.WrapBundleTransaction(refTxRaw)
				if err != nil {
					return err
				}

				refTxBz, err := l.txEncoder(wrappedTx)
				if err != nil {
					return err
				}

				if !bytes.Equal(refTxBz, proposalTxs[i+1]) {
					return errors.New("block proposal does not match the bundled transactions in the auction bid")
				}
			}
		}
	}

	return nil
}

func (l *TOBLane) Insert(goCtx context.Context, tx sdk.Tx) error {
	txHashStr, err := l.getTxHashStr(tx)
	if err != nil {
		return err
	}

	if err := l.index.Insert(goCtx, tx); err != nil {
		return fmt.Errorf("failed to insert tx into auction index: %w", err)
	}

	l.txIndex[txHashStr] = struct{}{}
	return nil
}

func (l *TOBLane) Select(goCtx context.Context, txs [][]byte) sdkmempool.Iterator {
	return l.index.Select(goCtx, txs)
}

func (l *TOBLane) CountTx() int {
	return l.index.CountTx()
}

func (l *TOBLane) Remove(tx sdk.Tx) error {
	txHashStr, err := l.getTxHashStr(tx)
	if err != nil {
		return fmt.Errorf("failed to get tx hash string: %w", err)
	}

	if err := l.index.Remove(tx); err != nil && !errors.Is(err, sdkmempool.ErrTxNotFound) {
		return fmt.Errorf("failed to remove invalid transaction from the mempool: %w", err)
	}

	delete(l.txIndex, txHashStr)
	return nil
}

func (l *TOBLane) processProposalVerifyTx(ctx sdk.Context, txBz []byte) (sdk.Tx, error) {
	tx, err := l.txDecoder(txBz)
	if err != nil {
		return nil, err
	}

	if _, err := l.verifyTx(ctx, tx); err != nil {
		return nil, err
	}

	return tx, nil
}

func (l *TOBLane) verifyTx(ctx sdk.Context, tx sdk.Tx) (sdk.Context, error) {
	if l.anteHandler != nil {
		newCtx, err := l.anteHandler(ctx, tx, false)
		return newCtx, err
	}

	return ctx, nil
}

// getTxHashStr returns the transaction hash string for a given transaction.
func (l *TOBLane) getTxHashStr(tx sdk.Tx) (string, error) {
	txBz, err := l.txEncoder(tx)
	if err != nil {
		return "", fmt.Errorf("failed to encode transaction: %w", err)
	}

	txHash := sha256.Sum256(txBz)
	txHashStr := hex.EncodeToString(txHash[:])

	return txHashStr, nil
}
