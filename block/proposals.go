package block

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

var _ BlockProposal = (*Proposal)(nil)

type (
	// BlockProposal is the interface/APIs that are required for proposal creation + interacting with
	// and updating proposals. BlockProposals are iteratively updated as each lane prepares its
	// partial proposal. Each lane must call UpdateProposal with its partial proposal in PrepareLane. BlockProposals
	// can also include vote extensions, which are included at the top of the proposal.
	BlockProposal interface { //nolint
		// UpdateProposal updates the proposal with the given transactions. There are a
		// few invarients that are checked:
		//  1. The total size of the proposal must be less than the maximum number of bytes allowed.
		//  2. The total size of the partial proposal must be less than the maximum number of bytes allowed for
		//     the lane.
		UpdateProposal(lane Lane, partialProposalTxs []sdk.Tx) error

		// GetProposalStatistics returns the statistics / info of the proposal.
		GetProposalStatistics() ProposalStatistics

		// GetTxs returns the transactions in the proposal.
		GetTxs() [][]byte

		// Contains returns true if the proposal contains the given transaction.
		Contains(txHash string) bool

		// AddVoteExtension adds a vote extension to the proposal.
		AddVoteExtension(voteExtension []byte)

		// GetVoteExtensions returns the vote extensions in the proposal.
		GetVoteExtensions() [][]byte

		// GetProposal returns all of the transactions in the proposal along with the vote extensions
		// at the top of the proposal.
		GetProposal() [][]byte
	}

	// Proposal defines a block proposal type.
	Proposal struct {
		// txs is the list of transactions in the proposal.
		txs [][]byte

		// voteExtensions is the list of vote extensions in the proposal.
		voteExtensions [][]byte

		// cache is a cache of the selected transactions in the proposal.
		cache map[string]struct{}

		// totalTxBytes is the total number of bytes currently included in the proposal.
		totalTxBytes int64

		// maxTxBytes is the maximum number of bytes that can be included in the proposal.
		maxTxBytes int64

		// totalGasLimit is the total gas limit currently included in the proposal.
		totalGasLimit uint64

		// maxGasLimit is the maximum gas limit that can be included in the proposal.
		maxGasLimit uint64

		// txEncoder is the transaction encoder.
		txEncoder sdk.TxEncoder
	}

	// ProposalStatistics defines the basic info/statistics of a proposal.
	ProposalStatistics struct {
		// NumTxs is the number of transactions in the proposal.
		NumTxs int

		// TotalTxBytes is the total number of bytes currently included in the proposal.
		TotalTxBytes int64

		// MaxTxBytes is the maximum number of bytes that can be included in the proposal.
		MaxTxBytes int64

		// TotalGasLimit is the total gas limit currently included in the proposal.
		TotalGasLimit uint64

		// MaxGasLimit is the maximum gas limit that can be included in the proposal.
		MaxGasLimit uint64
	}
)

// NewProposal returns a new empty proposal.
func NewProposal(txEncoder sdk.TxEncoder, maxTxBytes int64, maxGasLimit uint64) *Proposal {
	return &Proposal{
		txEncoder:      txEncoder,
		maxTxBytes:     maxTxBytes,
		maxGasLimit:    maxGasLimit,
		txs:            make([][]byte, 0),
		voteExtensions: make([][]byte, 0),
		cache:          make(map[string]struct{}),
	}
}

// UpdateProposal updates the proposal with the given transactions and total size. There are a
// few invarients that are checked:
//  1. The total size of the proposal must be less than the maximum number of bytes allowed.
//  2. The total size of the partial proposal must be less than the maximum number of bytes allowed for
//     the lane.
func (p *Proposal) UpdateProposal(lane Lane, partialProposalTxs []sdk.Tx) error {
	if len(partialProposalTxs) == 0 {
		return nil
	}

	hashes := make(map[string]struct{})
	txs := make([][]byte, len(partialProposalTxs))
	partialProposalSize := int64(0)
	partialProposalGasLimit := uint64(0)

	for index, tx := range partialProposalTxs {
		txInfo, err := GetTxInfo(p.txEncoder, tx)
		if err != nil {
			return fmt.Errorf("err retriveing transaction info: %s", err)
		}

		hashes[txInfo.Hash] = struct{}{}
		partialProposalSize += txInfo.Size
		partialProposalGasLimit += txInfo.GasLimit
		txs[index] = txInfo.TxBytes
	}

	laneLimit := GetLaneLimit(
		p.maxTxBytes, p.totalTxBytes,
		p.maxGasLimit, p.totalGasLimit,
		lane.GetMaxBlockSpace(),
	)

	// Invarient check: Ensure that the lane did not prepare a partial proposal that is too large.
	if partialProposalSize > laneLimit.MaxTxBytesLimit {
		return fmt.Errorf(
			"%s lane prepared a partial proposal that is too large: %d > %d",
			lane.Name(),
			partialProposalSize,
			laneLimit.MaxTxBytesLimit,
		)
	}

	// Invarient check: Ensure that the lane did not prepare a partial proposal that consumes too much gas.
	if partialProposalGasLimit > laneLimit.MaxGasLimit {
		return fmt.Errorf(
			"%s lane prepared a partial proposal that consumes too much gas: %d > %d",
			lane.Name(),
			partialProposalGasLimit,
			laneLimit.MaxGasLimit,
		)
	}

	// Invarient check: Ensure that the lane did not prepare a block proposal that is too large.
	updatedSize := p.totalTxBytes + partialProposalSize
	if updatedSize > p.maxTxBytes {
		return fmt.Errorf(
			"lane %s prepared a block proposal that is too large: %d > %d",
			lane.Name(),
			updatedSize,
			p.maxTxBytes,
		)
	}

	// Invarient check: Ensure that the lane did not prepare a block proposal that consumes too much gas.
	updatedGasLimit := p.totalGasLimit + partialProposalGasLimit
	if updatedGasLimit > p.maxGasLimit {
		return fmt.Errorf(
			"lane %s prepared a block proposal that consumes too much gas: %d > %d",
			lane.Name(),
			updatedGasLimit,
			p.maxGasLimit,
		)
	}

	// Update the proposal
	p.totalTxBytes = updatedSize
	p.totalGasLimit = updatedGasLimit
	p.txs = append(p.txs, txs...)

	for hash := range hashes {
		p.cache[hash] = struct{}{}

		lane.Logger().Info(
			"adding transaction to proposal",
			"lane", lane.Name(),
			"tx_hash", hash,
		)
	}

	lane.Logger().Info(
		"lane successfully updated proposal",
		"lane", lane.Name(),
		"num_txs", len(partialProposalTxs),
		"partial_proposal_size", partialProposalSize,
		"cumulative_proposal_size", updatedSize,
	)

	return nil
}

// GetProposal returns all of the transactions in the proposal along with the vote extensions
// at the top of the proposal.
func (p *Proposal) GetProposal() [][]byte {
	return append(p.voteExtensions, p.txs...)
}

// AddVoteExtension adds a vote extension to the proposal.
func (p *Proposal) AddVoteExtension(voteExtension []byte) {
	p.voteExtensions = append(p.voteExtensions, voteExtension)
}

// GetVoteExtensions returns the vote extensions in the proposal.
func (p *Proposal) GetVoteExtensions() [][]byte {
	return p.voteExtensions
}

// GetProposalStatus returns the status of the proposal.
func (p *Proposal) GetProposalStatistics() ProposalStatistics {
	return ProposalStatistics{
		NumTxs:        len(p.txs),
		TotalTxBytes:  p.totalTxBytes,
		MaxTxBytes:    p.maxTxBytes,
		TotalGasLimit: p.totalGasLimit,
		MaxGasLimit:   p.maxGasLimit,
	}
}

// GetTxs returns the transactions in the proposal.
func (p *Proposal) GetTxs() [][]byte {
	return p.txs
}

// Contains returns true if the proposal contains the given transaction.
func (p *Proposal) Contains(txHash string) bool {
	_, ok := p.cache[txHash]
	return ok
}
