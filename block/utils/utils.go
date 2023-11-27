package utils

import (
	"encoding/hex"
	"fmt"
	"strings"

	comettypes "github.com/cometbft/cometbft/types"

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

	txHashStr := strings.ToUpper(hex.EncodeToString(comettypes.Tx(txBz).Hash()))

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
