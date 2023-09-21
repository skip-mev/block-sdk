package block

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"cosmossdk.io/math"
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

	// GasTx is the interface to retrieve the gas limit of a transaction.
	//
	// TODO: Do we need to add an adapter to make this work with EVM transactions?
	GasTx interface {
		GetGas() uint64
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

	gasTx, ok := tx.(GasTx)
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

// GetLaneLimit returns the maximum number of bytes and gas limit that can be
// included/consumed in the proposal for the given lane.
func GetLaneLimit(
	maxTxBytes, consumedTxBytes int64,
	maxGaslimit, consumedGasLimit uint64,
	ratio math.LegacyDec,
) LaneLimit {
	var (
		txBytes  int64
		gasLimit uint64
	)

	// In the case where the ratio is zero, we return the max tx bytes remaining. Note, the only
	// lane that should have a ratio of zero is the default lane. This means the default lane
	// will have no limit on the number of transactions it can include in a block and is only
	// limited by the maxTxBytes included in the PrepareProposalRequest.
	if ratio.IsZero() {
		txBytesRemaning := maxTxBytes - consumedTxBytes
		if txBytesRemaning < 0 {
			txBytes = 0
		}

		txBytes = txBytesRemaning

		return NewLaneLimit(txBytes, maxGaslimit-consumedGasLimit)
	}

	// Otherwise, we calculate the max tx bytes / gas limit for the lane based on the ratio.
	txBytes = ratio.MulInt64(maxTxBytes).TruncateInt().Int64()
	gasLimit = ratio.MulInt64(int64(maxGaslimit)).TruncateInt().Uint64()

	return NewLaneLimit(txBytes, gasLimit)
}
