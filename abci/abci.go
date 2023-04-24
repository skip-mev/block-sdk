package abci

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"

	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cometbft/cometbft/libs/log"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkmempool "github.com/cosmos/cosmos-sdk/types/mempool"
)

type (
	Mempool interface {
		sdkmempool.Mempool
		AuctionBidSelect(ctx context.Context) sdkmempool.Iterator
		GetBundledTransactions(tx sdk.Tx) ([][]byte, error)
		WrapBundleTransaction(tx []byte) (sdk.Tx, error)
		IsAuctionTx(tx sdk.Tx) (bool, error)
		RemoveWithoutRefTx(tx sdk.Tx) error
	}

	ProposalHandler struct {
		mempool     Mempool
		logger      log.Logger
		anteHandler sdk.AnteHandler
		txEncoder   sdk.TxEncoder
		txDecoder   sdk.TxDecoder
	}
)

func NewProposalHandler(
	mp Mempool,
	logger log.Logger,
	anteHandler sdk.AnteHandler,
	txEncoder sdk.TxEncoder,
	txDecoder sdk.TxDecoder,
) *ProposalHandler {
	return &ProposalHandler{
		mempool:     mp,
		logger:      logger,
		anteHandler: anteHandler,
		txEncoder:   txEncoder,
		txDecoder:   txDecoder,
	}
}

// PrepareProposalHandler returns the PrepareProposal ABCI handler that performs
// top-of-block auctioning and general block proposal construction.
func (h *ProposalHandler) PrepareProposalHandler() sdk.PrepareProposalHandler {
	return func(ctx sdk.Context, req abci.RequestPrepareProposal) abci.ResponsePrepareProposal {
		var (
			selectedTxs  [][]byte
			totalTxBytes int64
		)

		bidTxIterator := h.mempool.AuctionBidSelect(ctx)
		txsToRemove := make(map[sdk.Tx]struct{}, 0)
		seenTxs := make(map[string]struct{}, 0)

		// Attempt to select the highest bid transaction that is valid and whose
		// bundled transactions are valid.
	selectBidTxLoop:
		for ; bidTxIterator != nil; bidTxIterator = bidTxIterator.Next() {
			cacheCtx, write := ctx.CacheContext()
			tmpBidTx := bidTxIterator.Tx()

			bidTxBz, err := h.PrepareProposalVerifyTx(cacheCtx, tmpBidTx)
			if err != nil {
				txsToRemove[tmpBidTx] = struct{}{}
				continue selectBidTxLoop
			}

			bidTxSize := int64(len(bidTxBz))
			if bidTxSize <= req.MaxTxBytes {
				bundledTransactions, err := h.mempool.GetBundledTransactions(tmpBidTx)
				if err != nil {
					// Some transactions in the bundle may be malformatted or invalid, so
					// we remove the bid transaction and try the next top bid.
					txsToRemove[tmpBidTx] = struct{}{}
					continue selectBidTxLoop
				}

				// store the bytes of each ref tx as sdk.Tx bytes in order to build a valid proposal
				sdkTxBytes := make([][]byte, len(bundledTransactions))

				// Ensure that the bundled transactions are valid
				for index, rawRefTx := range bundledTransactions {
					refTx, err := h.mempool.WrapBundleTransaction(rawRefTx)
					if err != nil {
						// Malformed bundled transaction, so we remove the bid transaction
						// and try the next top bid.
						txsToRemove[tmpBidTx] = struct{}{}
						continue selectBidTxLoop
					}

					txBz, err := h.PrepareProposalVerifyTx(cacheCtx, refTx)
					if err != nil {
						// Invalid bundled transaction, so we remove the bid transaction
						// and try the next top bid.
						txsToRemove[tmpBidTx] = struct{}{}
						continue selectBidTxLoop
					}

					sdkTxBytes[index] = txBz
				}

				// At this point, both the bid transaction itself and all the bundled
				// transactions are valid. So we select the bid transaction along with
				// all the bundled transactions. We also mark these transactions as seen and
				// update the total size selected thus far.
				totalTxBytes += bidTxSize
				selectedTxs = append(selectedTxs, bidTxBz)
				selectedTxs = append(selectedTxs, sdkTxBytes...)

				for _, refTxRaw := range sdkTxBytes {
					hash := sha256.Sum256(refTxRaw)
					txHash := hex.EncodeToString(hash[:])
					seenTxs[txHash] = struct{}{}
				}

				// Write the cache context to the original context when we know we have a
				// valid top of block bundle.
				write()

				break selectBidTxLoop
			}

			txsToRemove[tmpBidTx] = struct{}{}
			h.logger.Info(
				"failed to select auction bid tx; tx size is too large",
				"tx_size", bidTxSize,
				"max_size", req.MaxTxBytes,
			)
		}

		// Remove all invalid transactions from the mempool.
		for tx := range txsToRemove {
			h.RemoveTx(tx)
		}

		iterator := h.mempool.Select(ctx, nil)
		txsToRemove = map[sdk.Tx]struct{}{}

		// Select remaining transactions for the block proposal until we've reached
		// size capacity.
	selectTxLoop:
		for ; iterator != nil; iterator = iterator.Next() {
			memTx := iterator.Tx()

			// If the transaction is already included in the proposal, then we skip it.
			txBz, err := h.txEncoder(memTx)
			if err != nil {
				txsToRemove[memTx] = struct{}{}
				continue selectTxLoop
			}

			hash := sha256.Sum256(txBz)
			txHash := hex.EncodeToString(hash[:])
			if _, ok := seenTxs[txHash]; ok {
				continue selectTxLoop
			}

			txBz, err = h.PrepareProposalVerifyTx(ctx, memTx)
			if err != nil {
				txsToRemove[memTx] = struct{}{}
				continue selectTxLoop
			}

			txSize := int64(len(txBz))
			if totalTxBytes += txSize; totalTxBytes <= req.MaxTxBytes {
				selectedTxs = append(selectedTxs, txBz)
			} else {
				// We've reached capacity per req.MaxTxBytes so we cannot select any
				// more transactions.
				break selectTxLoop
			}
		}

		// Remove all invalid transactions from the mempool.
		for tx := range txsToRemove {
			h.RemoveTx(tx)
		}

		return abci.ResponsePrepareProposal{Txs: selectedTxs}
	}
}

// ProcessProposalHandler returns the ProcessProposal ABCI handler that performs
// block proposal verification.
func (h *ProposalHandler) ProcessProposalHandler() sdk.ProcessProposalHandler {
	return func(ctx sdk.Context, req abci.RequestProcessProposal) abci.ResponseProcessProposal {
		for index, txBz := range req.Txs {
			tx, err := h.ProcessProposalVerifyTx(ctx, txBz)
			if err != nil {
				return abci.ResponseProcessProposal{Status: abci.ResponseProcessProposal_REJECT}
			}

			isAuctionTx, err := h.mempool.IsAuctionTx(tx)
			if err != nil {
				return abci.ResponseProcessProposal{Status: abci.ResponseProcessProposal_REJECT}
			}

			if isAuctionTx {
				// Only the first transaction can be an auction bid tx
				if index != 0 {
					return abci.ResponseProcessProposal{Status: abci.ResponseProcessProposal_REJECT}
				}

				bundledTransactions, err := h.mempool.GetBundledTransactions(tx)
				if err != nil {
					return abci.ResponseProcessProposal{Status: abci.ResponseProcessProposal_REJECT}
				}

				// The order of transactions in the block proposal must follow the order of transactions in the bid.
				if len(req.Txs) < len(bundledTransactions)+1 {
					return abci.ResponseProcessProposal{Status: abci.ResponseProcessProposal_REJECT}
				}

				for i, refTxRaw := range bundledTransactions {
					// Wrap and then encode the bundled transaction to ensure that the underlying
					// reference transaction can be processed as an sdk.Tx.
					wrappedTx, err := h.mempool.WrapBundleTransaction(refTxRaw)
					if err != nil {
						return abci.ResponseProcessProposal{Status: abci.ResponseProcessProposal_REJECT}
					}

					refTxBz, err := h.txEncoder(wrappedTx)
					if err != nil {
						return abci.ResponseProcessProposal{Status: abci.ResponseProcessProposal_REJECT}
					}

					if !bytes.Equal(refTxBz, req.Txs[i+1]) {
						return abci.ResponseProcessProposal{Status: abci.ResponseProcessProposal_REJECT}
					}
				}
			}

		}

		return abci.ResponseProcessProposal{Status: abci.ResponseProcessProposal_ACCEPT}
	}
}

// PrepareProposalVerifyTx encodes a transaction and verifies it.
func (h *ProposalHandler) PrepareProposalVerifyTx(ctx sdk.Context, tx sdk.Tx) ([]byte, error) {
	txBz, err := h.txEncoder(tx)
	if err != nil {
		return nil, err
	}

	return txBz, h.verifyTx(ctx, tx)
}

// ProcessProposalVerifyTx decodes a transaction and verifies it.
func (h *ProposalHandler) ProcessProposalVerifyTx(ctx sdk.Context, txBz []byte) (sdk.Tx, error) {
	tx, err := h.txDecoder(txBz)
	if err != nil {
		return nil, err
	}

	return tx, h.verifyTx(ctx, tx)
}

// VerifyTx verifies a transaction against the application's state.
func (h *ProposalHandler) verifyTx(ctx sdk.Context, tx sdk.Tx) error {
	if h.anteHandler != nil {
		_, err := h.anteHandler(ctx, tx, false)
		return err
	}

	return nil
}

func (h *ProposalHandler) RemoveTx(tx sdk.Tx) {
	if err := h.mempool.RemoveWithoutRefTx(tx); err != nil && !errors.Is(err, sdkmempool.ErrTxNotFound) {
		panic(fmt.Errorf("failed to remove invalid transaction from the mempool: %w", err))
	}
}
