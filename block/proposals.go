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
		//  3. The total gas limit of the proposal must be less than the maximum gas limit allowed.
		//  4. The total gas limit of the partial proposal must be less than the maximum gas limit allowed for
		//     the lane.
		UpdateProposal(lane Lane, partialProposalTxs []sdk.Tx) error

		// GetStatistics returns the statistics / info of the proposal.
		GetStatistics() ProposalStatistics

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
		// totalTxBytesUsed is the total number of bytes currently included in the proposal.
		totalTxBytesUsed int64
		// maxTxBytes is the maximum number of bytes that can be included in the proposal.
		maxTxBytes int64
		// totalGasLimitUsed is the total gas limit currently included in the proposal.
		totalGasLimitUsed uint64
		// maxGasLimit is the maximum gas limit that can be included in the proposal.
		maxGasLimit uint64
		// txEncoder is the transaction encoder.
		txEncoder sdk.TxEncoder
	}

	// ProposalStatistics defines the basic info/statistics of a proposal.
	ProposalStatistics struct {
		// NumVoteExtensions is the number of vote extensions in the proposal.
		NumVoteExtensions int
		// NumTxs is the number of transactions in the proposal.
		NumTxs int
		// TotalTxBytesUsed is the total number of bytes currently included in the proposal.
		TotalTxBytesUsed int64
		// MaxTxBytes is the maximum number of bytes that can be included in the proposal.
		MaxTxBytes int64
		// TotalGasLimitUsed is the total gas limit currently included in the proposal.
		TotalGasLimitUsed uint64
		// MaxGas is the maximum gas limit that can be included in the proposal.
		MaxGas uint64
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
//  3. The total gas limit of the proposal must be less than the maximum gas limit allowed.
//  4. The total gas limit of the partial proposal must be less than the maximum gas limit allowed for
//     the lane.
func (p *Proposal) UpdateProposal(lane Lane, partialProposalTxs []sdk.Tx) error {
	if len(partialProposalTxs) == 0 {
		return nil
	}

	hashes := make(map[string]struct{})
	txs := make([][]byte, len(partialProposalTxs))
	partialProposalSize := int64(0)
	partialProposalGasLimit := uint64(0)

	// Aggregate info from the transactions.
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

	limits := GetLaneLimits(
		p.maxTxBytes, p.totalTxBytesUsed,
		p.maxGasLimit, p.totalGasLimitUsed,
		lane.GetMaxBlockSpace(),
	)

	// Invarient check: Ensure that the lane did not prepare a partial proposal that is too large.
	if partialProposalSize > limits.MaxTxBytes {
		return fmt.Errorf(
			"%s lane prepared a partial proposal that is too large: %d > %d",
			lane.Name(),
			partialProposalSize,
			limits.MaxTxBytes,
		)
	}

	// Invarient check: Ensure that the lane did not prepare a partial proposal that consumes too much gas.
	if partialProposalGasLimit > limits.MaxGas {
		return fmt.Errorf(
			"%s lane prepared a partial proposal that consumes too much gas: %d > %d",
			lane.Name(),
			partialProposalGasLimit,
			limits.MaxGas,
		)
	}

	// Invarient check: Ensure that the lane did not prepare a block proposal that is too large.
	updatedSize := p.totalTxBytesUsed + partialProposalSize
	if updatedSize > p.maxTxBytes {
		return fmt.Errorf(
			"lane %s prepared a block proposal that is too large: %d > %d",
			lane.Name(),
			updatedSize,
			p.maxTxBytes,
		)
	}

	// Invarient check: Ensure that the lane did not prepare a block proposal that consumes too much gas.
	updatedGasLimit := p.totalGasLimitUsed + partialProposalGasLimit
	if updatedGasLimit > p.maxGasLimit {
		return fmt.Errorf(
			"lane %s prepared a block proposal that consumes too much gas: %d > %d",
			lane.Name(),
			updatedGasLimit,
			p.maxGasLimit,
		)
	}

	// Update the proposal
	p.totalTxBytesUsed = updatedSize
	p.totalGasLimitUsed = updatedGasLimit
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
		"partial_proposal_gas_limit", partialProposalGasLimit,
		"cumulative_proposal_gas_limit", updatedGasLimit,
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

// GetStatistics returns the various statistics of the proposal.
func (p *Proposal) GetStatistics() ProposalStatistics {
	return ProposalStatistics{
		NumVoteExtensions: len(p.voteExtensions),
		NumTxs:            len(p.txs),
		TotalTxBytesUsed:  p.totalTxBytesUsed,
		MaxTxBytes:        p.maxTxBytes,
		TotalGasLimitUsed: p.totalGasLimitUsed,
		MaxGas:            p.maxGasLimit,
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
