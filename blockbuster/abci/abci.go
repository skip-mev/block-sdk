package abci

import (
	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cometbft/cometbft/libs/log"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/skip-mev/pob/blockbuster"
	"github.com/skip-mev/pob/blockbuster/lanes/terminator"
	"github.com/skip-mev/pob/blockbuster/utils"
)

type (
	// ProposalHandler is a wrapper around the ABCI++ PrepareProposal and ProcessProposal
	// handlers.
	ProposalHandler struct {
		logger              log.Logger
		prepareLanesHandler blockbuster.PrepareLanesHandler
		processLanesHandler blockbuster.ProcessLanesHandler
	}
)

// NewProposalHandler returns a new abci++ proposal handler.
func NewProposalHandler(logger log.Logger, mempool blockbuster.Mempool) *ProposalHandler {
	return &ProposalHandler{
		logger:              logger,
		prepareLanesHandler: ChainPrepareLanes(mempool.Registry()...),
		processLanesHandler: ChainProcessLanes(mempool.Registry()...),
	}
}

// PrepareProposalHandler prepares the proposal by selecting transactions from each lane
// according to each lane's selection logic. We select transactions in a greedy fashion. Note that
// each lane has an boundary on the number of bytes that can be included in the proposal. By default,
// the default lane will not have a boundary on the number of bytes that can be included in the proposal and
// will include all valid transactions in the proposal (up to MaxTxBytes).
func (h *ProposalHandler) PrepareProposalHandler() sdk.PrepareProposalHandler {
	return func(ctx sdk.Context, req abci.RequestPrepareProposal) (resp abci.ResponsePrepareProposal) {
		// In the case where there is a panic, we recover here and return an empty proposal.
		defer func() {
			if err := recover(); err != nil {
				h.logger.Error("failed to prepare proposal", "err", err)
				resp = abci.ResponsePrepareProposal{Txs: make([][]byte, 0)}
			}
		}()

		proposal := h.prepareLanesHandler(ctx, blockbuster.NewProposal(req.MaxTxBytes))

		resp = abci.ResponsePrepareProposal{
			Txs: proposal.Txs,
		}

		return
	}
}

// ProcessProposalHandler processes the proposal by verifying all transactions in the proposal
// according to each lane's verification logic. We verify proposals in a greedy fashion.
// If a lane's portion of the proposal is invalid, we reject the proposal. After a lane's portion
// of the proposal is verified, we pass the remaining transactions to the next lane in the chain.
func (h *ProposalHandler) ProcessProposalHandler() sdk.ProcessProposalHandler {
	return func(ctx sdk.Context, req abci.RequestProcessProposal) (resp abci.ResponseProcessProposal) {
		// In the case where any of the lanes panic, we recover here and return a reject status.
		defer func() {
			if err := recover(); err != nil {
				h.logger.Error("failed to process proposal", "err", err)
				resp = abci.ResponseProcessProposal{Status: abci.ResponseProcessProposal_REJECT}
			}
		}()

		// Verify the proposal using the verification logic from each lane.
		if _, err := h.processLanesHandler(ctx, req.Txs); err != nil {
			h.logger.Error("failed to validate the proposal", "err", err)
			return abci.ResponseProcessProposal{Status: abci.ResponseProcessProposal_REJECT}
		}

		return abci.ResponseProcessProposal{Status: abci.ResponseProcessProposal_ACCEPT}
	}
}

// ChainPrepareLanes chains together the proposal preparation logic from each lane
// into a single function. The first lane in the chain is the first lane to be prepared and
// the last lane in the chain is the last lane to be prepared.
//
// In the case where any of the lanes fail to prepare the partial proposal, the lane that failed
// will be skipped and the next lane in the chain will be called to prepare the proposal.
func ChainPrepareLanes(chain ...blockbuster.Lane) blockbuster.PrepareLanesHandler {
	if len(chain) == 0 {
		return nil
	}

	// Handle non-terminated decorators chain
	if (chain[len(chain)-1] != terminator.Terminator{}) {
		chain = append(chain, terminator.Terminator{})
	}

	return func(ctx sdk.Context, partialProposal *blockbuster.Proposal) (finalProposal *blockbuster.Proposal) {
		lane := chain[0]
		lane.Logger().Info("preparing lane", "lane", lane.Name())

		// Cache the context in the case where any of the lanes fail to prepare the proposal.
		cacheCtx, write := ctx.CacheContext()

		defer func() {
			if err := recover(); err != nil {
				lane.Logger().Error("failed to prepare lane", "lane", lane.Name(), "err", err)

				lanesRemaining := len(chain)
				switch {
				case lanesRemaining <= 2:
					// If there are only two lanes remaining, then the first lane in the chain
					// is the lane that failed to prepare the partial proposal and the second lane in the
					// chain is the terminator lane. We return the proposal as is.
					finalProposal = partialProposal
				default:
					// If there are more than two lanes remaining, then the first lane in the chain
					// is the lane that failed to prepare the proposal but the second lane in the
					// chain is not the terminator lane so there could potentially be more transactions
					// added to the proposal
					maxTxBytesForLane := utils.GetMaxTxBytesForLane(
						partialProposal,
						chain[1].GetMaxBlockSpace(),
					)

					finalProposal = chain[1].PrepareLane(
						ctx,
						partialProposal,
						maxTxBytesForLane,
						ChainPrepareLanes(chain[2:]...),
					)
				}
			} else {
				// Write the cache to the context since we know that the lane successfully prepared
				// the partial proposal.
				write()

				lane.Logger().Info("prepared lane", "lane", lane.Name())
			}
		}()

		// Get the maximum number of bytes that can be included in the proposal for this lane.
		maxTxBytesForLane := utils.GetMaxTxBytesForLane(
			partialProposal,
			lane.GetMaxBlockSpace(),
		)

		return lane.PrepareLane(
			cacheCtx,
			partialProposal,
			maxTxBytesForLane,
			ChainPrepareLanes(chain[1:]...),
		)
	}
}

// ChainProcessLanes chains together the proposal verification logic from each lane
// into a single function. The first lane in the chain is the first lane to be verified and
// the last lane in the chain is the last lane to be verified.
func ChainProcessLanes(chain ...blockbuster.Lane) blockbuster.ProcessLanesHandler {
	if len(chain) == 0 {
		return nil
	}

	// Handle non-terminated decorators chain
	if (chain[len(chain)-1] != terminator.Terminator{}) {
		chain = append(chain, terminator.Terminator{})
	}

	return func(ctx sdk.Context, proposalTxs [][]byte) (sdk.Context, error) {
		// Short circuit if there are no transactions to process.
		if len(proposalTxs) == 0 {
			return ctx, nil
		}

		chain[0].Logger().Info("processing lane", "lane", chain[0].Name())
		if err := chain[0].ProcessLaneBasic(proposalTxs); err != nil {
			return ctx, err
		}

		return chain[0].ProcessLane(ctx, proposalTxs, ChainProcessLanes(chain[1:]...))
	}
}
