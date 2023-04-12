package abci

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"

	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cometbft/cometbft/libs/log"
	"github.com/cosmos/cosmos-sdk/baseapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkmempool "github.com/cosmos/cosmos-sdk/types/mempool"
	"github.com/skip-mev/pob/mempool"
)

type ProposalHandler struct {
	mempool    *mempool.AuctionMempool
	logger     log.Logger
	txVerifier baseapp.ProposalTxVerifier
	txEncoder  sdk.TxEncoder
	txDecoder  sdk.TxDecoder
}

func NewProposalHandler(
	mp *mempool.AuctionMempool,
	logger log.Logger,
	txVerifier baseapp.ProposalTxVerifier,
	txEncoder sdk.TxEncoder,
	txDecoder sdk.TxDecoder,
) *ProposalHandler {
	return &ProposalHandler{
		mempool:    mp,
		logger:     logger,
		txVerifier: txVerifier,
		txEncoder:  txEncoder,
		txDecoder:  txDecoder,
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
			tmpBidTx := bidTxIterator.Tx()

			bidTxBz, err := h.txVerifier.PrepareProposalVerifyTx(tmpBidTx)
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

					if _, err := h.txVerifier.PrepareProposalVerifyTx(refTx); err != nil {
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

			txBz, err = h.txVerifier.PrepareProposalVerifyTx(memTx)
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
			tx, err := h.txVerifier.ProcessProposalVerifyTx(txBz)
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

func (h *ProposalHandler) RemoveTx(tx sdk.Tx) {
	if err := h.mempool.RemoveWithoutRefTx(tx); err != nil && !errors.Is(err, sdkmempool.ErrTxNotFound) {
		panic(fmt.Errorf("failed to remove invalid transaction from the mempool: %w", err))
	}
}
