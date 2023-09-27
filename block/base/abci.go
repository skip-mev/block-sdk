package base

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/skip-mev/block-sdk/block"
	"github.com/skip-mev/block-sdk/block/proposals"
	"github.com/skip-mev/block-sdk/block/utils"
)

// PrepareLane will prepare a partial proposal for the lane. It will select transactions from the
// lane respecting the selection logic of the prepareLaneHandler. It will then update the partial
// proposal with the selected transactions. If the proposal is unable to be updated, we return an
// error. The proposal will only be modified if it passes all of the invariant checks.
func (l *BaseLane) PrepareLane(
	ctx sdk.Context,
	proposal proposals.Proposal,
	next block.PrepareLanesHandler,
) (proposals.Proposal, error) {
	limit := proposal.GetLaneLimits(l.cfg.MaxBlockSpace)

	// Select transactions from the lane respecting the selection logic of the lane and the
	// max block space for the lane.
	txsToInclude, txsToRemove, err := l.prepareLaneHandler(ctx, proposal, limit)
	if err != nil {
		return proposal, err
	}

	// Remove all transactions that were invalid during the creation of the partial proposal.
	if err := utils.RemoveTxsFromLane(txsToRemove, l); err != nil {
		l.Logger().Error(
			"failed to remove transactions from lane",
			"lane", l.Name(),
			"err", err,
		)
	}

	// Update the proposal with the selected transactions. This fails if the lane attempted to add
	// more transactions than the allocated max block space for the lane.
	if err := proposal.UpdateProposal(l.Name(), txsToInclude, limit); err != nil {
		l.Logger().Error(
			"failed to update proposal",
			"lane", l.Name(),
			"err", err,
			"num_txs_to_add", len(txsToInclude),
			"num_txs_to_remove", len(txsToRemove),
			"max_lane_size", limit.MaxTxBytes,
			"max_lane_gas_limit", limit.MaxGasLimit,
			"max_block_size", l.cfg.MaxBlockSpace,
		)

		return proposal, err
	}

	l.Logger().Info(
		"lane prepared",
		"lane", l.Name(),
		"num_txs_added", len(txsToInclude),
		"num_txs_removed", len(txsToRemove),
	)

	return next(ctx, proposal)
}

// ProcessLane verifies that the transactions included in the block proposal are valid respecting
// the verification logic of the lane (processLaneHandler). If any of the transactions are invalid,
// we return an error. If all of the transactions are valid, we return the updated proposal.
func (l *BaseLane) ProcessLane(
	ctx sdk.Context,
	proposal proposals.Proposal,
	txs [][]byte,
	next block.ProcessLanesHandler,
) (proposals.Proposal, error) {
	// Assume that this lane is processing sdk.Tx's and decode the transactions.
	decodedTxs, err := utils.GetDecodedTxs(l.TxDecoder(), txs)
	if err != nil {
		l.Logger().Error(
			"failed to decode transactions",
			"lane", l.Name(),
			"err", err,
		)

		return proposal, err
	}

	// Verify the transactions that belong to this lane according to the verification logic of the lane.
	if err := l.processLaneHandler(ctx, decodedTxs); err != nil {
		return proposal, err
	}

	// Optimistically update the proposal with the partial proposal.
	limit := proposal.GetLaneLimits(l.cfg.MaxBlockSpace)
	if err := proposal.UpdateProposal(l.Name(), decodedTxs, limit); err != nil {
		l.Logger().Error(
			"failed to update proposal",
			"lane", l.Name(),
			"err", err,
			"num_txs_to_verify", len(decodedTxs),
			"max_lane_size", limit.MaxTxBytes,
			"max_lane_gas_limit", limit.MaxGasLimit,
			"max_block_size", l.cfg.MaxBlockSpace,
		)

		return proposal, err
	}

	l.Logger().Info(
		"lane processed",
		"lane", l.Name(),
		"num_txs_verified", len(decodedTxs),
	)

	return next(ctx, proposal)
}

// AnteVerifyTx verifies that the transaction is valid respecting the ante verification logic of
// of the antehandler chain.
func (l *BaseLane) AnteVerifyTx(ctx sdk.Context, tx sdk.Tx, simulate bool) (sdk.Context, error) {
	if l.cfg.AnteHandler != nil {
		return l.cfg.AnteHandler(ctx, tx, simulate)
	}

	return ctx, nil
}
