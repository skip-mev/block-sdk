package proposals

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	// MaxUint64 is the maximum value of a uint64.
	MaxUint64 = 1<<64 - 1
)

type (
	// LaneLimits defines the constraints for a partial proposal. Each lane must only propose
	// transactions that satisfy these constraints. Otherwise the partial proposal update will
	// be rejected.
	LaneLimits struct {
		// MaxTxBytes is the maximum number of bytes allowed in the partial proposal.
		MaxTxBytes int64
		// MaxGasLimit is the maximum gas limit allowed in the partial proposal.
		MaxGasLimit uint64
	}
)

// GetBlockLimits retrieves the maximum number of bytes and gas limit allowed in a block.
func GetBlockLimits(ctx sdk.Context) (int64, uint64) {
	blockParams := ctx.ConsensusParams().Block

	// If the max gas is set to 0, then the max gas limit for the block can be infinite.
	// Otherwise we use the max gas limit casted as a uint64 which is how gas limits are
	// extracted from sdk.Tx's.
	var maxGasLimit uint64
	if maxGas := blockParams.MaxGas; maxGas > 0 {
		maxGasLimit = uint64(maxGas)
	} else {
		maxGasLimit = MaxUint64
	}

	return blockParams.MaxBytes, maxGasLimit
}
