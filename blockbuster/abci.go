package blockbuster

import (
	"crypto/sha256"
	"encoding/hex"

	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cometbft/cometbft/libs/log"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type ProposalHandler struct {
	logger  log.Logger
	mempool Mempool
}

func NewProposalHandler(logger log.Logger, mempool Mempool) *ProposalHandler {
	return &ProposalHandler{
		logger:  logger,
		mempool: mempool,
	}
}

func (h *ProposalHandler) PrepareProposalHandler() sdk.PrepareProposalHandler {
	return func(ctx sdk.Context, req abci.RequestPrepareProposal) abci.ResponsePrepareProposal {
		var (
			selectedTxs  = make(map[string][]byte)
			totalTxBytes int64
		)

		for _, l := range h.mempool.registry {
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

func (h *ProposalHandler) ProcessProposalHandler() sdk.ProcessProposalHandler {
	return func(ctx sdk.Context, req abci.RequestProcessProposal) abci.ResponseProcessProposal {
		for _, l := range h.mempool.registry {
			if err := l.ProcessLane(ctx, req.Txs); err != nil {
				h.logger.Error("failed to process lane", "lane", l.Name(), "err", err)
				return abci.ResponseProcessProposal{Status: abci.ResponseProcessProposal_REJECT}
			}
		}

		return abci.ResponseProcessProposal{Status: abci.ResponseProcessProposal_ACCEPT}
	}
}
