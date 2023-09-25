package proposals

import (
	"fmt"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/skip-mev/block-sdk/block/utils"
)

type (
	// PartialProposal defines the transactions, size, and more
	// of a partial proposal.
	PartialProposal struct {
		// SdkTxs is the list of transactions in the proposal.
		SdkTxs []sdk.Tx
		// txs is the list of transactions in the proposal.
		Txs [][]byte
		// hashes is the list of hashes of transactions in the proposal.
		Hashes map[string]struct{}
		// size is the total size of the proposal.
		Size int64
		// gasLimit is the total gas limit of the proposal.
		GasLimit uint64
		// Limit contains the partial proposal constaints.
		Limit LaneLimits
	}

	// LaneLimits defines the total number of bytes and units of gas that can be included in a block proposal
	// for a given lane.
	LaneLimits struct {
		MaxTxBytes int64
		MaxGas     uint64
	}
)

// NewPartialProposal returns a new empty partial proposal.
func NewPartialProposal(txs [][]byte) PartialProposal {
	return PartialProposal{
		Txs:    txs,
		Hashes: make(map[string]struct{}),
	}
}

// NewPartialProposalFromTxs returns a new partial proposal from a list of transactions.
func NewPartialProposalFromTxs(
	txEncoder sdk.TxEncoder,
	partialProposalTxs []sdk.Tx,
	limit LaneLimits,
) (PartialProposal, error) {
	// Aggregate info from the transactions.
	hashes := make(map[string]struct{})
	txs := make([][]byte, len(partialProposalTxs))
	partialProposalSize := int64(0)
	partialProposalGasLimit := uint64(0)

	for index, tx := range partialProposalTxs {
		txInfo, err := utils.GetTxInfo(txEncoder, tx)
		if err != nil {
			return PartialProposal{}, fmt.Errorf("err retriveing transaction info: %s", err)
		}

		hashes[txInfo.Hash] = struct{}{}
		partialProposalSize += txInfo.Size
		partialProposalGasLimit += txInfo.GasLimit
		txs[index] = txInfo.TxBytes
	}

	proposal := PartialProposal{
		SdkTxs:   partialProposalTxs,
		Txs:      txs,
		Hashes:   hashes,
		Size:     partialProposalSize,
		GasLimit: partialProposalGasLimit,
		Limit:    limit,
	}

	if err := proposal.ValidateBasic(); err != nil {
		return PartialProposal{}, err
	}

	return proposal, nil
}

func (p *PartialProposal) ValidateBasic() error {
	if p.Limit.MaxGas < p.GasLimit {
		return fmt.Errorf("partial proposal gas limit is above the maximum allowed")
	}

	if p.Limit.MaxTxBytes < p.Size {
		return fmt.Errorf("partial proposal size is above the maximum allowed")
	}

	return nil
}

// NewLaneLimits returns a new lane limit.
func NewLaneLimits(maxTxBytesLimit int64, maxGasLimit uint64) LaneLimits {
	return LaneLimits{
		MaxTxBytes: maxTxBytesLimit,
		MaxGas:     maxGasLimit,
	}
}

// GetLaneLimits returns the maximum number of bytes and gas limit that can be
// included/consumed in the proposal for the given lane.
func GetLaneLimits(
	maxTxBytes, consumedTxBytes int64,
	maxGaslimit, consumedGasLimit uint64,
	ratio math.LegacyDec,
) LaneLimits {
	var (
		txBytes  int64
		gasLimit uint64
	)

	// In the case where the ratio is zero, we return the max tx bytes remaining. Note, the only
	// lane that should have a ratio of zero is the default lane. This means the default lane
	// will have no limit on the number of transactions it can include in a block and is only
	// limited by the maxTxBytes included in the PrepareProposalRequest.
	if ratio.IsZero() {
		txBytes := maxTxBytes - consumedTxBytes
		if txBytes < 0 {
			txBytes = 0
		}

		// Unsigned subtraction needs an additional check
		if consumedGasLimit >= maxGaslimit {
			gasLimit = 0
		} else {
			gasLimit = maxGaslimit - consumedGasLimit
		}

		return NewLaneLimits(txBytes, gasLimit)
	}

	// Otherwise, we calculate the max tx bytes / gas limit for the lane based on the ratio.
	txBytes = ratio.MulInt64(maxTxBytes).TruncateInt().Int64()
	gasLimit = ratio.MulInt(math.NewIntFromUint64(maxGaslimit)).TruncateInt().Uint64()

	return NewLaneLimits(txBytes, gasLimit)
}
