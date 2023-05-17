package v2

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"

	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cometbft/cometbft/libs/log"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkmempool "github.com/cosmos/cosmos-sdk/types/mempool"
	pobabci "github.com/skip-mev/pob/abci"
	mempool "github.com/skip-mev/pob/mempool"
)

const (
	// NumInjectedTxs is the minimum number of transactions that were injected into
	// the proposal but are not actual transactions. In this case, the auction
	// info is injected into the proposal but should be ignored by the application.ß
	NumInjectedTxs = 1

	// AuctionInfoIndex is the index of the auction info in the proposal.
	AuctionInfoIndex = 0
)

type (
	// ProposalMempool contains the methods required by the ProposalHandler
	// to interact with the local mempool.
	ProposalMempool interface {
		sdkmempool.Mempool

		// The AuctionFactory interface is utilized to retrieve, validate, and wrap bid
		// information into the block proposal.
		mempool.AuctionFactory

		// AuctionBidSelect returns an iterator that iterates over the top bid
		// transactions in the mempool.
		AuctionBidSelect(ctx context.Context) sdkmempool.Iterator
	}

	// ProposalHandler contains the functionality and handlers required to\
	// process, validate and build blocks.
	ProposalHandler struct {
		mempool     ProposalMempool
		logger      log.Logger
		anteHandler sdk.AnteHandler
		txEncoder   sdk.TxEncoder
		txDecoder   sdk.TxDecoder
	}
)

// NewProposalHandler returns a ProposalHandler that contains the functionality and handlers
// required to process, validate and build blocks.
func NewProposalHandler(
	mp ProposalMempool,
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
		// Proposal includes all of the transactions that will be included in the
		// block along with the vote extensions from the previous block included at
		// the beginning of the proposal. Vote extensions must be included in the
		// first slot of the proposal because they are inaccessible in ProcessProposal.
		proposal := make([][]byte, 0)

		// Build the top of block portion of the proposal given the vote extensions
		// from the previous block.
		topOfBlock := h.BuildTOB(ctx, req.LocalLastCommit, req.MaxTxBytes)

		// If information is unable to be marshaled, we return an empty proposal. This will
		// cause another proposal to be generated after it is rejected in ProcessProposal.
		lastCommitInfo, err := req.LocalLastCommit.Marshal()
		if err != nil {
			return abci.ResponsePrepareProposal{Txs: proposal}
		}

		auctionInfo := pobabci.AuctionInfo{
			ExtendedCommitInfo: lastCommitInfo,
			MaxTxBytes:         req.MaxTxBytes,
			NumTxs:             uint64(len(topOfBlock.Txs)),
		}

		// Add the auction info and top of block transactions into the proposal.
		auctionInfoBz, err := auctionInfo.Marshal()
		if err != nil {
			return abci.ResponsePrepareProposal{Txs: proposal}
		}

		proposal = append(proposal, auctionInfoBz)
		proposal = append(proposal, topOfBlock.Txs...)

		// Select remaining transactions for the block proposal until we've reached
		// size capacity.
		totalTxBytes := topOfBlock.Size
		txsToRemove := make(map[sdk.Tx]struct{}, 0)
		for iterator := h.mempool.Select(ctx, nil); iterator != nil; iterator = iterator.Next() {
			memTx := iterator.Tx()

			// If the transaction has already been seen in the top of block, skip it.
			txBz, err := h.txEncoder(memTx)
			if err != nil {
				txsToRemove[memTx] = struct{}{}
				continue
			}

			hashBz := sha256.Sum256(txBz)
			hash := hex.EncodeToString(hashBz[:])
			if _, ok := topOfBlock.Cache[hash]; ok {
				continue
			}

			// Verify that the transaction is valid.
			txBz, err = h.PrepareProposalVerifyTx(ctx, memTx)
			if err != nil {
				txsToRemove[memTx] = struct{}{}
				continue
			}

			txSize := int64(len(txBz))
			if totalTxBytes += txSize; totalTxBytes <= req.MaxTxBytes {
				proposal = append(proposal, txBz)
			} else {
				// We've reached capacity per req.MaxTxBytes so we cannot select any
				// more transactions.
				break
			}
		}

		// Remove all invalid transactions from the mempool.
		for tx := range txsToRemove {
			h.RemoveTx(tx)
		}

		return abci.ResponsePrepareProposal{Txs: proposal}
	}
}

// ProcessProposalHandler returns the ProcessProposal ABCI handler that performs
// block proposal verification.
func (h *ProposalHandler) ProcessProposalHandler() sdk.ProcessProposalHandler {
	return func(ctx sdk.Context, req abci.RequestProcessProposal) abci.ResponseProcessProposal {
		proposal := req.Txs

		// Verify that the same top of block transactions can be built from the vote
		// extensions included in the proposal.
		auctionInfo, err := h.VerifyTOB(ctx, proposal)
		if err != nil {
			return abci.ResponseProcessProposal{Status: abci.ResponseProcessProposal_REJECT}
		}

		// Track the transactions that need to be removed from the mempool.
		txsToRemove := make(map[sdk.Tx]struct{}, 0)
		invalidProposal := false

		// Verify that the remaining transactions in the proposal are valid.
		for _, txBz := range proposal[auctionInfo.NumTxs+NumInjectedTxs:] {
			tx, err := h.ProcessProposalVerifyTx(ctx, txBz)
			if tx == nil || err != nil {
				invalidProposal = true
				if tx != nil {
					txsToRemove[tx] = struct{}{}
				}

				continue
			}

			// The only auction transactions that should be included in the block proposal
			// must be at the top of the block.
			if bidInfo, err := h.mempool.GetAuctionBidInfo(tx); err != nil || bidInfo != nil {
				invalidProposal = true
			}
		}
		// Remove all invalid transactions from the mempool.
		for tx := range txsToRemove {
			h.RemoveTx(tx)
		}

		if invalidProposal {
			return abci.ResponseProcessProposal{Status: abci.ResponseProcessProposal_REJECT}
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

// RemoveTx removes a transaction from the application-side mempool.
func (h *ProposalHandler) RemoveTx(tx sdk.Tx) {
	if err := h.mempool.Remove(tx); err != nil && !errors.Is(err, sdkmempool.ErrTxNotFound) {
		panic(fmt.Errorf("failed to remove invalid transaction from the mempool: %w", err))
	}
}

// VerifyTx verifies a transaction against the application's state.
func (h *ProposalHandler) verifyTx(ctx sdk.Context, tx sdk.Tx) error {
	if h.anteHandler != nil {
		_, err := h.anteHandler(ctx, tx, false)
		return err
	}

	return nil
}
