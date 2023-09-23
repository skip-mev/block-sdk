package abci

import (
	"fmt"

	"cosmossdk.io/log"
	abci "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/skip-mev/block-sdk/block"
	"github.com/skip-mev/block-sdk/block/proposals"
	"github.com/skip-mev/block-sdk/block/proposals/types"
	"github.com/skip-mev/block-sdk/lanes/terminator"
)

const (
	// MaxUint64 is the maximum value of a uint64.
	MaxUint64 = 1<<64 - 1
)

type (
	// ProposalHandler is a wrapper around the ABCI++ PrepareProposal and ProcessProposal
	// handlers.
	ProposalHandler struct {
		logger              log.Logger
		txDecoder           sdk.TxDecoder
		txEncoder           sdk.TxEncoder
		prepareLanesHandler block.PrepareLanesHandler
		processLanesHandler block.ProcessLanesHandler
		mempool             block.Mempool
	}
)

// NewProposalHandler returns a new abci++ proposal handler. This proposal handler will
// iteratively call each of the lanes in the chain to prepare and process the proposal.
func NewProposalHandler(
	logger log.Logger,
	txDecoder sdk.TxDecoder,
	txEncoder sdk.TxEncoder,
	mempool block.Mempool,
) *ProposalHandler {
	return &ProposalHandler{
		logger:              logger,
		txDecoder:           txDecoder,
		txEncoder:           txEncoder,
		prepareLanesHandler: ChainPrepareLanes(mempool.Registry()...),
		processLanesHandler: ChainProcessLanes(mempool.Registry()...),
		mempool:             mempool,
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

		h.logger.Info("mempool distribution before proposal creation", "distribution", h.mempool.GetTxDistribution())

		blockParams := ctx.ConsensusParams().Block

		// If the max gas is set to 0, then the max gas limit for the block can be infinite.
		// Otherwise we use the max gas limit casted as a uint64 which is how gas limits are
		// extracted from sdk.Tx's.
		var maxGasLimit uint64
		if maxGas := blockParams.MaxGas; maxGas > 0 {
			maxGasLimit = uint64(maxGas)
		} else {
			maxGasLimit = MaxUint64
		}

		proposal, err := h.prepareLanesHandler(
			ctx,
			proposals.NewProposal(
				h.txEncoder,
				blockParams.MaxBytes,
				maxGasLimit,
			),
		)
		if err != nil {
			h.logger.Error("failed to prepare proposal", "err", err)
			return &abci.ResponsePrepareProposal{Txs: make([][]byte, 0)}, err
		}

		metaData := proposal.GetMetaData()

		h.logger.Info(
			"prepared proposal",
			"num_txs", metaData.NumTxs,
			"total_tx_bytes", metaData.TotalTxBytes,
			"max_tx_bytes", blockParams.MaxBytes,
			"total_gas_limit", metaData.TotalGasLimit,
			"max_gas_limit", maxGasLimit,
			"height", req.Height,
		)

		h.logger.Info("mempool distribution after proposal creation", "distribution", h.mempool.GetTxDistribution())

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

		// Conduct basic sanity checks on the proposal metadata.
		txsByLane, err := h.ValidateProposalBasic(req.Txs)
		if err != nil {
			h.logger.Error("failed to validate proposal", "err", err)
			return &abci.ResponseProcessProposal{Status: abci.ResponseProcessProposal_REJECT}, err
		}

		// Verify the proposal using the verification logic from each lane.
		if _, err := h.processLanesHandler(ctx, txsByLane); err != nil {
			h.logger.Error("failed to validate the proposal", "err", err)
			return &abci.ResponseProcessProposal{Status: abci.ResponseProcessProposal_REJECT}, err
		}

		h.logger.Info("validated proposal", "num_txs", len(txsByLane))

		return &abci.ResponseProcessProposal{Status: abci.ResponseProcessProposal_ACCEPT}, nil
	}
}

// ValidateProposalBasic validates the proposal against the basic invariants that are required
// for the proposal to be valid. This includes:
//  1. The proposal must contain the proposal metadata.
//  2. The proposal metadata must be valid.
//  3. The partial proposals must consume less than the maximum number of bytes allowed.
//  4. The partial proposals must consume less than the maximum gas limit allowed.
func (h *ProposalHandler) ValidateProposalBasic(proposal [][]byte) ([][]sdk.Tx, error) {
	// If the proposal is empty, then the metadata was not included.
	if len(proposal) == 0 {
		return nil, fmt.Errorf("proposal does not contain proposal metadata")
	}

	metaDataBz, txs := proposal[0], proposal[1:]

	// Retrieve the metadata from the proposal.
	var metaData types.ProposalMetaData
	if err := metaData.Unmarshal(metaDataBz); err != nil {
		return nil, fmt.Errorf("failed to unmarshal proposal metadata: %w", err)
	}

	lanes := h.mempool.Registry()
	txsByLane := make([][]sdk.Tx, len(lanes))
	for _, lane := range lanes {
		laneMetaData, ok := metaData.Lanes[lane.Name()]
		if !ok {
			txsByLane = append(txsByLane, nil)
			continue
		}

		if laneMetaData.NumTxs > uint64(len(txs)) {
			return nil, fmt.Errorf(
				"proposal metadata contains invalid number of transactions for lane %s; got %d, expected %d",
				lane.Name(),
				len(txs),
				laneMetaData.NumTxs,
			)
		}

		// Extract the transactions for the lane from the proposal.
		laneTxs := txs[:laneMetaData.NumTxs]

		// Validate the partial proposal against the basic invariants.
		decodedTxs, err := h.ValidatePartialProposalBasic(lane, laneTxs)
		if err != nil {
			return nil, fmt.Errorf("failed to validate partial proposal for lane %s: %w", lane.Name(), err)
		}

		// Remove the transactions for the lane from the proposal.
		txs = txs[laneMetaData.NumTxs:]

		// Add the decoded transactions to the list of transactions for the lane.
		txsByLane = append(txsByLane, decodedTxs)
	}

	// If there are any transactions remaining in the proposal, then the proposal is invalid.
	if len(txs) > 0 {
		return nil, fmt.Errorf("proposal contains invalid number of transactions")
	}

	return txsByLane, nil
}

func (h *ProposalHandler) ValidatePartialProposalBasic(lane block.Lane, proposal [][]byte) ([]sdk.Tx, error) {
	panic("implement me")
}

// ChainPrepareLanes chains together the proposal preparation logic from each lane
// into a single function. The first lane in the chain is the first lane to be prepared and
// the last lane in the chain is the last lane to be prepared.
//
// In the case where any of the lanes fail to prepare the partial proposal, the lane that failed
// will be skipped and the next lane in the chain will be called to prepare the proposal.
func ChainPrepareLanes(chain ...block.Lane) block.PrepareLanesHandler {
	if len(chain) == 0 {
		return nil
	}

	// Handle non-terminated decorators chain
	if (chain[len(chain)-1] != terminator.Terminator{}) {
		chain = append(chain, terminator.Terminator{})
	}

	return func(ctx sdk.Context, partialProposal proposals.Proposal) (finalProposal proposals.Proposal, err error) {
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

				if len(chain) <= 2 {
					// If there are only two lanes remaining, then the first lane in the chain
					// is the lane that failed to prepare the partial proposal and the second lane in the
					// chain is the terminator lane. We return the proposal as is.
					finalProposal, err = partialProposal, nil
				} else {
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
		metaData := partialProposal.GetMetaData()
		limit := proposals.GetLaneLimits(
			partialProposal.GetMaxTxBytes(), metaData.TotalTxBytes,
			partialProposal.GetMaxGasLimit(), metaData.TotalGasLimit,
			lane.GetMaxBlockSpace(),
		)

		return lane.PrepareLane(
			cacheCtx,
			partialProposal,
			limit,
			ChainPrepareLanes(chain[1:]...),
		)
	}
}

// ChainProcessLanes chains together the proposal verification logic from each lane
// into a single function. The first lane in the chain is the first lane to be verified and
// the last lane in the chain is the last lane to be verified.
func ChainProcessLanes(chain ...block.Lane) block.ProcessLanesHandler {
	if len(chain) == 0 {
		return nil
	}

	// Handle non-terminated decorators chain
	if (chain[len(chain)-1] != terminator.Terminator{}) {
		chain = append(chain, terminator.Terminator{})
	}

	return func(ctx sdk.Context, txsByLane [][]sdk.Tx) (sdk.Context, error) {
		// Determine the lane which is going to include the corresponding transactions
		// in the block.
		txs := txsByLane[len(txsByLane)-len(chain)]

		chain[0].Logger().Info(
			"processing lane",
			"lane", chain[0].Name(),
			"num_txs", len(txs),
		)

		return chain[0].ProcessLane(ctx, txs, ChainProcessLanes(chain[1:]...))
	}
}
