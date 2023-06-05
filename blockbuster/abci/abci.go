package abci

import (
	"crypto/sha256"
	"encoding/hex"

	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cometbft/cometbft/libs/log"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/skip-mev/pob/blockbuster"
	"github.com/skip-mev/pob/blockbuster/lanes/terminator"
)

type (
	// ProposalHandler is a wrapper around the ABCI++ PrepareProposal and ProcessProposal
	// handlers.
	ProposalHandler struct {
		logger              log.Logger
		mempool             blockbuster.Mempool
		txEncoder           sdk.TxEncoder
		processLanesHandler blockbuster.ProcessLanesHandler
	}
)

// NewProposalHandler returns a new ProposalHandler.
func NewProposalHandler(logger log.Logger, mempool blockbuster.Mempool, txEncoder sdk.TxEncoder) *ProposalHandler {
	return &ProposalHandler{
		logger:              logger,
		mempool:             mempool,
		txEncoder:           txEncoder,
		processLanesHandler: ChainProcessLanes(mempool.Registry()...),
	}
}

// ChainProcessLane chains together the proposal verification logic from each lane
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
		return chain[0].ProcessLane(ctx, proposalTxs, ChainProcessLanes(chain[1:]...))
	}
}

func (h *ProposalHandler) PrepareProposalHandler() sdk.PrepareProposalHandler {
	return func(ctx sdk.Context, req abci.RequestPrepareProposal) abci.ResponsePrepareProposal {
		var (
			selectedTxs  = make(map[string][]byte)
			totalTxBytes int64
		)

		for _, l := range h.mempool.Registry() {
			if totalTxBytes < req.MaxTxBytes {
				laneTxs, err := l.PrepareLane(ctx, req.MaxTxBytes, selectedTxs)
				if err != nil {
					h.logger.Error("failed to prepare lane; skipping", "lane", l.Name(), "err", err)
					continue
				}

				for _, txBz := range laneTxs {
					totalTxBytes += int64(len(txBz))

					txHash := sha256.Sum256(txBz)
					txHashStr := hex.EncodeToString(txHash[:])

					selectedTxs[txHashStr] = txBz
				}
			}
		}

		proposalTxs := make([][]byte, 0, len(selectedTxs))
		for _, txBz := range selectedTxs {
			proposalTxs = append(proposalTxs, txBz)
		}

		return abci.ResponsePrepareProposal{Txs: proposalTxs}
	}
}

// ProcessProposalHandler processes the proposal by verifying all transactions in the proposal
// according to each lane's verification logic. We verify proposals in a greedy fashion.
// If a lane's portion of the proposal is invalid, we reject the proposal. After a lane's portion
// of the proposal is verified, we pass the remaining transactions to the next lane in the chain.
func (h *ProposalHandler) ProcessProposalHandler() sdk.ProcessProposalHandler {
	return func(ctx sdk.Context, req abci.RequestProcessProposal) abci.ResponseProcessProposal {
		if _, err := h.processLanesHandler(ctx, req.Txs); err != nil {
			h.logger.Error("failed to process lanes", "err", err)
			return abci.ResponseProcessProposal{Status: abci.ResponseProcessProposal_REJECT}
		}

		return abci.ResponseProcessProposal{Status: abci.ResponseProcessProposal_ACCEPT}
	}
}
