package proposals

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/skip-mev/block-sdk/block/utils"
)

// UpdateProposal updates the proposal with the given transactions and lane limits. There are a
// few invarients that are checked:
//  1. The total size of the proposal must be less than the maximum number of bytes allowed.
//  2. The total size of the partial proposal must be less than the maximum number of bytes allowed for
//     the lane.
//  3. The total gas limit of the proposal must be less than the maximum gas limit allowed.
//  4. The total gas limit of the partial proposal must be less than the maximum gas limit allowed for
//     the lane.
//  5. The lane must not have already prepared a partial proposal.
//  6. The transaction must not already be in the proposal.
func (p *Proposal) UpdateProposal(lane string, partialProposal []sdk.Tx, limit LaneLimits) error {
	if len(partialProposal) == 0 {
		return nil
	}

	// Aggregate info from the transactions.
	hashes := make(map[string]struct{})
	txs := make([][]byte, len(partialProposal))
	partialProposalSize := int64(0)
	partialProposalGasLimit := uint64(0)

	for index, tx := range partialProposal {
		txInfo, err := utils.GetTxInfo(p.TxEncoder, tx)
		if err != nil {
			return fmt.Errorf("err retriveing transaction info: %s", err)
		}

		// Invarient check: Ensure that the transaction is not already in the proposal.
		if _, ok := p.Cache[txInfo.Hash]; ok {
			return fmt.Errorf("transaction %s is already in the proposal", txInfo.Hash)
		}

		hashes[txInfo.Hash] = struct{}{}
		partialProposalSize += txInfo.Size
		partialProposalGasLimit += txInfo.GasLimit
		txs[index] = txInfo.TxBytes
	}

	// Invarient check: Ensure that the partial proposal is not too large.
	if partialProposalSize > limit.MaxTxBytes {
		return fmt.Errorf(
			"partial proposal is too large: %d > %d",
			partialProposalSize,
			limit.MaxTxBytes,
		)
	}

	// Invarient check: Ensure that the partial proposal does not consume too much gas.
	if partialProposalGasLimit > limit.MaxGasLimit {
		return fmt.Errorf(
			"partial proposal consumes too much gas: %d > %d",
			partialProposalGasLimit,
			limit.MaxGasLimit,
		)
	}

	// Invarient check: Ensure that the lane did not prepare a block proposal that is too large.
	updatedSize := p.Info.BlockSize + partialProposalSize
	if updatedSize > p.Info.MaxBlockSize {
		return fmt.Errorf(
			"block proposal is too large: %d > %d",
			updatedSize,
			p.Info.MaxBlockSize,
		)
	}

	// Invarient check: Ensure that the lane did not prepare a block proposal that consumes too much gas.
	updatedGasLimit := p.Info.GasLimit + partialProposalGasLimit
	if updatedGasLimit > p.Info.MaxGasLimit {
		return fmt.Errorf(
			"block proposal consumes too much gas: %d > %d",
			updatedGasLimit,
			p.Info.MaxGasLimit,
		)
	}

	// Invarient check: Ensure we have not already prepared a partial proposal for this lane.
	if _, ok := p.Info.TxsByLane[lane]; ok {
		return fmt.Errorf("lane %s already prepared a partial proposal", lane)
	}

	// Update the proposal.
	p.Info.BlockSize = updatedSize
	p.Info.GasLimit = updatedGasLimit

	// Update the lane info.
	p.Info.TxsByLane[lane] = uint64(len(partialProposal))

	// Update the proposal.
	p.Txs = append(p.Txs, txs...)
	for hash := range hashes {
		p.Cache[hash] = struct{}{}
	}

	return nil
}
