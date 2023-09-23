package proposals

import "cosmossdk.io/math"

// LaneLimits defines the total number of bytes and units of gas that can be included in a block proposal
// for a given lane.
type LaneLimits struct {
	MaxTxBytes int64
	MaxGas     uint64
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
