package blockbuster

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"cosmossdk.io/log"
	"cosmossdk.io/math"
	"github.com/skip-mev/pob/blockbuster/utils"
)

var _ BlockProposal = (*Proposal)(nil)

type (
	// LaneProposal defines the interface/APIs that are required for the proposal to interact
	// with a lane.
	LaneProposal interface {
		// Logger returns the lane's logger.
		Logger() log.Logger

		// GetMaxBlockSpace returns the maximum block space for the lane as a relative percentage.
		GetMaxBlockSpace() math.LegacyDec

		// Name returns the name of the lane.
		Name() string
	}

	// BlockProposal is the interface/APIs that are required for proposal creation + interacting with
	// and updating proposals. BlockProposals are iteratively updated as each lane prepares its
	// partial proposal. Each lane must call UpdateProposal with its partial proposal in PrepareLane. BlockProposals
	// can also include vote extensions, which are included at the top of the proposal.
	BlockProposal interface {
		// UpdateProposal updates the proposal with the given transactions. There are a
		// few invarients that are checked:
		//  1. The total size of the proposal must be less than the maximum number of bytes allowed.
		//  2. The total size of the partial proposal must be less than the maximum number of bytes allowed for
		//     the lane.
		UpdateProposal(lane LaneProposal, partialProposalTxs [][]byte) error

		// GetMaxTxBytes returns the maximum number of bytes that can be included in the proposal.
		GetMaxTxBytes() int64

		// GetTotalTxBytes returns the total number of bytes currently included in the proposal.
		GetTotalTxBytes() int64

		// GetTxs returns the transactions in the proposal.
		GetTxs() [][]byte

		// GetNumTxs returns the number of transactions in the proposal.
		GetNumTxs() int

		// Contains returns true if the proposal contains the given transaction.
		Contains(tx []byte) bool

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
	}
)

// NewProposal returns a new empty proposal.
func NewProposal(maxTxBytes int64) *Proposal {
	return &Proposal{
		txs:            make([][]byte, 0),
		voteExtensions: make([][]byte, 0),
		cache:          make(map[string]struct{}),
		maxTxBytes:     maxTxBytes,
	}
}

// UpdateProposal updates the proposal with the given transactions and total size. There are a
// few invarients that are checked:
//  1. The total size of the proposal must be less than the maximum number of bytes allowed.
//  2. The total size of the partial proposal must be less than the maximum number of bytes allowed for
//     the lane.
func (p *Proposal) UpdateProposal(lane LaneProposal, partialProposalTxs [][]byte) error {
	if len(partialProposalTxs) == 0 {
		return nil
	}

	partialProposalSize := int64(0)
	for _, tx := range partialProposalTxs {
		partialProposalSize += int64(len(tx))
	}

	// Invarient check: Ensure that the lane did not prepare a partial proposal that is too large.
	maxTxBytesForLane := utils.GetMaxTxBytesForLane(p.GetMaxTxBytes(), p.GetTotalTxBytes(), lane.GetMaxBlockSpace())
	if partialProposalSize > maxTxBytesForLane {
		return fmt.Errorf(
			"%s lane prepared a partial proposal that is too large: %d > %d",
			lane.Name(),
			partialProposalSize,
			maxTxBytesForLane,
		)
	}

	// Invarient check: Ensure that the lane did not prepare a block proposal that is too large.
	updatedSize := p.totalTxBytes + partialProposalSize
	if updatedSize > p.maxTxBytes {
		return fmt.Errorf(
			"lane %s prepared a block proposal that is too large: %d > %d",
			lane.Name(),
			p.totalTxBytes,
			p.maxTxBytes,
		)
	}
	p.totalTxBytes = updatedSize

	lane.Logger().Info(
		"adding transactions to proposal",
		"lane", lane.Name(),
		"num_txs", len(partialProposalTxs),
		"total_tx_bytes", partialProposalSize,
		"cumulative_size", updatedSize,
	)

	p.txs = append(p.txs, partialProposalTxs...)

	for _, tx := range partialProposalTxs {
		txHash := sha256.Sum256(tx)
		txHashStr := hex.EncodeToString(txHash[:])

		p.cache[txHashStr] = struct{}{}
	}

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

// GetMaxTxBytes returns the maximum number of bytes that can be included in the proposal.
func (p *Proposal) GetMaxTxBytes() int64 {
	return p.maxTxBytes
}

// GetTotalTxBytes returns the total number of bytes currently included in the proposal.
func (p *Proposal) GetTotalTxBytes() int64 {
	return p.totalTxBytes
}

// GetTxs returns the transactions in the proposal.
func (p *Proposal) GetTxs() [][]byte {
	return p.txs
}

// GetNumTxs returns the number of transactions in the proposal.
func (p *Proposal) GetNumTxs() int {
	return len(p.txs)
}

// Contains returns true if the proposal contains the given transaction.
func (p *Proposal) Contains(tx []byte) bool {
	txHash := sha256.Sum256(tx)
	txHashStr := hex.EncodeToString(txHash[:])

	_, ok := p.cache[txHashStr]
	return ok
}
