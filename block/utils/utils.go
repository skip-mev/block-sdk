package utils

import (
	"encoding/hex"
	"fmt"
	"strings"

	comettypes "github.com/cometbft/cometbft/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkmempool "github.com/cosmos/cosmos-sdk/types/mempool"

	signerextraction "github.com/skip-mev/block-sdk/v2/adapters/signer_extraction_adapter"
)

// TxWithInfo contains the information required for a transaction to be
// included in a proposal.
type TxWithInfo struct {
	// Hash is the hex-encoded hash of the transaction.
	Hash string
	// Size is the size of the transaction in bytes.
	Size int64
	// GasLimit is the gas limit of the transaction.
	GasLimit uint64
	// TxBytes is the bytes of the transaction.
	TxBytes []byte
	// Priority defines the priority of the transaction.
	Priority any
	// Signers defines the signers of a transaction.
	Signers []signerextraction.SignerData
}

// NewTxInfo returns a new TxInfo instance.
func NewTxInfo(
	hash string,
	size int64,
	gasLimit uint64,
	txBytes []byte,
	priority any,
	signers []signerextraction.SignerData,
) TxWithInfo {
	return TxWithInfo{
		Hash:     hash,
		Size:     size,
		GasLimit: gasLimit,
		TxBytes:  txBytes,
		Priority: priority,
		Signers:  signers,
	}
}

// String implements the fmt.Stringer interface.
func (t TxWithInfo) String() string {
	return fmt.Sprintf("TxWithInfo{Hash: %s, Size: %d, GasLimit: %d, Priority: %s, Signers: %v}",
		t.Hash, t.Size, t.GasLimit, t.Priority, t.Signers)
}

// GetTxHash returns the string hash representation of a transaction.
func GetTxHash(encoder sdk.TxEncoder, tx sdk.Tx) (string, error) {
	txBz, err := encoder(tx)
	if err != nil {
		return "", fmt.Errorf("failed to encode transaction: %w", err)
	}

	txHashStr := strings.ToUpper(hex.EncodeToString(comettypes.Tx(txBz).Hash()))
	return txHashStr, nil
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
