package proposals

import (
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/skip-mev/block-sdk/block/proposals/types"
)

type (
	// Proposal defines a block proposal type.
	Proposal struct {
		// txs is the list of transactions in the proposal.
		Txs [][]byte
		// cache is a cache of the selected transactions in the proposal.
		Cache map[string]struct{}
		// MaxBlockSize corresponds to the maximum number of bytes allowed in the block.
		MaxBlockSize int64
		// MaxGasLimit corresponds to the maximum gas limit allowed in the block.
		MaxGasLimit uint64
		// BlockSize corresponds to the current size of the block.
		BlockSize int64
		// GasLimt corresponds to the current gas limit of the block.
		GasLimt uint64
		// txEncoder is the transaction encoder.
		TxEncoder sdk.TxEncoder
		// laneInfo contains information about the various lanes that built the proposal.
		Info types.ProposalInfo
	}
)

// NewProposal returns a new empty proposal.
func NewProposal(txEncoder sdk.TxEncoder, maxBlockSize int64, maxGasLimit uint64) Proposal {
	return Proposal{
		TxEncoder:    txEncoder,
		MaxBlockSize: maxBlockSize,
		MaxGasLimit:  maxGasLimit,
		Txs:          make([][]byte, 0),
		Cache:        make(map[string]struct{}),
		Info:         types.ProposalInfo{TxsByLane: make(map[string]uint64)},
	}
}

// GetProposalWithInfo returns all of the transactions in the proposal along with information
// about the lanes that built the proposal.
func (p *Proposal) GetProposalWithInfo() ([][]byte, error) {
	// Marshall the laneInfo into the first slot of the proposal.
	laneInfo, err := p.Info.Marshal()
	if err != nil {
		return nil, err
	}

	proposal := [][]byte{laneInfo}
	proposal = append(proposal, p.Txs...)

	return proposal, nil
}

// GetLaneLimits returns the maximum number of bytes and gas limit that can be
// included/consumed in the proposal for the given lane.
func (p *Proposal) GetLaneLimits(ratio math.LegacyDec) LaneLimits {
	var (
		txBytes  int64
		gasLimit uint64
	)

	// In the case where the ratio is zero, we return the max tx bytes remaining. Note, the only
	// lane that should have a ratio of zero is the default lane. This means the default lane
	// will have no limit on the number of transactions it can include in a block and is only
	// limited by the maxTxBytes included in the PrepareProposalRequest.
	if ratio.IsZero() {
		txBytes = p.MaxBlockSize - p.BlockSize
		if txBytes < 0 {
			txBytes = 0
		}

		// Unsigned subtraction needs an additional check
		if p.GasLimt >= p.MaxGasLimit {
			gasLimit = 0
		} else {
			gasLimit = p.MaxGasLimit - p.GasLimt
		}
	} else {
		// Otherwise, we calculate the max tx bytes / gas limit for the lane based on the ratio.
		txBytes = ratio.MulInt64(p.MaxBlockSize).TruncateInt().Int64()
		gasLimit = ratio.MulInt(math.NewIntFromUint64(p.MaxGasLimit)).TruncateInt().Uint64()
	}

	return LaneLimits{
		MaxTxBytes:  txBytes,
		MaxGasLimit: gasLimit,
	}
}

// Contains returns true if the proposal contains the given transaction.
func (p *Proposal) Contains(txHash string) bool {
	_, ok := p.Cache[txHash]
	return ok
}
