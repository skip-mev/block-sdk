package abci

import (
	"errors"
	"fmt"

	cometabci "github.com/cometbft/cometbft/abci/types"
	"github.com/cometbft/cometbft/libs/log"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkmempool "github.com/cosmos/cosmos-sdk/types/mempool"
	"github.com/skip-mev/pob/blockbuster"
	"github.com/skip-mev/pob/blockbuster/abci"
	"github.com/skip-mev/pob/blockbuster/lanes/auction"
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
	// TOBLaneProposal is the interface that defines all of the dependencies that
	// are required to interact with the top of block lane.
	TOBLaneProposal interface {
		sdkmempool.Mempool

		// Factory defines the API/functionality which is responsible for determining
		// if a transaction is a bid transaction and how to extract relevant
		// information from the transaction (bid, timeout, bidder, etc.).
		auction.Factory

		// VerifyTx is utilized to verify a bid transaction according to the preferences
		// of the top of block lane.
		VerifyTx(ctx sdk.Context, tx sdk.Tx) error

		// ProcessLaneBasic is utilized to verify the rest of the proposal according to
		// the preferences of the top of block lane. This is used to verify that no
		ProcessLaneBasic(txs [][]byte) error
	}

	// ProposalHandler contains the functionality and handlers required to\
	// process, validate and build blocks.
	ProposalHandler struct {
		prepareLanesHandler blockbuster.PrepareLanesHandler
		processLanesHandler blockbuster.ProcessLanesHandler
		tobLane             TOBLaneProposal
		logger              log.Logger
		txEncoder           sdk.TxEncoder
		txDecoder           sdk.TxDecoder
	}
)

// NewProposalHandler returns a ProposalHandler that contains the functionality and handlers
// required to process, validate and build blocks.
func NewProposalHandler(
	lanes []blockbuster.Lane,
	tobLane TOBLaneProposal,
	logger log.Logger,
	txEncoder sdk.TxEncoder,
	txDecoder sdk.TxDecoder,
) *ProposalHandler {
	return &ProposalHandler{
		prepareLanesHandler: abci.ChainPrepareLanes(lanes...),
		processLanesHandler: abci.ChainProcessLanes(lanes...),
		tobLane:             tobLane,
		logger:              logger,
		txEncoder:           txEncoder,
		txDecoder:           txDecoder,
	}
}

// PrepareProposalHandler returns the PrepareProposal ABCI handler that performs
// top-of-block auctioning and general block proposal construction.
func (h *ProposalHandler) PrepareProposalHandler() sdk.PrepareProposalHandler {
	return func(ctx sdk.Context, req cometabci.RequestPrepareProposal) cometabci.ResponsePrepareProposal {
		// Build the top of block portion of the proposal given the vote extensions
		// from the previous block.
		topOfBlock := h.BuildTOB(ctx, req.LocalLastCommit, req.MaxTxBytes)

		// If information is unable to be marshaled, we return an empty proposal. This will
		// cause another proposal to be generated after it is rejected in ProcessProposal.
		lastCommitInfo, err := req.LocalLastCommit.Marshal()
		if err != nil {
			h.logger.Error("failed to marshal last commit info", "err", err)
			return cometabci.ResponsePrepareProposal{Txs: nil}
		}

		auctionInfo := &AuctionInfo{
			ExtendedCommitInfo: lastCommitInfo,
			MaxTxBytes:         req.MaxTxBytes,
			NumTxs:             uint64(len(topOfBlock.Txs)),
		}

		// Add the auction info and top of block transactions into the proposal.
		auctionInfoBz, err := auctionInfo.Marshal()
		if err != nil {
			h.logger.Error("failed to marshal auction info", "err", err)
			return cometabci.ResponsePrepareProposal{Txs: nil}
		}

		topOfBlock.Txs = append([][]byte{auctionInfoBz}, topOfBlock.Txs...)

		// Prepare the proposal by selecting transactions from each lane according to
		// each lane's selection logic.
		proposal := h.prepareLanesHandler(ctx, topOfBlock)

		return cometabci.ResponsePrepareProposal{Txs: proposal.Txs}
	}
}

// ProcessProposalHandler returns the ProcessProposal ABCI handler that performs
// block proposal verification.
func (h *ProposalHandler) ProcessProposalHandler() sdk.ProcessProposalHandler {
	return func(ctx sdk.Context, req cometabci.RequestProcessProposal) cometabci.ResponseProcessProposal {
		proposal := req.Txs

		// Verify that the same top of block transactions can be built from the vote
		// extensions included in the proposal.
		auctionInfo, err := h.VerifyTOB(ctx, proposal)
		if err != nil {
			h.logger.Error("failed to verify top of block transactions", "err", err)
			return cometabci.ResponseProcessProposal{Status: cometabci.ResponseProcessProposal_REJECT}
		}

		// Do a basic check of the rest of the proposal to make sure no auction transactions
		// are included.
		if err := h.tobLane.ProcessLaneBasic(proposal[NumInjectedTxs:]); err != nil {
			h.logger.Error("failed to process proposal", "err", err)
			return cometabci.ResponseProcessProposal{Status: cometabci.ResponseProcessProposal_REJECT}
		}

		// Verify that the rest of the proposal is valid according to each lane's verification logic.
		if _, err = h.processLanesHandler(ctx, proposal[auctionInfo.NumTxs:]); err != nil {
			h.logger.Error("failed to process proposal", "err", err)
			return cometabci.ResponseProcessProposal{Status: cometabci.ResponseProcessProposal_REJECT}
		}

		return cometabci.ResponseProcessProposal{Status: cometabci.ResponseProcessProposal_ACCEPT}
	}
}

// RemoveTx removes a transaction from the application-side mempool.
func (h *ProposalHandler) RemoveTx(tx sdk.Tx) {
	if err := h.tobLane.Remove(tx); err != nil && !errors.Is(err, sdkmempool.ErrTxNotFound) {
		panic(fmt.Errorf("failed to remove invalid transaction from the mempool: %w", err))
	}
}
