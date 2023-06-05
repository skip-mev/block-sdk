package blockbuster

import (
	"crypto/sha256"
	"encoding/hex"

	"github.com/cometbft/cometbft/libs/log"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkmempool "github.com/cosmos/cosmos-sdk/types/mempool"
)

type (
	// Proposal defines a block proposal type.
	Proposal struct {
		// Txs is the list of transactions in the proposal.
		Txs [][]byte

		// SelectedTxs is a cache of the selected transactions in the proposal.
		SelectedTxs map[string]struct{}

		// TotalTxBytes is the total number of bytes currently included in the proposal.
		TotalTxBytes int64

		// MaxTxBytes is the maximum number of bytes that can be included in the proposal.
		MaxTxBytes int64
	}

	// PrepareLanesHandler wraps all of the lanes Prepare function into a single chained
	// function. You can think of it like an AnteHandler, but for preparing proposals in the
	// context of lanes instead of modules.
	PrepareLanesHandler func(ctx sdk.Context, proposal *Proposal) *Proposal

	// ProcessLanesHandler wraps all of the lanes Process functions into a single chained
	// function. You can think of it like an AnteHandler, but for processing proposals in the
	// context of lanes instead of modules.
	ProcessLanesHandler func(ctx sdk.Context, proposalTxs [][]byte) (sdk.Context, error)

	// BaseLaneConfig defines the basic functionality needed for a lane.
	BaseLaneConfig struct {
		Logger      log.Logger
		TxEncoder   sdk.TxEncoder
		TxDecoder   sdk.TxDecoder
		AnteHandler sdk.AnteHandler

		// MaxBlockSpace defines the relative percentage of block space that can be
		// used by this lane. NOTE: If this is set to zero, then there is no limit
		// on the number of transactions that can be included in the block for this
		// lane (up to maxTxBytes as provided by the request). This is useful for the default lane.
		MaxBlockSpace sdk.Dec
	}

	// Lane defines an interface used for block construction
	Lane interface {
		sdkmempool.Mempool

		// Name returns the name of the lane.
		Name() string

		// Match determines if a transaction belongs to this lane.
		Match(tx sdk.Tx) bool

		// VerifyTx verifies the transaction belonging to this lane.
		VerifyTx(ctx sdk.Context, tx sdk.Tx) error

		// Contains returns true if the mempool contains the given transaction.
		Contains(tx sdk.Tx) (bool, error)

		// PrepareLane which builds a portion of the block. Inputs include the max
		// number of bytes that can be included in the block and the selected transactions
		// thus from from previous lane(s) as mapping from their HEX-encoded hash to
		// the raw transaction.
		PrepareLane(ctx sdk.Context, proposal *Proposal, next PrepareLanesHandler) *Proposal

		// ProcessLane verifies this lane's portion of a proposed block.
		ProcessLane(ctx sdk.Context, proposalTxs [][]byte, next ProcessLanesHandler) (sdk.Context, error)
	}
)

// NewLaneConfig returns a new LaneConfig. This will be embedded in a lane.
func NewBaseLaneConfig(logger log.Logger, txEncoder sdk.TxEncoder, txDecoder sdk.TxDecoder, anteHandler sdk.AnteHandler, maxBlockSpace sdk.Dec) BaseLaneConfig {
	return BaseLaneConfig{
		Logger:        logger,
		TxEncoder:     txEncoder,
		TxDecoder:     txDecoder,
		AnteHandler:   anteHandler,
		MaxBlockSpace: maxBlockSpace,
	}
}

func NewProposal(maxTxBytes int64) *Proposal {
	return &Proposal{
		Txs:         make([][]byte, 0),
		SelectedTxs: make(map[string]struct{}),
		MaxTxBytes:  maxTxBytes,
	}
}

// UpdateProposal updates the proposal with the given transactions and total size.
func (p *Proposal) UpdateProposal(txs [][]byte, totalSize int64) *Proposal {
	p.TotalTxBytes += totalSize
	p.Txs = append(p.Txs, txs...)

	for _, tx := range txs {
		txHash := sha256.Sum256(tx)
		txHashStr := hex.EncodeToString(txHash[:])

		p.SelectedTxs[txHashStr] = struct{}{}
	}

	return p
}
