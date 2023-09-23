package proposals

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/skip-mev/block-sdk/block/utils"
)

// PartialProposal defines the transactions, size, and more
// of a partial proposal.
type PartialProposal struct {
	// txs is the list of transactions in the proposal.
	Txs [][]byte
	// hashes is the list of hashes of transactions in the proposal.
	Hashes map[string]struct{}
	// size is the total size of the proposal.
	Size int64
	// gasLimit is the total gas limit of the proposal.
	GasLimit uint64
}

// NewPartialProposal returns a new empty partial proposal.
func NewPartialProposal() *PartialProposal {
	return &PartialProposal{
		Txs:    make([][]byte, 0),
		Hashes: make(map[string]struct{}),
	}
}

// NewPartialProposalFromTxs returns a new partial proposal from a list of transactions.
func NewPartialProposalFromTxs(txEncoder sdk.TxEncoder, partialProposalTxs []sdk.Tx) (PartialProposal, error) {
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

	return PartialProposal{
		Txs:      txs,
		Hashes:   hashes,
		Size:     partialProposalSize,
		GasLimit: partialProposalGasLimit,
	}, nil
}
