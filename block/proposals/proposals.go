package proposals

import (
	"cosmossdk.io/log"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/skip-mev/block-sdk/v2/block/proposals/types"
)

type (
	// Proposal defines a block proposal type.
	Proposal struct {
		Logger log.Logger

		// Txs is the list of transactions in the proposal.
		Txs [][]byte
		// Cache is a cache of the selected transactions in the proposal.
		Cache map[string]struct{}
		// Info contains information about the state of the proposal.
		Info types.ProposalInfo
	}
)

// NewProposalWithContext returns a new empty proposal.
func NewProposalWithContext(ctx sdk.Context, logger log.Logger) Proposal {
	maxBlockSize, maxGasLimit := GetBlockLimits(ctx)
	return NewProposal(logger, maxBlockSize, maxGasLimit)
}

// NewProposal returns a new empty proposal. Any transactions added to the proposal
// will be subject to the given max block size and max gas limit.
func NewProposal(logger log.Logger, maxBlockSize int64, maxGasLimit uint64) Proposal {
	return Proposal{
		Logger: logger,
		Txs:    make([][]byte, 0),
		Cache:  make(map[string]struct{}),
		Info: types.ProposalInfo{
			TxsByLane:    make(map[string]uint64),
			MaxBlockSize: maxBlockSize,
			MaxGasLimit:  maxGasLimit,
		},
	}
}

// GetProposalWithInfo returns all of the transactions in the proposal along with information
// about the lanes that built the proposal.
//
// NOTE: This is currently not used in production but likely will be once
// ABCI 3.0 is released.
func (p *Proposal) GetProposalWithInfo() ([][]byte, error) {
	// Marshall the proposal info into the first slot of the proposal.
	infoBz, err := p.Info.Marshal()
	if err != nil {
		return nil, err
	}

	proposal := [][]byte{infoBz}
	proposal = append(proposal, p.Txs...)

	return proposal, nil
}

// GetLaneLimits returns the maximum number of bytes and gas limit that can be
// included/consumed in the proposal for the given block space ratio. Lane's
// must first call this function to determine the maximum number of bytes and
// gas limit they can include in the proposal before constructing a partial
// proposal.
func (p *Proposal) GetLaneLimits(ratio math.LegacyDec) LaneLimits {
	var (
		txBytes  int64
		gasLimit uint64
	)

	// In the case where the ratio is zero, we return the max tx bytes remaining.
	// Note, the only lane that should have a ratio of zero is the default lane.
	if ratio.IsZero() {
		txBytes = p.Info.MaxBlockSize - p.Info.BlockSize
		if txBytes < 0 {
			txBytes = 0
		}

		// Unsigned subtraction needs an additional check
		if p.Info.GasLimit >= p.Info.MaxGasLimit {
			gasLimit = 0
		} else {
			gasLimit = p.Info.MaxGasLimit - p.Info.GasLimit
		}
	} else {
		// Otherwise, we calculate the max tx bytes / gas limit for the lane based on the ratio.
		txBytes = ratio.MulInt64(p.Info.MaxBlockSize).TruncateInt().Int64()
		gasLimit = ratio.MulInt(math.NewIntFromUint64(p.Info.MaxGasLimit)).TruncateInt().Uint64()
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
