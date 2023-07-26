package abci

import (
	"cosmossdk.io/log"
	"cosmossdk.io/math"
	cometabci "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkmempool "github.com/cosmos/cosmos-sdk/types/mempool"
	"github.com/skip-mev/pob/blockbuster"
	"github.com/skip-mev/pob/blockbuster/abci"
	"github.com/skip-mev/pob/blockbuster/lanes/auction"
	"github.com/skip-mev/pob/blockbuster/utils"
)

const (
	// NumInjectedTxs is the minimum number of transactions that were injected into
	// the proposal but are not actual transactions. In this case, the auction
	// info is injected into the proposal but should be ignored by the application.ß
	NumInjectedTxs = 1

	// AuctionInfoIndex is the index of the auction info in the proposal.
	AuctionInfoIndex = 0
)

type (
	// TOBLaneProposal is the interface that defines all of the dependencies that
	// are required to interact with the top of block lane.
	TOBLaneProposal interface {
		sdkmempool.Mempool

		// Factory defines the API/functionality which is responsible for determining
		// if a transaction is a bid transaction and how to extract relevant
		// information from the transaction (bid, timeout, bidder, etc.).
		auction.Factory

		// VerifyTx is utilized to verify a bid transaction according to the preferences
		// of the top of block lane.
		VerifyTx(ctx sdk.Context, tx sdk.Tx) error

		// GetMaxBlockSpace returns the maximum block space that can be used by the top of
		// block lane as a percentage of the total block space.
		GetMaxBlockSpace() math.LegacyDec

		// Logger returns the logger for the top of block lane.
		Logger() log.Logger

		// Name returns the name of the top of block lane.
		Name() string
	}

	// ProposalHandler contains the functionality and handlers required to\
	// process, validate and build blocks.
	ProposalHandler struct {
		logger    log.Logger
		txEncoder sdk.TxEncoder
		txDecoder sdk.TxDecoder

		// prepareLanesHandler is responsible for preparing the proposal by selecting
		// transactions from each lane according to each lane's selection logic.
		prepareLanesHandler blockbuster.PrepareLanesHandler

		// processLanesHandler is responsible for verifying that the proposal is valid
		// according to each lane's verification logic.
		processLanesHandler blockbuster.ProcessLanesHandler

		// tobLane is the top of block lane which is utilized to verify transactions that
		// should be included in the top of block.
		tobLane TOBLaneProposal

		// validateVoteExtensionsFn is the function responsible for validating vote extensions.
		validateVoteExtensionsFn ValidateVoteExtensionsFn
	}
)

// NewProposalHandler returns a ProposalHandler that contains the functionality and handlers
// required to process, validate and build blocks.
func NewProposalHandler(
	lanes []blockbuster.Lane,
	tobLane TOBLaneProposal,
	logger log.Logger,
	txEncoder sdk.TxEncoder,

	txDecoder sdk.TxDecoder,
	validateVeFN ValidateVoteExtensionsFn,
) *ProposalHandler {
	return &ProposalHandler{
		// We prepare lanes skipping the first lane because the first lane is the top of block lane.
		prepareLanesHandler:      abci.ChainPrepareLanes(lanes[1:]...),
		processLanesHandler:      abci.ChainProcessLanes(lanes...),
		tobLane:                  tobLane,
		logger:                   logger,
		txEncoder:                txEncoder,
		txDecoder:                txDecoder,
		validateVoteExtensionsFn: validateVeFN,
	}
}

// PrepareProposalHandler returns the PrepareProposal ABCI handler that performs
// top-of-block auctioning and general block proposal construction. This handler
// will first attempt to construct the top of the block by utilizing the vote
// extensions from the previous height. If the vote extensions are not available,
// then no top of block auction is performed. After this, the rest of the proposal
// will be constructed by selecting transactions from each lane according to each
// lane's selection logic.
func (h *ProposalHandler) PrepareProposalHandler() sdk.PrepareProposalHandler {
	return func(ctx sdk.Context, req *cometabci.RequestPrepareProposal) (*cometabci.ResponsePrepareProposal, error) {
		partialProposal := blockbuster.NewProposal(req.MaxTxBytes)
		voteExtensionsEnabled := h.VoteExtensionsEnabled(ctx)

		h.logger.Info(
			"preparing proposal",
			"height", req.Height,
			"vote_extensions_enabled", voteExtensionsEnabled,
		)

		if voteExtensionsEnabled {
			// Build the top of block portion of the proposal given the vote extensions
			// from the previous height.
			partialProposal = h.BuildTOB(ctx, req.LocalLastCommit, req.MaxTxBytes)

			h.logger.Info(
				"built top of block",
				"num_txs", partialProposal.GetNumTxs(),
				"size", partialProposal.GetTotalTxBytes(),
			)

			// If information is unable to be marshaled, we return an empty proposal. This will
			// cause another proposal to be generated after it is rejected in ProcessProposal.
			lastCommitInfo, err := req.LocalLastCommit.Marshal()
			if err != nil {
				h.logger.Error("failed to marshal last commit info", "err", err)
				return &cometabci.ResponsePrepareProposal{Txs: nil}, err
			}

			auctionInfo := &AuctionInfo{
				ExtendedCommitInfo: lastCommitInfo,
				MaxTxBytes:         req.MaxTxBytes,
				NumTxs:             uint64(partialProposal.GetNumTxs()),
			}

			// Add the auction info and top of block transactions into the proposal.
			auctionInfoBz, err := auctionInfo.Marshal()
			if err != nil {
				h.logger.Error("failed to marshal auction info", "err", err)
				return &cometabci.ResponsePrepareProposal{Txs: nil}, err
			}

			partialProposal.AddVoteExtension(auctionInfoBz)
		}

		// Prepare the proposal by selecting transactions from each lane according to
		// each lane's selection logic.
		finalProposal, err := h.prepareLanesHandler(ctx, partialProposal)
		if err != nil {
			h.logger.Error("failed to prepare proposal", "err", err)
			return &cometabci.ResponsePrepareProposal{Txs: nil}, err
		}

		h.logger.Info(
			"prepared proposal",
			"num_txs", finalProposal.GetNumTxs(),
			"size", finalProposal.GetTotalTxBytes(),
		)

		return &cometabci.ResponsePrepareProposal{Txs: finalProposal.GetProposal()}, err
	}
}

// ProcessProposalHandler returns the ProcessProposal ABCI handler that performs
// block proposal verification. This handler will first attempt to verify the top
// of block transactions by utilizing the vote extensions from the previous height.
// If the vote extensions are not available, then no top of block verification is done.
// After this, the rest of the proposal will be verified according to each lane's
// verification logic.
func (h *ProposalHandler) ProcessProposalHandler() sdk.ProcessProposalHandler {
	return func(ctx sdk.Context, req *cometabci.RequestProcessProposal) (*cometabci.ResponseProcessProposal, error) {
		txs := req.Txs
		voteExtensionsEnabled := h.VoteExtensionsEnabled(ctx)

		h.logger.Info(
			"processing proposal",
			"height", req.Height,
			"vote_extensions_enabled", voteExtensionsEnabled,
			"num_txs", len(req.Txs),
		)

		// If vote extensions have been enabled, verify that the same top of block transactions can be
		// built from the vote extensions included in the proposal. Otherwise verify that the proposal
		// is valid according to each lane's verification logic.
		if voteExtensionsEnabled {
			auctionInfo, err := h.VerifyTOB(ctx, txs)
			if err != nil {
				h.logger.Error("failed to verify top of block transactions", "err", err)
				return &cometabci.ResponseProcessProposal{Status: cometabci.ResponseProcessProposal_REJECT}, err
			}

			h.logger.Info(
				"verified top of block",
				"num_txs", auctionInfo.NumTxs,
			)

			txs = req.Txs[NumInjectedTxs:]
		}

		decodedTxs, err := utils.GetDecodedTxs(h.txDecoder, txs)
		if err != nil {
			h.logger.Error("failed to decode transactions", "err", err)
			return &cometabci.ResponseProcessProposal{Status: cometabci.ResponseProcessProposal_REJECT}, err
		}

		// Verify that the rest of the proposal is valid according to each lane's verification logic.
		if _, err = h.processLanesHandler(ctx, decodedTxs); err != nil {
			h.logger.Error("failed to process proposal", "err", err)
			return &cometabci.ResponseProcessProposal{Status: cometabci.ResponseProcessProposal_REJECT}, err
		}

		return &cometabci.ResponseProcessProposal{Status: cometabci.ResponseProcessProposal_ACCEPT}, nil
	}
}

// VoteExtensionsEnabled determines if vote extensions are enabled for the current block.
func (h *ProposalHandler) VoteExtensionsEnabled(ctx sdk.Context) bool {
	cp := ctx.ConsensusParams()
	if cp.Abci == nil || cp.Abci.VoteExtensionsEnableHeight == 0 {
		return false
	}

	// We do a > here because the vote extensions are enabled at block height H
	// but will only be used at block height H+1.
	return ctx.BlockHeight() > cp.Abci.VoteExtensionsEnableHeight
}
