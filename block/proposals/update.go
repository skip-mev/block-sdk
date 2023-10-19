package proposals

import (
	"fmt"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	signerextraction "github.com/skip-mev/block-sdk/adapters/signer_extraction_adapter"
	"github.com/skip-mev/block-sdk/block/utils"
)

// Lane defines the contract interface for a lane.
type Lane interface {
	GetMaxBlockSpace() math.LegacyDec
	Name() string
}

// UpdateProposal updates the proposal with the given transactions and lane limits. There are a
// few invariants that are checked:
//  1. The total size of the proposal must be less than the maximum number of bytes allowed.
//  2. The total size of the partial proposal must be less than the maximum number of bytes allowed for
//     the lane.
//  3. The total gas limit of the proposal must be less than the maximum gas limit allowed.
//  4. The total gas limit of the partial proposal must be less than the maximum gas limit allowed for
//     the lane.
//  5. The lane must not have already prepared a partial proposal.
//  6. The transaction must not already be in the proposal.
func (p *Proposal) UpdateProposal(lane Lane, partialProposal []sdk.Tx) error {
	if len(partialProposal) == 0 {
		return nil
	}

	// invariant check: Ensure we have not already prepared a partial proposal for this lane.
	if _, ok := p.Info.TxsByLane[lane.Name()]; ok {
		return fmt.Errorf("lane %s already prepared a partial proposal", lane)
	}

	// Aggregate info from the transactions.
	hashes := make(map[string]struct{})
	txs := make([][]byte, len(partialProposal))
	partialProposalSize := int64(0)
	partialProposalGasLimit := uint64(0)

	signerAdapter := signerextraction.NewDefaultAdapter()

	for index, tx := range partialProposal {
		txInfo, err := utils.GetTxInfo(p.TxEncoder, tx)
		if err != nil {
			return fmt.Errorf("err retrieving transaction info: %s", err)
		}

		feeTx := tx.(sdk.FeeTx)
		signers, err := signerAdapter.GetSigners(tx)
		if err != nil {
			return fmt.Errorf("err retrieving signers: %s", err)
		}

		p.Logger.Info(
			"updating proposal with tx",
			"index", index,
			"lane", lane.Name(),
			"tx_hash", txInfo.Hash,
			"tx_size", txInfo.Size,
			"tx_gas_limit", txInfo.GasLimit,
			// "tx_bytes", txInfo.TxBytes,
			// "raw_tx", base64.StdEncoding.EncodeToString(txInfo.TxBytes),
			"fee", feeTx.GetFee(),
			"gas", feeTx.GetGas(),
			"signer", signers[0].Signer.String(),
			"nonce", signers[0].Sequence,
		)

		// invariant check: Ensure that the transaction is not already in the proposal.
		if _, ok := p.Cache[txInfo.Hash]; ok {
			return fmt.Errorf("transaction %s is already in the proposal", txInfo.Hash)
		}

		hashes[txInfo.Hash] = struct{}{}
		partialProposalSize += txInfo.Size
		partialProposalGasLimit += txInfo.GasLimit
		txs[index] = txInfo.TxBytes
	}

	// invariant check: Ensure that the partial proposal is not too large.
	limit := p.GetLaneLimits(lane.GetMaxBlockSpace())
	if partialProposalSize > limit.MaxTxBytes {
		return fmt.Errorf(
			"partial proposal is too large: %d > %d",
			partialProposalSize,
			limit.MaxTxBytes,
		)
	}

	// invariant check: Ensure that the partial proposal does not consume too much gas.
	if partialProposalGasLimit > limit.MaxGasLimit {
		return fmt.Errorf(
			"partial proposal consumes too much gas: %d > %d",
			partialProposalGasLimit,
			limit.MaxGasLimit,
		)
	}

	// invariant check: Ensure that the lane did not prepare a block proposal that is too large.
	updatedSize := p.Info.BlockSize + partialProposalSize
	if updatedSize > p.Info.MaxBlockSize {
		return fmt.Errorf(
			"block proposal is too large: %d > %d",
			updatedSize,
			p.Info.MaxBlockSize,
		)
	}

	// invariant check: Ensure that the lane did not prepare a block proposal that consumes too much gas.
	updatedGasLimit := p.Info.GasLimit + partialProposalGasLimit
	if updatedGasLimit > p.Info.MaxGasLimit {
		return fmt.Errorf(
			"block proposal consumes too much gas: %d > %d",
			updatedGasLimit,
			p.Info.MaxGasLimit,
		)
	}

	// Update the proposal.
	p.Info.BlockSize = updatedSize
	p.Info.GasLimit = updatedGasLimit

	// Update the lane info.
	p.Info.TxsByLane[lane.Name()] = uint64(len(partialProposal))

	// Update the proposal.
	p.Txs = append(p.Txs, txs...)
	for hash := range hashes {
		p.Cache[hash] = struct{}{}
	}

	return nil
}
