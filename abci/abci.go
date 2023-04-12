package abci

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"

	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cometbft/cometbft/libs/log"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkmempool "github.com/cosmos/cosmos-sdk/types/mempool"
	"github.com/skip-mev/pob/mempool"
)

type ProposalHandler struct {
	mempool     *mempool.AuctionMempool
	logger      log.Logger
	anteHandler sdk.AnteHandler
	txEncoder   sdk.TxEncoder
	txDecoder   sdk.TxDecoder
}

func NewProposalHandler(
	mp *mempool.AuctionMempool,
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
				bidMsg, err := mempool.GetMsgAuctionBidFromTx(tmpBidTx)
				if err != nil {
					// This should never happen, as CheckTx will ensure only valid bids
					// enter the mempool, but in case it does, we need to remove the
					// transaction from the mempool.
					txsToRemove[tmpBidTx] = struct{}{}
					continue selectBidTxLoop
				}

				for _, refTxRaw := range bidMsg.Transactions {
					refTx, err := h.txDecoder(refTxRaw)
					if err != nil {
						// Malformed bundled transaction, so we remove the bid transaction
						// and try the next top bid.
						txsToRemove[tmpBidTx] = struct{}{}
						continue selectBidTxLoop
					}

					if _, err := h.PrepareProposalVerifyTx(cacheCtx, refTx); err != nil {
						// Invalid bundled transaction, so we remove the bid transaction
						// and try the next top bid.
						txsToRemove[tmpBidTx] = struct{}{}
						continue selectBidTxLoop
					}
				}

				// At this point, both the bid transaction itself and all the bundled
				// transactions are valid. So we select the bid transaction along with
				// all the bundled transactions. We also mark these transactions as seen and
				// update the total size selected thus far.
				totalTxBytes += bidTxSize
				selectedTxs = append(selectedTxs, bidTxBz)
				selectedTxs = append(selectedTxs, bidMsg.Transactions...)

				for _, refTxRaw := range bidMsg.Transactions {
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

			msgAuctionBid, err := mempool.GetMsgAuctionBidFromTx(tx)
			if err != nil {
				return abci.ResponseProcessProposal{Status: abci.ResponseProcessProposal_REJECT}
			}

			if msgAuctionBid != nil {
				// Only the first transaction can be an auction bid tx
				if index != 0 {
					return abci.ResponseProcessProposal{Status: abci.ResponseProcessProposal_REJECT}
				}

				// The order of transactions in the block proposal must follow the order of transactions in the bid.
				if len(req.Txs) < len(msgAuctionBid.Transactions)+1 {
					return abci.ResponseProcessProposal{Status: abci.ResponseProcessProposal_REJECT}
				}

				for i, refTxRaw := range msgAuctionBid.Transactions {
					if !bytes.Equal(refTxRaw, req.Txs[i+1]) {
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
