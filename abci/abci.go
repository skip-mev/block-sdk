package abci

import (
	"fmt"

	"cosmossdk.io/log"
	abci "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/skip-mev/block-sdk/block"
	"github.com/skip-mev/block-sdk/block/proposals"
	"github.com/skip-mev/block-sdk/block/proposals/types"
)

const (
	// MetaDataIndex is the index of the proposal metadata in the proposal.
	MetaDataIndex = 0
)

type (
	// ProposalHandler is a wrapper around the ABCI++ PrepareProposal and ProcessProposal
	// handlers.
	ProposalHandler struct {
		logger              log.Logger
		txDecoder           sdk.TxDecoder
		txEncoder           sdk.TxEncoder
		prepareLanesHandler block.PrepareLanesHandler
		processLanesHandler block.ProcessLanesHandler
		mempool             block.Mempool
	}
)

// NewProposalHandler returns a new abci++ proposal handler. This proposal handler will
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
// according to each lane's selection logic. We select transactions in a greedy fashion. Note that
// each lane has an boundary on the number of bytes that can be included in the proposal. By default,
// the default lane will not have a boundary on the number of bytes that can be included in the proposal and
// will include all valid transactions in the proposal (up to MaxTxBytes).
func (h *ProposalHandler) PrepareProposalHandler() sdk.PrepareProposalHandler {
	return func(ctx sdk.Context, req *abci.RequestPrepareProposal) (resp *abci.ResponsePrepareProposal, err error) {
		// In the case where there is a panic, we recover here and return an empty proposal.
		defer func() {
			if rec := recover(); rec != nil {
				h.logger.Error("failed to prepare proposal", "err", err)

				resp = &abci.ResponsePrepareProposal{Txs: make([][]byte, 0)}
				err = fmt.Errorf("failed to prepare proposal: %v", rec)
			}
		}()

		h.logger.Info("mempool distribution before proposal creation", "distribution", h.mempool.GetTxDistribution())

		// Build an empty placeholder proposal with the maximum block size and gas limit.
		maxBlockSize, maxGasLimit := proposals.GetBlockLimits(ctx)
		emptyProposal := proposals.NewProposal(h.txEncoder, maxBlockSize, maxGasLimit)

		// Fill the proposal with transactions from each lane respecting the maximum block size and gas limit.
		finalProposal, err := h.prepareLanesHandler(ctx, emptyProposal)
		if err != nil {
			h.logger.Error("failed to prepare proposal", "err", err)
			return &abci.ResponsePrepareProposal{Txs: make([][]byte, 0)}, err
		}

		// Retrieve the proposal with metadata and transactions.
		proposal := finalProposal.GetProposal()

		h.logger.Info(
			"prepared proposal",
			"num_txs", len(proposal),
			"total_tx_bytes", finalProposal.BlockSize,
			"max_tx_bytes", maxBlockSize,
			"total_gas_limit", finalProposal.GasLimt,
			"max_gas_limit", maxGasLimit,
			"height", req.Height,
		)

		h.logger.Info("mempool distribution after proposal creation", "distribution", h.mempool.GetTxDistribution())

		return &abci.ResponsePrepareProposal{
			Txs: proposal,
		}, nil
	}
}

// ProcessProposalHandler processes the proposal by verifying all transactions in the proposal
// according to each lane's verification logic. We verify proposals in a greedy fashion.
// If a lane's portion of the proposal is invalid, we reject the proposal. After a lane's portion
// of the proposal is verified, we pass the remaining transactions to the next lane in the chain.
func (h *ProposalHandler) ProcessProposalHandler() sdk.ProcessProposalHandler {
	return func(ctx sdk.Context, req *abci.RequestProcessProposal) (resp *abci.ResponseProcessProposal, err error) {
		// In the case where any of the lanes panic, we recover here and return a reject status.
		defer func() {
			if rec := recover(); rec != nil {
				h.logger.Error("failed to process proposal", "recover_err", rec)

				resp = &abci.ResponseProcessProposal{Status: abci.ResponseProcessProposal_REJECT}
				err = fmt.Errorf("failed to process proposal: %v", rec)
			}
		}()

		partialProposals, err := h.ValidateBasic(ctx, req.Txs)
		if err != nil {
			h.logger.Error("failed to validate proposal", "err", err)
			return &abci.ResponseProcessProposal{Status: abci.ResponseProcessProposal_REJECT}, err
		}

		processLanesHandler := ChainProcessLanes(partialProposals, h.mempool.Registry())

		// Build an empty placeholder proposal with the maximum block size and gas limit.
		maxBlockSize, maxGasLimit := proposals.GetBlockLimits(ctx)
		emptyProposal := proposals.NewProposal(h.txEncoder, maxBlockSize, maxGasLimit)

		// Verify the proposal according to the verification logic from each lane.
		finalProposal, err := processLanesHandler(ctx, emptyProposal)
		if err != nil {
			h.logger.Error("failed to validate the proposal", "err", err)
			return &abci.ResponseProcessProposal{Status: abci.ResponseProcessProposal_REJECT}, err
		}

		// Retrieve the proposal with metadata and transactions.
		proposal := finalProposal.GetProposal()

		h.logger.Info(
			"processed proposal",
			"num_txs", len(proposal),
			"total_tx_bytes", finalProposal.BlockSize,
			"max_tx_bytes", maxBlockSize,
			"total_gas_limit", finalProposal.GasLimt,
			"max_gas_limit", maxGasLimit,
			"height", req.Height,
		)

		return &abci.ResponseProcessProposal{Status: abci.ResponseProcessProposal_ACCEPT}, nil
	}
}

// ValidateBasic validates the proposal against the basic invariants that are required
// for the proposal to be valid. This includes:
//  1. The proposal must contain the proposal metadata and must be valid.
//  2. The proposal must contain the correct number of transactions for each lane.
func (h *ProposalHandler) ValidateBasic(ctx sdk.Context, proposal [][]byte) ([][][]byte, error) {
	// If the proposal is empty, then the metadata was not included.
	if len(proposal) == 0 {
		return nil, fmt.Errorf("proposal does not contain proposal metadata")
	}

	metaDataBz, txs := proposal[MetaDataIndex], proposal[1:]

	// Retrieve the metadata from the proposal.
	var metaData types.ProposalInfo
	if err := metaData.Unmarshal(metaDataBz); err != nil {
		return nil, fmt.Errorf("failed to unmarshal proposal metadata: %w", err)
	}

	lanes := h.mempool.Registry()
	partialProposals := make([][][]byte, len(lanes))

	if metaData.Lanes == nil {
		if len(txs) > 0 {
			return nil, fmt.Errorf("proposal contains invalid number of transactions")
		}

		return partialProposals, nil
	}

	// Iterate through all of the lanes and match the corresponding transactions to the lane.
	for index, lane := range lanes {
		laneMetaData := metaData.Lanes[lane.Name()]
		if laneMetaData.NumTxs > uint64(len(txs)) {
			return nil, fmt.Errorf(
				"proposal metadata contains invalid number of transactions for lane %s; got %d, expected %d",
				lane.Name(),
				len(txs),
				laneMetaData.NumTxs,
			)
		}

		partialProposals[index] = txs[:laneMetaData.NumTxs]
		txs = txs[laneMetaData.NumTxs:]
	}

	// If there are any transactions remaining in the proposal, then the proposal is invalid.
	if len(txs) > 0 {
		return nil, fmt.Errorf("proposal contains invalid number of transactions")
	}

	return partialProposals, nil
}
