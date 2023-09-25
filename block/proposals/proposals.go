package proposals

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/skip-mev/block-sdk/block/proposals/types"
)

type (
	// Proposal defines a block proposal type.
	Proposal struct {
		// txs is the list of transactions in the proposal.
		txs [][]byte
		// cache is a cache of the selected transactions in the proposal.
		cache map[string]struct{}
		// limit contains the block limits.
		info BlockInfo
		// txEncoder is the transaction encoder.
		txEncoder sdk.TxEncoder
		// metaData is the metadata of the proposal.
		metaData types.ProposalMetaData
	}

	// BlockLimits defines the total number of bytes and units of gas that can be included in a block proposal.
	BlockInfo struct {
		MaxTxBytes  int64
		MaxGas      uint64
		CurrentSize int64
		CurrentGas  uint64
	}
)

// NewProposal returns a new empty proposal.
func NewProposal(txEncoder sdk.TxEncoder, maxTxBytes int64, maxGasLimit uint64) Proposal {
	return Proposal{
		txEncoder: txEncoder,
		info:      NewBlockLimits(maxTxBytes, maxGasLimit),
		txs:       make([][]byte, 0),
		cache:     make(map[string]struct{}),
		metaData:  NewProposalMetaData(),
	}
}

// NewBlockLimits returns a new block limits.
func NewBlockLimits(maxTxBytes int64, maxGasLimit uint64) BlockInfo {
	return BlockInfo{
		MaxTxBytes: maxTxBytes,
		MaxGas:     maxGasLimit,
	}
}

// NewProposalMetaData returns a new proposal metadata.
func NewProposalMetaData() types.ProposalMetaData {
	return types.ProposalMetaData{
		Lanes: make(map[string]*types.LaneMetaData),
	}
}

// GetProposal returns all of the transactions in the proposal along with the vote extensions
// at the top of the proposal.
func (p *Proposal) GetProposal() [][]byte {
	// marshall the metadata into the first slot of the proposal.
	metaData, err := p.metaData.Marshal()
	if err != nil {
		// This should never happen
		panic(err)
	}

	proposal := [][]byte{metaData}
	proposal = append(proposal, p.txs...)

	return proposal
}

// GetStatistics returns the various statistics of the proposal.
func (p *Proposal) GetMetaData() *types.ProposalMetaData {
	return &p.metaData
}

// GetMaxGasLimit returns the maximum gas limit that can be included in the proposal.
func (p *Proposal) GetMaxGasLimit() uint64 {
	return p.info.MaxGas
}

// GetMaxTxBytes returns the maximum number of bytes that can be included in the proposal.
func (p *Proposal) GetMaxTxBytes() int64 {
	return p.info.MaxTxBytes
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
