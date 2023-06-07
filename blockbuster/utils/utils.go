package utils

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkmempool "github.com/cosmos/cosmos-sdk/types/mempool"
	"github.com/skip-mev/pob/blockbuster"
)

// GetTxHashStr returns the hex-encoded hash of the transaction alongside the
// transaction bytes.
func GetTxHashStr(txEncoder sdk.TxEncoder, tx sdk.Tx) ([]byte, string, error) {
	txBz, err := txEncoder(tx)
	if err != nil {
		return nil, "", fmt.Errorf("failed to encode transaction: %w", err)
	}

	txHash := sha256.Sum256(txBz)
	txHashStr := hex.EncodeToString(txHash[:])

	return txBz, txHashStr, nil
}

// RemoveTxsFromLane removes the transactions from the given lane's mempool.
func RemoveTxsFromLane(txs map[sdk.Tx]struct{}, mempool sdkmempool.Mempool) error {
	for tx := range txs {
		if err := mempool.Remove(tx); err != nil {
			return err
		}
	}

	return nil
}

// GetMaxTxBytesForLane returns the maximum number of bytes that can be included in the proposal
// for the given lane.
func GetMaxTxBytesForLane(proposal *blockbuster.Proposal, ratio sdk.Dec) int64 {
	// In the case where the ratio is zero, we return the max tx bytes remaining. Note, the only
	// lane that should have a ratio of zero is the default lane. This means the default lane
	// will have no limit on the number of transactions it can include in a block and is only
	// limited by the maxTxBytes included in the PrepareProposalRequest.
	if ratio.IsZero() {
		remainder := proposal.MaxTxBytes - proposal.TotalTxBytes
		if remainder < 0 {
			return 0
		}

		return remainder
	}

	// Otherwise, we calculate the max tx bytes for the lane based on the ratio.
	return ratio.MulInt64(proposal.MaxTxBytes).TruncateInt().Int64()
}
