package abci

import (
	"crypto/sha256"
	"encoding/hex"

	"cosmossdk.io/log"
	cometabci "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkmempool "github.com/cosmos/cosmos-sdk/types/mempool"
	"github.com/skip-mev/pob/blockbuster/lanes/auction"
	"github.com/skip-mev/pob/blockbuster/utils"
)

type (
	// TOBLaneVE contains the methods required by the VoteExtensionHandler
	// to interact with the local mempool i.e. the top of block lane.
	TOBLaneVE interface {
		sdkmempool.Mempool

		// Factory defines the API/functionality which is responsible for determining
		// if a transaction is a bid transaction and how to extract relevant
		// information from the transaction (bid, timeout, bidder, etc.).
		auction.Factory

		// VerifyTx is utilized to verify a bid transaction according to the preferences
		// of the top of block lane.
		VerifyTx(ctx sdk.Context, tx sdk.Tx) error
	}

	// VoteExtensionHandler contains the functionality and handlers required to
	// process, validate and build vote extensions.
	VoteExtensionHandler struct {
		logger log.Logger

		// tobLane is the top of block lane which is used to extract the top bidding
		// auction transaction from the local mempool.
		tobLane TOBLaneVE

		// txDecoder is used to decode the top bidding auction transaction
		txDecoder sdk.TxDecoder

		// txEncoder is used to encode the top bidding auction transaction
		txEncoder sdk.TxEncoder

		// cache is used to store the results of the vote extension verification
		// for a given block height.
		cache map[string]error

		// currentHeight is the block height the cache is valid for.
		currentHeight int64
	}
)

// NewVoteExtensionHandler returns an VoteExtensionHandler that contains the functionality and handlers
// required to inject, process, and validate vote extensions.
func NewVoteExtensionHandler(logger log.Logger, lane TOBLaneVE, txDecoder sdk.TxDecoder, txEncoder sdk.TxEncoder) *VoteExtensionHandler {
	return &VoteExtensionHandler{
		logger:        logger,
		tobLane:       lane,
		txDecoder:     txDecoder,
		txEncoder:     txEncoder,
		cache:         make(map[string]error),
		currentHeight: 0,
	}
}

// ExtendVoteHandler returns the ExtendVoteHandler ABCI handler that extracts
// the top bidding valid auction transaction from a validator's local mempool and
// returns it in its vote extension.
func (h *VoteExtensionHandler) ExtendVoteHandler() sdk.ExtendVoteHandler {
	return func(ctx sdk.Context, req *cometabci.RequestExtendVote) (*cometabci.ResponseExtendVote, error) {
		// Iterate through auction bids until we find a valid one
		auctionIterator := h.tobLane.Select(ctx, nil)
		txsToRemove := make(map[sdk.Tx]struct{}, 0)

		defer func() {
			if err := utils.RemoveTxsFromLane(txsToRemove, h.tobLane); err != nil {
				h.logger.Info(
					"failed to remove transactions from lane",
					"err", err,
				)
			}
		}()

		for ; auctionIterator != nil; auctionIterator = auctionIterator.Next() {
			bidTx := auctionIterator.Tx()

			// Verify the bid tx can be encoded and included in vote extension
			bidTxBz, hash, err := utils.GetTxHashStr(h.txEncoder, bidTx)
			if err != nil {
				h.logger.Info(
					"failed to get hash of auction bid tx",
					"err", err,
				)
				txsToRemove[bidTx] = struct{}{}

				continue
			}

			// Validate the auction transaction against a cache state
			cacheCtx, _ := ctx.CacheContext()
			if err := h.tobLane.VerifyTx(cacheCtx, bidTx); err != nil {
				h.logger.Info(
					"failed to verify auction bid tx",
					"tx_hash", hash,
					"err", err,
				)
				txsToRemove[bidTx] = struct{}{}

				continue
			}

			h.logger.Info("extending vote with auction transaction", "tx_hash", hash)
			return &cometabci.ResponseExtendVote{VoteExtension: bidTxBz}, nil
		}

		h.logger.Info(
			"extending vote with no auction transaction",
			"height", ctx.BlockHeight(),
		)

		return &cometabci.ResponseExtendVote{VoteExtension: []byte{}}, nil
	}
}

// VerifyVoteExtensionHandler returns the VerifyVoteExtensionHandler ABCI handler
// that verifies the vote extension included in RequestVerifyVoteExtension.
// In particular, it verifies that the vote extension is a valid auction transaction.
func (h *VoteExtensionHandler) VerifyVoteExtensionHandler() sdk.VerifyVoteExtensionHandler {
	return func(ctx sdk.Context, req *cometabci.RequestVerifyVoteExtension) (*cometabci.ResponseVerifyVoteExtension, error) {
		txBz := req.VoteExtension
		if len(txBz) == 0 {
			h.logger.Info(
				"verified vote extension with no auction transaction",
				"height", ctx.BlockHeight(),
			)

			return &cometabci.ResponseVerifyVoteExtension{Status: cometabci.ResponseVerifyVoteExtension_ACCEPT}, nil
		}

		// Reset the cache if necessary
		h.resetCache(ctx.BlockHeight())

		hashBz := sha256.Sum256(txBz)
		hash := hex.EncodeToString(hashBz[:])

		// Short circuit if we have already verified this vote extension
		if err, ok := h.cache[hash]; ok {
			if err != nil {
				h.logger.Info(
					"rejected vote extension",
					"tx_hash", hash,
					"height", ctx.BlockHeight(),
				)

				return &cometabci.ResponseVerifyVoteExtension{Status: cometabci.ResponseVerifyVoteExtension_REJECT}, err
			}

			h.logger.Info(
				"verified vote extension",
				"tx_hash", hash,
				"height", ctx.BlockHeight(),
			)

			return &cometabci.ResponseVerifyVoteExtension{Status: cometabci.ResponseVerifyVoteExtension_ACCEPT}, nil
		}

		// Decode the vote extension which should be a valid auction transaction
		bidTx, err := h.txDecoder(txBz)
		if err != nil {
			h.logger.Info(
				"rejected vote extension",
				"tx_hash", hash,
				"height", ctx.BlockHeight(),
				"err", err,
			)

			h.cache[hash] = err
			return &cometabci.ResponseVerifyVoteExtension{Status: cometabci.ResponseVerifyVoteExtension_REJECT}, err
		}

		// Verify the auction transaction and cache the result
		if err = h.tobLane.VerifyTx(ctx, bidTx); err != nil {
			h.logger.Info(
				"rejected vote extension",
				"tx_hash", hash,
				"height", ctx.BlockHeight(),
				"err", err,
			)

			if err := h.tobLane.Remove(bidTx); err != nil {
				h.logger.Info(
					"failed to remove auction transaction from lane",
					"tx_hash", hash,
					"height", ctx.BlockHeight(),
					"err", err,
				)
			}

			h.cache[hash] = err
			return &cometabci.ResponseVerifyVoteExtension{Status: cometabci.ResponseVerifyVoteExtension_REJECT}, err
		}

		h.cache[hash] = nil

		h.logger.Info(
			"verified vote extension",
			"tx_hash", hash,
			"height", ctx.BlockHeight(),
		)

		return &cometabci.ResponseVerifyVoteExtension{Status: cometabci.ResponseVerifyVoteExtension_ACCEPT}, nil
	}
}

// checkStaleCache checks if the current height differs than the previous height at which
// the vote extensions were verified in. If so, it resets the cache to allow transactions to be
// reverified.
func (h *VoteExtensionHandler) resetCache(blockHeight int64) {
	if h.currentHeight != blockHeight {
		h.cache = make(map[string]error)
		h.currentHeight = blockHeight
	}
}
