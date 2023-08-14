package abci

import (
	"fmt"

	"cosmossdk.io/log"
	abci "github.com/cometbft/cometbft/abci/types"
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
		txDecoder           sdk.TxDecoder
		prepareLanesHandler blockbuster.PrepareLanesHandler
		processLanesHandler blockbuster.ProcessLanesHandler
	}
)

// NewProposalHandler returns a new abci++ proposal handler. This proposal handler will
// iteratively call each of the lanes in the chain to prepare and process a proposal.
func NewProposalHandler(logger log.Logger, txDecoder sdk.TxDecoder, lanes []blockbuster.Lane) *ProposalHandler {
	return &ProposalHandler{
		logger:              logger,
		txDecoder:           txDecoder,
		prepareLanesHandler: ChainPrepareLanes(lanes...),
		processLanesHandler: ChainProcessLanes(lanes...),
	}
}

// PrepareProposalHandler prepares the proposal by selecting transactions from each lane
// according to each lane's selection logic. We select transactions in a greedy fashion. Note that
// each lane has an boundary on the number of bytes that can be included in the proposal. By default,
// the default lane will not have a boundary on the number of bytes that can be included in the proposal and
// will include all valid transactions in the proposal (up to MaxTxBytes).
func (h *ProposalHandler) PrepareProposalHandler() sdk.PrepareProposalHandler {
	return func(ctx sdk.Context, req *abci.RequestPrepareProposal) (resp *abci.ResponsePrepareProposal, err error) {
		// In the case where there is a panic, we recover here and return an empty proposal.
		defer func() {
			if err := recover(); err != nil {
				h.logger.Error("failed to prepare proposal", "err", err)
				resp = &abci.ResponsePrepareProposal{Txs: make([][]byte, 0)}
			}
		}()

		proposal, err := h.prepareLanesHandler(ctx, blockbuster.NewProposal(req.MaxTxBytes))
		if err != nil {
			h.logger.Error("failed to prepare proposal", "err", err)
			return &abci.ResponsePrepareProposal{Txs: make([][]byte, 0)}, err
		}

		h.logger.Info(
			"prepared proposal",
			"num_txs", proposal.GetNumTxs(),
			"total_tx_bytes", proposal.GetTotalTxBytes(),
			"height", req.Height,
		)

		return &abci.ResponsePrepareProposal{
			Txs: proposal.GetProposal(),
		}, nil
	}
}

// ProcessProposalHandler processes the proposal by verifying all transactions in the proposal
// according to each lane's verification logic. We verify proposals in a greedy fashion.
// If a lane's portion of the proposal is invalid, we reject the proposal. After a lane's portion
// of the proposal is verified, we pass the remaining transactions to the next lane in the chain.
func (h *ProposalHandler) ProcessProposalHandler() sdk.ProcessProposalHandler {
	return func(ctx sdk.Context, req *abci.RequestProcessProposal) (resp *abci.ResponseProcessProposal, err error) {
		// In the case where any of the lanes panic, we recover here and return a reject status.
		defer func() {
			if rec := recover(); rec != nil {
				h.logger.Error("failed to process proposal", "recover_err", rec)

				resp = &abci.ResponseProcessProposal{Status: abci.ResponseProcessProposal_REJECT}
				err = fmt.Errorf("failed to process proposal: %v", rec)
			}
		}()

		txs := req.Txs
		if len(txs) == 0 {
			h.logger.Info("accepted empty proposal")
			return &abci.ResponseProcessProposal{Status: abci.ResponseProcessProposal_ACCEPT}, nil
		}

		// Decode the transactions from the proposal.
		decodedTxs, err := utils.GetDecodedTxs(h.txDecoder, txs)
		if err != nil {
			h.logger.Error("failed to decode transactions", "err", err)
			return &abci.ResponseProcessProposal{Status: abci.ResponseProcessProposal_REJECT}, err
		}

		// Verify the proposal using the verification logic from each lane.
		if _, err := h.processLanesHandler(ctx, decodedTxs); err != nil {
			h.logger.Error("failed to validate the proposal", "err", err)
			return &abci.ResponseProcessProposal{Status: abci.ResponseProcessProposal_REJECT}, err
		}

		h.logger.Info("validated proposal", "num_txs", len(txs))

		return &abci.ResponseProcessProposal{Status: abci.ResponseProcessProposal_ACCEPT}, nil
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

	return func(ctx sdk.Context, partialProposal blockbuster.BlockProposal) (finalProposal blockbuster.BlockProposal, err error) {
		lane := chain[0]
		lane.Logger().Info("preparing lane", "lane", lane.Name())

		// Cache the context in the case where any of the lanes fail to prepare the proposal.
		cacheCtx, write := ctx.CacheContext()

		// We utilize a recover to handle any panics or errors that occur during the preparation
		// of a lane's transactions. This defer will first check if there was a panic or error
		// thrown from the lane's preparation logic. If there was, we log the error, skip the lane,
		// and call the next lane in the chain to the prepare the proposal.
		defer func() {
			if rec := recover(); rec != nil || err != nil {
				lane.Logger().Error("failed to prepare lane", "lane", lane.Name(), "err", err, "recover_error", rec)
				lane.Logger().Info("skipping lane", "lane", lane.Name())

				lanesRemaining := len(chain)
				switch {
				case lanesRemaining <= 2:
					// If there are only two lanes remaining, then the first lane in the chain
					// is the lane that failed to prepare the partial proposal and the second lane in the
					// chain is the terminator lane. We return the proposal as is.
					finalProposal, err = partialProposal, nil
				default:
					// If there are more than two lanes remaining, then the first lane in the chain
					// is the lane that failed to prepare the proposal but the second lane in the
					// chain is not the terminator lane so there could potentially be more transactions
					// added to the proposal
					finalProposal, err = ChainPrepareLanes(chain[1:]...)(ctx, partialProposal)
				}
			} else {
				// Write the cache to the context since we know that the lane successfully prepared
				// the partial proposal. State is written to in a backwards, cascading fashion. This means
				// that the final context will only be updated after all other lanes have successfully
				// prepared the partial proposal.
				write()
			}
		}()

		// Get the maximum number of bytes that can be included in the proposal for this lane.
		maxTxBytesForLane := utils.GetMaxTxBytesForLane(
			partialProposal.GetMaxTxBytes(),
			partialProposal.GetTotalTxBytes(),
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

	return func(ctx sdk.Context, proposalTxs []sdk.Tx) (sdk.Context, error) {
		// Short circuit if there are no transactions to process.
		if len(proposalTxs) == 0 {
			return ctx, nil
		}

		chain[0].Logger().Info("processing lane", "lane", chain[0].Name())

		if err := chain[0].CheckOrder(ctx, proposalTxs); err != nil {
			chain[0].Logger().Error("failed to process lane", "lane", chain[0].Name(), "err", err)
			return ctx, err
		}

		return chain[0].ProcessLane(ctx, proposalTxs, ChainProcessLanes(chain[1:]...))
	}
}
