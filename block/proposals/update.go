package proposals

import (
	"fmt"

	"cosmossdk.io/math"

	"github.com/skip-mev/block-sdk/v2/block/utils"
)

// Lane defines the contract interface for a lane.
type Lane interface {
	Name() string
	GetMaxBlockSpace() math.LegacyDec
}

// UpdateProposal updates the proposal with the given transactions and lane limits. There are a
// few invariants that are checked:
//  1. The total size of the proposal must be less than the maximum number of bytes allowed.
//  2. The total size of the partial proposal must be less than the maximum number of bytes allowed for
//     the lane.
//  3. The total gas limit of the proposal must be less than the maximum gas limit allowed.
//  4. The total gas limit of the partial proposal must be less than the maximum gas limit allowed for
//     the lane.
//  5. The lane must not have already prepared a partial proposal.
//  6. The transaction must not already be in the proposal.
func (p *Proposal) UpdateProposal(lane Lane, partialProposal []utils.TxWithInfo) error {
	if len(partialProposal) == 0 {
		return nil
	}

	// invariant check: Ensure we have not already prepared a partial proposal for this lane.
	if _, ok := p.Info.TxsByLane[lane.Name()]; ok {
		return fmt.Errorf("lane %s already prepared a partial proposal", lane)
	}

	// Aggregate info from the transactions.
	hashes := make(map[string]struct{})
	txs := make([][]byte, len(partialProposal))
	partialProposalSize := int64(0)
	partialProposalGasLimit := uint64(0)

	for index, tx := range partialProposal {
		p.Logger.Info(
			"updating proposal with tx",
			"index", index+len(p.Txs),
			"lane", lane.Name(),
			"hash", tx.Hash,
			"size", tx.Size,
			"gas_limit", tx.GasLimit,
			"signers", tx.Signers,
			"priority", tx.Priority,
		)

		// invariant check: Ensure that the transaction is not already in the proposal.
		if _, ok := p.Cache[tx.Hash]; ok {
			return fmt.Errorf("transaction %s is already in the proposal", tx.Hash)
		}

		hashes[tx.Hash] = struct{}{}
		partialProposalSize += tx.Size
		partialProposalGasLimit += tx.GasLimit
		txs[index] = tx.TxBytes
	}

	// invariant check: Ensure that the partial proposal is not too large.
	limit := p.GetLaneLimits(lane.GetMaxBlockSpace())
	if partialProposalSize > limit.MaxTxBytes {
		return fmt.Errorf(
			"partial proposal is too large: %d > %d",
			partialProposalSize,
			limit.MaxTxBytes,
		)
	}

	// invariant check: Ensure that the partial proposal does not consume too much gas.
	if partialProposalGasLimit > limit.MaxGasLimit {
		return fmt.Errorf(
			"partial proposal consumes too much gas: %d > %d",
			partialProposalGasLimit,
			limit.MaxGasLimit,
		)
	}

	// invariant check: Ensure that the lane did not prepare a block proposal that is too large.
	updatedSize := p.Info.BlockSize + partialProposalSize
	if updatedSize > p.Info.MaxBlockSize {
		return fmt.Errorf(
			"block proposal is too large: %d > %d",
			updatedSize,
			p.Info.MaxBlockSize,
		)
	}

	// invariant check: Ensure that the lane did not prepare a block proposal that consumes too much gas.
	updatedGasLimit := p.Info.GasLimit + partialProposalGasLimit
	if updatedGasLimit > p.Info.MaxGasLimit {
		return fmt.Errorf(
			"block proposal consumes too much gas: %d > %d",
			updatedGasLimit,
			p.Info.MaxGasLimit,
		)
	}

	// Update the proposal.
	p.Info.BlockSize = updatedSize
	p.Info.GasLimit = updatedGasLimit

	// Update the lane info.
	p.Info.TxsByLane[lane.Name()] = uint64(len(partialProposal))

	// Update the proposal.
	p.Txs = append(p.Txs, txs...)
	for hash := range hashes {
		p.Cache[hash] = struct{}{}
	}

	return nil
}
