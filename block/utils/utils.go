package utils

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkmempool "github.com/cosmos/cosmos-sdk/types/mempool"
)

type (
	// TxInfo is the information about a transaction.
	TxInfo struct {
		// Hash is the hex-encoded hash of the transaction.
		Hash string
		// Size is the size of the transaction in bytes.
		Size int64
		// Gas is the gas limit of the transaction.
		GasLimit uint64
		// TxBytes is the bytes of the transaction.
		TxBytes []byte
	}
)

// GetTxHashStr returns the hex-encoded hash of the transaction alongside the
// transaction bytes.
func GetTxInfo(txEncoder sdk.TxEncoder, tx sdk.Tx) (TxInfo, error) {
	txBz, err := txEncoder(tx)
	if err != nil {
		return TxInfo{}, fmt.Errorf("failed to encode transaction: %w", err)
	}

	txHash := sha256.Sum256(txBz)
	txHashStr := hex.EncodeToString(txHash[:])

	// TODO: Does anything need to be done to support EVM transactions?
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

// RemoveTxsFromLane removes the transactions from the given lane's mempool.
func RemoveTxsFromLane(txs []sdk.Tx, mempool sdkmempool.Mempool) error {
	for _, tx := range txs {
		if err := mempool.Remove(tx); err != nil {
			return err
		}
	}

	return nil
}
