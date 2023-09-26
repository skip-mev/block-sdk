package utils

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkmempool "github.com/cosmos/cosmos-sdk/types/mempool"
)

type (
	// TxInfo contains the information required for a transaction to be
	// included in a proposal.
	TxInfo struct {
		// Hash is the hex-encoded hash of the transaction.
		Hash string
		// Size is the size of the transaction in bytes.
		Size int64
		// GasLimit is the gas limit of the transaction.
		GasLimit uint64
		// TxBytes is the bytes of the transaction.
		TxBytes []byte
	}
)

// GetTxHashStr returns the TxInfo of a given transaction.
func GetTxInfo(txEncoder sdk.TxEncoder, tx sdk.Tx) (TxInfo, error) {
	txBz, err := txEncoder(tx)
	if err != nil {
		return TxInfo{}, fmt.Errorf("failed to encode transaction: %w", err)
	}

	txHash := sha256.Sum256(txBz)
	txHashStr := hex.EncodeToString(txHash[:])

	// TODO: Add an adapter to lanes so that this can be flexible to support EVM, etc.
	gasTx, ok := tx.(sdk.FeeTx)
	if !ok {
		return TxInfo{}, fmt.Errorf("failed to cast transaction to GasTx")
	}

	return TxInfo{
		Hash:     txHashStr,
		Size:     int64(len(txBz)),
		GasLimit: gasTx.GetGas(),
		TxBytes:  txBz,
	}, nil
}

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

// GetDecodedTxs returns the decoded transactions from the given bytes.
func GetDecodedTxs(txDecoder sdk.TxDecoder, txs [][]byte) ([]sdk.Tx, error) {
	var decodedTxs []sdk.Tx
	for _, txBz := range txs {
		tx, err := txDecoder(txBz)
		if err != nil {
			return nil, fmt.Errorf("failed to decode transaction: %w", err)
		}

		decodedTxs = append(decodedTxs, tx)
	}

	return decodedTxs, nil
}

// GetEncodedTxs returns the encoded transactions from the given bytes.
func GetEncodedTxs(txEncoder sdk.TxEncoder, txs []sdk.Tx) ([][]byte, error) {
	var encodedTxs [][]byte
	for _, tx := range txs {
		txBz, err := txEncoder(tx)
		if err != nil {
			return nil, fmt.Errorf("failed to encode transaction: %w", err)
		}

		encodedTxs = append(encodedTxs, txBz)
	}

	return encodedTxs, nil
}

// RemoveTxsFromLane removes the transactions from the given lane's mempool.
func RemoveTxsFromLane(txs []sdk.Tx, mempool sdkmempool.Mempool) error {
	for _, tx := range txs {
		if err := mempool.Remove(tx); err != nil {
			return err
		}
	}

	return nil
}

// GetMaxTxBytesForLane returns the maximum number of bytes that can be included in the proposal
// for the given lane.
func GetMaxTxBytesForLane(maxTxBytes, totalTxBytes int64, ratio math.LegacyDec) int64 {
	// In the case where the ratio is zero, we return the max tx bytes remaining. Note, the only
	// lane that should have a ratio of zero is the default lane. This means the default lane
	// will have no limit on the number of transactions it can include in a block and is only
	// limited by the maxTxBytes included in the PrepareProposalRequest.
	if ratio.IsZero() {
		remainder := maxTxBytes - totalTxBytes
		if remainder < 0 {
			return 0
		}

		return remainder
	}

	// Otherwise, we calculate the max tx bytes for the lane based on the ratio.
	return ratio.MulInt64(maxTxBytes).TruncateInt().Int64()
}
