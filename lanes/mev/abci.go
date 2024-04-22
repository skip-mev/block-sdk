package mev

import (
	"bytes"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/skip-mev/block-sdk/v2/block/base"
	"github.com/skip-mev/block-sdk/v2/block/proposals"
)

// Implements the MEV lane's PrepareLaneHandler and ProcessLaneHandler.
type ProposalHandler struct {
	lane    *base.BaseLane
	factory Factory
}

// NewProposalHandler returns a new mev proposal handler.
func NewProposalHandler(lane *base.BaseLane, factory Factory) *ProposalHandler {
	return &ProposalHandler{
		lane:    lane,
		factory: factory,
	}
}

// PrepareLaneHandler will attempt to select the highest bid transaction that is valid
// and whose bundled transactions are valid and include them in the proposal. It
// will return no transactions if no valid bids are found. If any of the bids are invalid,
// it will return them and will only remove the bids and not the bundled transactions.
func (h *ProposalHandler) PrepareLaneHandler() base.PrepareLaneHandler {
	return func(ctx sdk.Context, proposal proposals.Proposal, limit proposals.LaneLimits) ([]sdk.Tx, []sdk.Tx, error) {
		// Define all of the info we need to select transactions for the partial proposal.
		var (
			txsToInclude []sdk.Tx
			txsToRemove  []sdk.Tx
		)

		// Attempt to select the highest bid transaction that is valid and whose
		// bundled transactions are valid.
		for iterator := h.lane.Select(ctx, nil); iterator != nil; iterator = iterator.Next() {
			bidTx := iterator.Tx()

			if !h.lane.Match(ctx, bidTx) {
				h.lane.Logger().Info("failed to select auction bid tx for lane; tx does not match lane")

				txsToRemove = append(txsToRemove, bidTx)
				continue
			}

			cacheCtx, write := ctx.CacheContext()

			bundle, err := h.VerifyBidBasic(cacheCtx, bidTx, proposal, limit)
			if err != nil {
				h.lane.Logger().Info(
					"failed to select auction bid tx for lane; tx is invalid",
					"err", err,
				)

				txsToRemove = append(txsToRemove, bidTx)
				continue
			}

			if err := h.VerifyBidTx(cacheCtx, bidTx, bundle); err != nil {
				h.lane.Logger().Info(
					"failed to select auction bid tx for lane; tx is invalid",
					"err", err,
				)

				txsToRemove = append(txsToRemove, bidTx)
				continue
			}

			// At this point, both the bid transaction itself and all the bundled
			// transactions are valid. So we select the bid transaction along with
			// all the bundled transactions. We also mark these transactions as seen and
			// update the total size selected thus far.
			txsToInclude = append(txsToInclude, bidTx)
			txsToInclude = append(txsToInclude, bundle...)

			// Write the cache context to the original context when we know we have a
			// valid bundle.
			write()

			break
		}

		return txsToInclude, txsToRemove, nil
	}
}

// ProcessLaneHandler will ensure that block proposals that include transactions from
// the mev lane are valid. In particular, the invariant checks that we perform are:
//  1. If the first transaction does not match the lane, no other MEV transactions
//     should be included in the proposal.
//  2. The bid transaction must be valid.
//  3. The bundled transactions must be valid.
//  4. The bundled transactions must match the transactions in the block proposal in the
//     same order they were defined in the bid transaction.
//  5. The bundled transactions must not be bid transactions.
func (h *ProposalHandler) ProcessLaneHandler() base.ProcessLaneHandler {
	return func(ctx sdk.Context, partialProposal []sdk.Tx) ([]sdk.Tx, []sdk.Tx, error) {
		if len(partialProposal) == 0 {
			return nil, nil, nil
		}

		bidTx := partialProposal[0]
		if !h.lane.Match(ctx, bidTx) {
			// If the transaction does not belong to this lane, we return the remaining transactions
			// iff there are no matches in the remaining transactions after this index.
			if len(partialProposal) > 1 {
				if err := h.lane.VerifyNoMatches(ctx, partialProposal[1:]); err != nil {
					return nil, nil, fmt.Errorf("failed to verify no matches: %w", err)
				}
			}

			return nil, partialProposal, nil
		}

		bidInfo, err := h.factory.GetAuctionBidInfo(bidTx)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to get bid info from auction bid tx for lane %s: %w", h.lane.Name(), err)
		}

		if bidInfo == nil {
			return nil, nil, fmt.Errorf("bid info is nil")
		}

		// Check that all bundled transactions were included.
		bundleSize := len(bidInfo.Transactions) + 1
		if bundleSize > len(partialProposal) {
			return nil, nil, fmt.Errorf(
				"expected %d transactions in lane %s but got %d",
				bundleSize,
				h.lane.Name(),
				len(partialProposal),
			)
		}

		// Ensure the transactions in the proposal match the bundled transactions in the bid transaction.
		bundle := partialProposal[1:bundleSize]
		for index, bundledTxBz := range bidInfo.Transactions {
			bundledTx, err := h.factory.WrapBundleTransaction(bundledTxBz)
			if err != nil {
				return nil, nil, fmt.Errorf("invalid bid tx; failed to decode bundled tx: %w", err)
			}

			expectedTxBz, err := h.lane.TxEncoder()(bundledTx)
			if err != nil {
				return nil, nil, fmt.Errorf("invalid bid tx; failed to encode bundled tx: %w", err)
			}

			actualTxBz, err := h.lane.TxEncoder()(bundle[index])
			if err != nil {
				return nil, nil, fmt.Errorf("invalid bid tx; failed to encode tx: %w", err)
			}

			// Verify that the bundled transaction matches the transaction in the block proposal.
			if !bytes.Equal(actualTxBz, expectedTxBz) {
				return nil, nil, fmt.Errorf("invalid bid tx; bundled tx does not match tx in block proposal")
			}
		}

		// Verify the top-level bid transaction.
		//
		// TODO: There is duplicate work being done in VerifyBidTx and here.
		if err := h.VerifyBidTx(ctx, bidTx, bundle); err != nil {
			return nil, nil, fmt.Errorf("invalid bid tx; failed to verify bid tx: %w", err)
		}

		return partialProposal[:bundleSize], partialProposal[bundleSize:], nil
	}
}

// VerifyBidBasic will verify that the bid transaction and all of its bundled
// transactions respect the basic invariants of the lane (e.g. size, gas limit).
func (h *ProposalHandler) VerifyBidBasic(
	ctx sdk.Context,
	bidTx sdk.Tx,
	proposal proposals.Proposal,
	limit proposals.LaneLimits,
) ([]sdk.Tx, error) {
	// Verify the transaction is a bid transaction.
	bidInfo, err := h.factory.GetAuctionBidInfo(bidTx)
	if err != nil {
		return nil, fmt.Errorf("failed to get bid info from auction bid tx for lane %s: %w", h.lane.Name(), err)
	}

	if bidInfo == nil {
		return nil, fmt.Errorf("bid info is nil")
	}

	txInfo, err := h.lane.GetTxInfo(ctx, bidTx)
	if err != nil {
		return nil, fmt.Errorf("err retrieving transaction info: %s", err)
	}

	// This should never happen, but we check just in case.
	if proposal.Contains(txInfo.Hash) {
		return nil, fmt.Errorf("invalid bid tx; bid tx is already in the proposal")
	}

	totalSize := txInfo.Size
	totalGasLimit := txInfo.GasLimit
	bundle := make([]sdk.Tx, len(bidInfo.Transactions))

	// Verify size and gas limit of the bundled transactions.
	for index, bundledTxBz := range bidInfo.Transactions {
		bundledTx, err := h.factory.WrapBundleTransaction(bundledTxBz)
		if err != nil {
			return nil, fmt.Errorf("invalid bid tx; failed to decode bundled tx: %w", err)
		}

		bundledTxInfo, err := h.lane.GetTxInfo(ctx, bundledTx)
		if err != nil {
			return nil, fmt.Errorf("err retrieving transaction info: %s", err)
		}

		if proposal.Contains(bundledTxInfo.Hash) {
			return nil, fmt.Errorf("invalid bid tx; bundled tx is already in the proposal")
		}

		totalSize += bundledTxInfo.Size
		totalGasLimit += bundledTxInfo.GasLimit
		bundle[index] = bundledTx
	}

	if totalSize > limit.MaxTxBytes {
		return nil, fmt.Errorf(
			"partial proposal is too large: %d > %d",
			totalSize,
			limit.MaxTxBytes,
		)
	}

	if totalGasLimit > limit.MaxGasLimit {
		return nil, fmt.Errorf(
			"partial proposal consumes too much gas: %d > %d",
			totalGasLimit,
			limit.MaxGasLimit,
		)
	}

	return bundle, nil
}

// VerifyBidTx will verify that the bid transaction and all of its bundled
// transactions are valid.
func (h *ProposalHandler) VerifyBidTx(ctx sdk.Context, bidTx sdk.Tx, bundle []sdk.Tx) error {
	bidInfo, err := h.factory.GetAuctionBidInfo(bidTx)
	if err != nil {
		return fmt.Errorf("failed to get bid info from auction bid tx for lane %s: %w", h.lane.Name(), err)
	}

	if bidInfo == nil {
		return fmt.Errorf("bid info is nil")
	}

	// verify the top-level bid transaction
	if err = h.lane.VerifyTx(ctx, bidTx, false); err != nil {
		return fmt.Errorf("invalid bid tx; failed to execute ante handler: %w", err)
	}

	// verify all of the bundled transactions
	for _, bundledTx := range bundle {
		if h.lane.Match(ctx, bundledTx) {
			return fmt.Errorf("invalid bid tx; bundled tx is another bid transaction")
		}

		if err = h.lane.VerifyTx(ctx, bundledTx, false); err != nil {
			return fmt.Errorf("invalid bid tx; failed to execute bundled transaction: %w", err)
		}
	}

	return nil
}
