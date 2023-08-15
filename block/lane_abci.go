package blockbuster

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/skip-mev/pob/blockbuster/utils"
)

// PrepareLane will prepare a partial proposal for the lane. It will select transactions from the
// lane respecting the selection logic of the prepareLaneHandler. It will then update the partial
// proposal with the selected transactions. If the proposal is unable to be updated, we return an
// error. The proposal will only be modified if it passes all of the invarient checks.
func (l *LaneConstructor) PrepareLane(
	ctx sdk.Context,
	proposal BlockProposal,
	maxTxBytes int64,
	next PrepareLanesHandler,
) (BlockProposal, error) {
	txs, txsToRemove, err := l.prepareLaneHandler(ctx, proposal, maxTxBytes)
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

	// Update the proposal with the selected transactions.
	if err := proposal.UpdateProposal(l, txs); err != nil {
		return proposal, err
	}

	return next(ctx, proposal)
}

// CheckOrder checks that the ordering logic of the lane is respected given the set of transactions
// in the block proposal. If the ordering logic is not respected, we return an error.
func (l *LaneConstructor) CheckOrder(ctx sdk.Context, txs []sdk.Tx) error {
	return l.checkOrderHandler(ctx, txs)
}

// ProcessLane verifies that the transactions included in the block proposal are valid respecting
// the verification logic of the lane (processLaneHandler). If the transactions are valid, we
// return the transactions that do not belong to this lane to the next lane. If the transactions
// are invalid, we return an error.
func (l *LaneConstructor) ProcessLane(ctx sdk.Context, txs []sdk.Tx, next ProcessLanesHandler) (sdk.Context, error) {
	remainingTxs, err := l.processLaneHandler(ctx, txs)
	if err != nil {
		return ctx, err
	}

	return next(ctx, remainingTxs)
}

// AnteVerifyTx verifies that the transaction is valid respecting the ante verification logic of
// of the antehandler chain.
func (l *LaneConstructor) AnteVerifyTx(ctx sdk.Context, tx sdk.Tx, simulate bool) (sdk.Context, error) {
	if l.cfg.AnteHandler != nil {
		return l.cfg.AnteHandler(ctx, tx, simulate)
	}

	return ctx, nil
}
