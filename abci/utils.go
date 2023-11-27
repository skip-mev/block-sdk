package abci

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/skip-mev/block-sdk/block"
	"github.com/skip-mev/block-sdk/block/proposals"
	"github.com/skip-mev/block-sdk/lanes/terminator"
)

<<<<<<< HEAD
// ExtractLanes validates the proposal against the basic invariants that are required
// for the proposal to be valid. This includes:
//  1. The proposal must contain the proposal information and must be valid.
//  2. The proposal must contain the correct number of transactions for each lane.
func (h *ProposalHandler) ExtractLanes(proposal [][]byte) (types.ProposalInfo, [][][]byte, error) {
	// If the proposal is empty, then the metadata was not included.
	if len(proposal) == 0 {
		return types.ProposalInfo{}, nil, fmt.Errorf("proposal does not contain proposal metadata")
	}

	metaDataBz, txs := proposal[ProposalInfoIndex], proposal[ProposalInfoIndex+1:]

	// Retrieve the metadata from the proposal.
	var metaData types.ProposalInfo
	if err := metaData.Unmarshal(metaDataBz); err != nil {
		return types.ProposalInfo{}, nil, fmt.Errorf("failed to unmarshal proposal metadata: %w", err)
	}

	lanes := h.mempool.Registry()
	partialProposals := make([][][]byte, len(lanes))

	if metaData.TxsByLane == nil {
		if len(txs) > 0 {
			return types.ProposalInfo{}, nil, fmt.Errorf("proposal contains invalid number of transactions")
		}

		return types.ProposalInfo{}, partialProposals, nil
	}

	h.logger.Info(
		"received proposal with metadata",
		"max_block_size", metaData.MaxBlockSize,
		"max_gas_limit", metaData.MaxGasLimit,
		"gas_limit", metaData.GasLimit,
		"block_size", metaData.BlockSize,
		"lanes_with_txs", metaData.TxsByLane,
	)

	// Iterate through all of the lanes and match the corresponding transactions to the lane.
	for index, lane := range lanes {
		numTxs := metaData.TxsByLane[lane.Name()]
		if numTxs > uint64(len(txs)) {
			return types.ProposalInfo{}, nil, fmt.Errorf(
				"proposal metadata contains invalid number of transactions for lane %s; got %d, expected %d",
				lane.Name(),
				len(txs),
				numTxs,
			)
		}

		partialProposals[index] = txs[:numTxs]
		txs = txs[numTxs:]
	}

	// If there are any transactions remaining in the proposal, then the proposal is invalid.
	if len(txs) > 0 {
		return types.ProposalInfo{}, nil, fmt.Errorf("proposal contains invalid number of transactions")
	}

	return metaData, partialProposals, nil
}

// ValidateBlockLimits validates the block limits of the proposal against the block limits
// of the chain.
func (h *ProposalHandler) ValidateBlockLimits(finalProposal proposals.Proposal, proposalInfo types.ProposalInfo) error {
	// Conduct final checks on block size and gas limit.
	if finalProposal.Info.BlockSize != proposalInfo.BlockSize {
		h.logger.Error(
			"proposal block size does not match",
			"expected", proposalInfo.BlockSize,
			"got", finalProposal.Info.BlockSize,
		)

		return fmt.Errorf("proposal block size does not match")
	}

	if finalProposal.Info.GasLimit != proposalInfo.GasLimit {
		h.logger.Error(
			"proposal gas limit does not match",
			"expected", proposalInfo.GasLimit,
			"got", finalProposal.Info.GasLimit,
		)

		return fmt.Errorf("proposal gas limit does not match")
	}

	return nil
}

=======
>>>>>>> f7dfbda (feat: Greedy Algorithm for Lane Verification (#236))
// ChainPrepareLanes chains together the proposal preparation logic from each lane into a
// single function. The first lane in the chain is the first lane to be prepared and the
// last lane in the chain is the last lane to be prepared. In the case where any of the lanes
// fail to prepare the partial proposal, the lane that failed will be skipped and the next
// lane in the chain will be called to prepare the proposal.
func ChainPrepareLanes(chain []block.Lane) block.PrepareLanesHandler {
	if len(chain) == 0 {
		return nil
	}

	// Handle non-terminated decorators chain
	if (chain[len(chain)-1] != terminator.Terminator{}) {
		chain = append(chain, terminator.Terminator{})
	}

	return func(ctx sdk.Context, partialProposal proposals.Proposal) (finalProposal proposals.Proposal, err error) {
		lane := chain[0]

		// Cache the context in the case where any of the lanes fail to prepare the proposal.
		cacheCtx, write := ctx.CacheContext()

		// We utilize a recover to handle any panics or errors that occur during the preparation
		// of a lane's transactions. This defer will first check if there was a panic or error
		// thrown from the lane's preparation logic. If there was, we log the error, skip the lane,
		// and call the next lane in the chain to the prepare the proposal.
		defer func() {
			if rec := recover(); rec != nil || err != nil {
				if len(chain) <= 2 {
					// If there are only two lanes remaining, then the first lane in the chain
					// is the lane that failed to prepare the partial proposal and the second lane in the
					// chain is the terminator lane. We return the proposal as is.
					finalProposal, err = partialProposal, nil
				} else {
					// If there are more than two lanes remaining, then the first lane in the chain
					// is the lane that failed to prepare the proposal but the second lane in the
					// chain is not the terminator lane so there could potentially be more transactions
					// added to the proposal
					finalProposal, err = ChainPrepareLanes(chain[1:])(ctx, partialProposal)
				}
			} else {
				// Write the cache to the context since we know that the lane successfully prepared
				// the partial proposal. State is written to in a backwards, cascading fashion. This means
				// that the final context will only be updated after all other lanes have successfully
				// prepared the partial proposal.
				write()
			}
		}()

		return lane.PrepareLane(
			cacheCtx,
			partialProposal,
			ChainPrepareLanes(chain[1:]),
		)
	}
}

// ChainProcessLanes chains together the proposal verification logic from each lane
// into a single function. The first lane in the chain is the first lane to be verified and
// the last lane in the chain is the last lane to be verified. Each lane will validate
// the transactions that belong to the lane and pass any remaining transactions to the next
// lane in the chain. If any of the lanes fail to verify the transactions, the proposal will
// be rejected. If there are any remaining transactions after all lanes have been processed,
// the proposal will be rejected.
func ChainProcessLanes(chain []block.Lane) block.ProcessLanesHandler {
	if len(chain) == 0 {
		return nil
	}

	// Handle non-terminated decorators chain
	if (chain[len(chain)-1] != terminator.Terminator{}) {
		chain = append(chain, terminator.Terminator{})
	}

	return func(ctx sdk.Context, proposal proposals.Proposal, txs []sdk.Tx) (proposals.Proposal, error) {
		lane := chain[0]
		return lane.ProcessLane(ctx, proposal, txs, ChainProcessLanes(chain[1:]))
	}
}
