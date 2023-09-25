package proposals

import (
	"fmt"

	"github.com/skip-mev/block-sdk/block/proposals/types"
)

// UpdateProposal updates the proposal with the given transactions and total size. There are a
// few invarients that are checked:
//  1. The total size of the proposal must be less than the maximum number of bytes allowed.
//  2. The total size of the partial proposal must be less than the maximum number of bytes allowed for
//     the lane.
//  3. The total gas limit of the proposal must be less than the maximum gas limit allowed.
//  4. The total gas limit of the partial proposal must be less than the maximum gas limit allowed for
//     the lane.
func (p *Proposal) UpdateProposal(
	lane string,
	partialProposal PartialProposal,
) error {
	if len(partialProposal.Txs) == 0 {
		return nil
	}

	// Invarient check: Ensure that the lane did not prepare a block proposal that is too large.
	updatedSize := p.metaData.TotalTxBytes + partialProposal.Size
	if updatedSize > p.info.MaxTxBytes {
		return fmt.Errorf(
			"block proposal is too large: %d > %d",
			updatedSize,
			p.info.MaxTxBytes,
		)
	}

	// Invarient check: Ensure that the lane did not prepare a block proposal that consumes too much gas.
	updatedGasLimit := p.metaData.TotalGasLimit + partialProposal.GasLimit
	if updatedGasLimit > p.info.MaxGas {
		return fmt.Errorf(
			"block proposal consumes too much gas: %d > %d",
			updatedGasLimit,
			p.info.MaxGas,
		)
	}

	if err := p.updateProposal(lane, partialProposal); err != nil {
		return err
	}

	return nil
}

// updateProposal updates the proposal with the given transactions and total size.
func (p *Proposal) updateProposal(lane string, partialProposal PartialProposal) error {
	// Ensure we have not already prepared a partial proposal for this lane.
	if _, ok := p.metaData.Lanes[lane]; ok {
		return fmt.Errorf("lane %s already prepared a partial proposal", lane)
	}

	// Aggregate info from the transactions for this lane.
	laneStatistics := &types.LaneMetaData{
		NumTxs:        uint64(len(partialProposal.Txs)),
		TotalTxBytes:  partialProposal.Size,
		TotalGasLimit: partialProposal.GasLimit,
	}
	p.metaData.Lanes[lane] = laneStatistics

	// Update the proposal.
	p.txs = append(p.txs, partialProposal.Txs...)
	for hash := range partialProposal.Hashes {
		p.cache[hash] = struct{}{}
	}

	return nil
}
