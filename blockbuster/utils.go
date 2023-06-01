package blockbuster

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// GetTxHashStr returns the hex-encoded hash of the transaction.
func GetTxHashStr(txEncoder sdk.TxEncoder, tx sdk.Tx) (string, error) {
	txBz, err := txEncoder(tx)
	if err != nil {
		return "", fmt.Errorf("failed to encode transaction: %w", err)
	}

	txHash := sha256.Sum256(txBz)
	txHashStr := hex.EncodeToString(txHash[:])

	return txHashStr, nil
}
