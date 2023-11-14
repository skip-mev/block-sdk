package abci

import (
	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cometbft/cometbft/libs/log"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/skip-mev/block-sdk/block"
	"github.com/skip-mev/block-sdk/block/proposals"
)

const (
	// ProposalInfoIndex is the index of the proposal metadata in the proposal.
	ProposalInfoIndex = 0
)

type (
	// ProposalHandler is a wrapper around the ABCI++ PrepareProposal and ProcessProposal
	// handlers.
	ProposalHandler struct {
		logger              log.Logger
		txDecoder           sdk.TxDecoder
		txEncoder           sdk.TxEncoder
		prepareLanesHandler block.PrepareLanesHandler
		mempool             block.Mempool
	}
)

// NewProposalHandler returns a new ABCI++ proposal handler. This proposal handler will
// iteratively call each of the lanes in the chain to prepare and process the proposal.
func NewProposalHandler(
	logger log.Logger,
	txDecoder sdk.TxDecoder,
	txEncoder sdk.TxEncoder,
	mempool block.Mempool,
) *ProposalHandler {
	return &ProposalHandler{
		logger:              logger,
		txDecoder:           txDecoder,
		txEncoder:           txEncoder,
		prepareLanesHandler: ChainPrepareLanes(mempool.Registry()),
		mempool:             mempool,
	}
}

// PrepareProposalHandler prepares the proposal by selecting transactions from each lane
// according to each lane's selection logic. We select transactions in the order in which the
// lanes are configured on the chain. Note that each lane has an boundary on the number of
// bytes/gas that can be included in the proposal. By default, the default lane will not have
// a boundary on the number of bytes that can be included in the proposal and will include all
// valid transactions in the proposal (up to MaxBlockSize, MaxGasLimit).
func (h *ProposalHandler) PrepareProposalHandler() sdk.PrepareProposalHandler {
	return func(ctx sdk.Context, req abci.RequestPrepareProposal) (resp abci.ResponsePrepareProposal) {
		if req.Height <= 1 {
			return abci.ResponsePrepareProposal{Txs: req.Txs}
		}

		// In the case where there is a panic, we recover here and return an empty proposal.
		defer func() {
			if rec := recover(); rec != nil {
				h.logger.Error("failed to prepare proposal", "err", rec)

				// TODO: Should we attempt to return a empty proposal here with empty proposal info?
				resp = abci.ResponsePrepareProposal{Txs: make([][]byte, 0)}
			}
		}()

		h.logger.Info(
			"mempool distribution before proposal creation",
			"distribution", h.mempool.GetTxDistribution(),
			"height", req.Height,
		)

		// Fill the proposal with transactions from each lane.
		finalProposal, err := h.prepareLanesHandler(ctx, proposals.NewProposalWithContext(h.logger, ctx, h.txEncoder))
		if err != nil {
			h.logger.Error("failed to prepare proposal", "err", err)
			return abci.ResponsePrepareProposal{Txs: make([][]byte, 0)}
		}

		// Retrieve the proposal with metadata and transactions.
		txs, err := finalProposal.GetProposalWithInfo()
		if err != nil {
			h.logger.Error("failed to get proposal with metadata", "err", err)
			return abci.ResponsePrepareProposal{Txs: make([][]byte, 0)}
		}

		h.logger.Info(
			"prepared proposal",
			"num_txs", len(txs),
			"total_tx_bytes", finalProposal.Info.BlockSize,
			"max_tx_bytes", finalProposal.Info.MaxBlockSize,
			"total_gas_limit", finalProposal.Info.GasLimit,
			"max_gas_limit", finalProposal.Info.MaxGasLimit,
			"height", req.Height,
		)

		h.logger.Info(
			"mempool distribution after proposal creation",
			"distribution", h.mempool.GetTxDistribution(),
			"height", req.Height,
		)

		return abci.ResponsePrepareProposal{
			Txs: txs,
		}
	}
}

// ProcessProposalHandler processes the proposal by verifying all transactions in the proposal
// according to each lane's verification logic. Proposals are verified similar to how they are
// constructed. After a proposal is processed, it should amount to the same proposal that was prepared.
// Each proposal will first be broken down by the lanes that prepared each partial proposal. Then, each
// lane will iteratively verify the transactions that it belong to it. If any lane fails to verify the
// transactions, then the proposal is rejected.
func (h *ProposalHandler) ProcessProposalHandler() sdk.ProcessProposalHandler {
	return func(ctx sdk.Context, req abci.RequestProcessProposal) (resp abci.ResponseProcessProposal) {
		if req.Height <= 1 {
			return abci.ResponseProcessProposal{Status: abci.ResponseProcessProposal_ACCEPT}
		}

		// In the case where any of the lanes panic, we recover here and return a reject status.
		defer func() {
			if rec := recover(); rec != nil {
				h.logger.Error("failed to process proposal", "recover_err", rec)

				resp = abci.ResponseProcessProposal{Status: abci.ResponseProcessProposal_REJECT}
			}
		}()

		// Extract all of the lanes and their corresponding transactions from the proposal.
		proposalInfo, partialProposals, err := h.ExtractLanes(req.Txs)
		if err != nil {
			h.logger.Error("failed to validate proposal", "err", err)
			return abci.ResponseProcessProposal{Status: abci.ResponseProcessProposal_REJECT}
		}

		// Build handler that will verify the partial proposals according to each lane's verification logic.
		processLanesHandler := ChainProcessLanes(partialProposals, h.mempool.Registry())
		finalProposal, err := processLanesHandler(ctx, proposals.NewProposalWithContext(h.logger, ctx, h.txEncoder))
		if err != nil {
			h.logger.Error("failed to validate the proposal", "err", err)
			return abci.ResponseProcessProposal{Status: abci.ResponseProcessProposal_REJECT}
		}

		// Ensure block size and gas limit are correct.
		if err := h.ValidateBlockLimits(finalProposal, proposalInfo); err != nil {
			h.logger.Error("failed to validate the proposal", "err", err)
			return abci.ResponseProcessProposal{Status: abci.ResponseProcessProposal_REJECT}
		}

		h.logger.Info(
			"processed proposal",
			"num_txs", len(req.Txs),
			"total_tx_bytes", finalProposal.Info.BlockSize,
			"max_tx_bytes", finalProposal.Info.MaxBlockSize,
			"total_gas_limit", finalProposal.Info.GasLimit,
			"max_gas_limit", finalProposal.Info.MaxGasLimit,
			"height", req.Height,
		)

		return abci.ResponseProcessProposal{Status: abci.ResponseProcessProposal_ACCEPT}
	}
}
