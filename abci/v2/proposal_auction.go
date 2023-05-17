package v2

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"reflect"
	"sort"

	abci "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	pobabci "github.com/skip-mev/pob/abci"
)

// TopOfBlock contains information about how the top of block should be built.
type TopOfBlock struct {
	// Txs contains the transactions that should be included in the top of block.
	Txs [][]byte

	// Size is the total size of the top of block.
	Size int64

	// Cache is the cache of transactions that were seen, stored in order to ignore them
	// when building the rest of the block.
	Cache map[string]struct{}
}

func NewTopOfBlock() TopOfBlock {
	return TopOfBlock{
		Cache: make(map[string]struct{}),
	}
}

// BuildTOB inputs all of the vote extensions and outputs a top of block proposal
// that includes the highest bidding valid transaction along with all the bundled
// transactions.
func (h *ProposalHandler) BuildTOB(ctx sdk.Context, voteExtensionInfo abci.ExtendedCommitInfo, maxBytes int64) TopOfBlock {
	// Get the bid transactions from the vote extensions.
	sortedBidTxs := h.GetBidsFromVoteExtensions(voteExtensionInfo.Votes)

	// Track the transactions we can remove from the mempool
	txsToRemove := make(map[sdk.Tx]struct{})

	// Attempt to select the highest bid transaction that is valid and whose
	// bundled transactions are valid.
	var topOfBlock TopOfBlock
	for _, bidTx := range sortedBidTxs {
		// Cache the context so that we can write it back to the original context
		// when we know we have a valid top of block bundle.
		cacheCtx, write := ctx.CacheContext()

		// Attempt to build the top of block using the bid transaction.
		proposal, err := h.buildTOB(cacheCtx, bidTx)
		if err != nil {
			h.logger.Info(
				"vote extension auction failed to verify auction tx",
				"err", err,
			)
			txsToRemove[bidTx] = struct{}{}
			continue
		}

		if proposal.Size <= maxBytes {
			// At this point, both the bid transaction itself and all the bundled
			// transactions are valid. So we select the bid transaction along with
			// all the bundled transactions and apply the state changes to the cache
			// context.
			topOfBlock = proposal
			write()

			break
		}

		h.logger.Info(
			"failed to select auction bid tx; auction tx size is too large",
			"tx_size", proposal.Size,
			"max_size", maxBytes,
		)
	}

	// Remove all of the transactions that were not valid.
	for tx := range txsToRemove {
		h.RemoveTx(tx)
	}

	return topOfBlock
}

// VerifyTOB verifies that the set of vote extensions used in prepare proposal deterministically
// produce the same top of block proposal.
func (h *ProposalHandler) VerifyTOB(ctx sdk.Context, proposalTxs [][]byte) (*pobabci.AuctionInfo, error) {
	// Proposal must include at least the auction info.
	if len(proposalTxs) < NumInjectedTxs {
		return nil, fmt.Errorf("proposal is too small; expected at least %d slots", NumInjectedTxs)
	}

	// Extract the auction info from the proposal.
	auctionInfo := &pobabci.AuctionInfo{}
	if err := auctionInfo.Unmarshal(proposalTxs[AuctionInfoIndex]); err != nil {
		return nil, fmt.Errorf("failed to unmarshal auction info: %w", err)
	}

	// Verify that the proposal contains the expected number of top of block transactions.
	if len(proposalTxs) < int(auctionInfo.NumTxs)+NumInjectedTxs {
		return nil, fmt.Errorf("number of txs in proposal do not match expected in auction info; expected at least %d slots", auctionInfo.NumTxs+NumInjectedTxs)
	}

	// unmarshal the vote extension information from the auction info
	lastCommitInfo := abci.ExtendedCommitInfo{}
	if err := lastCommitInfo.Unmarshal(auctionInfo.ExtendedCommitInfo); err != nil {
		return nil, fmt.Errorf("failed to unmarshal last commit info from auction info: %w", err)
	}

	// Build the top of block proposal from the auction info.
	expectedTOB := h.BuildTOB(ctx, lastCommitInfo, auctionInfo.MaxTxBytes)

	// Verify that the top of block txs matches the top of block proposal txs.
	actualTOBTxs := proposalTxs[NumInjectedTxs : auctionInfo.NumTxs+NumInjectedTxs]
	if !reflect.DeepEqual(actualTOBTxs, expectedTOB.Txs) {
		return nil, fmt.Errorf("expected top of block txs does not match top of block proposal")
	}

	return auctionInfo, nil
}

// GetBidsFromVoteExtensions returns all of the auction bid transactions from
// the vote extensions in sorted descending order.
func (h *ProposalHandler) GetBidsFromVoteExtensions(voteExtensions []abci.ExtendedVoteInfo) []sdk.Tx {
	bidTxs := make([]sdk.Tx, 0)

	// Iterate through all vote extensions and extract the auction transactions.
	for _, voteInfo := range voteExtensions {
		voteExtension := voteInfo.VoteExtension

		// Check if the vote extension contains an auction transaction.
		if bidTx, err := h.getAuctionTxFromVoteExtension(voteExtension); err == nil {
			bidTxs = append(bidTxs, bidTx)
		}
	}

	// Sort the auction transactions by their bid amount in descending order.
	sort.Slice(bidTxs, func(i, j int) bool {
		// In the case of an error, we want to sort the transaction to the end of the list.
		bidInfoI, err := h.mempool.GetAuctionBidInfo(bidTxs[i])
		if err != nil {
			return false
		}

		bidInfoJ, err := h.mempool.GetAuctionBidInfo(bidTxs[j])
		if err != nil {
			return true
		}

		return bidInfoI.Bid.IsGTE(bidInfoJ.Bid)
	})

	return bidTxs
}

// buildTOB verifies that the auction and bundled transactions are valid and
// returns the transactions that should be included in the top of block, size
// of the auction transaction and bundle, and a cache of all transactions that
// should be ignored.
func (h *ProposalHandler) buildTOB(ctx sdk.Context, bidTx sdk.Tx) (TopOfBlock, error) {
	proposal := NewTopOfBlock()

	// Ensure that the bid transaction is valid
	bidTxBz, err := h.PrepareProposalVerifyTx(ctx, bidTx)
	if err != nil {
		return proposal, err
	}

	bidInfo, err := h.mempool.GetAuctionBidInfo(bidTx)
	if err != nil {
		return proposal, err
	}

	// store the bytes of each ref tx as sdk.Tx bytes in order to build a valid proposal
	sdkTxBytes := make([][]byte, len(bidInfo.Transactions))

	// Ensure that the bundled transactions are valid
	for index, rawRefTx := range bidInfo.Transactions {
		// convert the bundled raw transaction to a sdk.Tx
		refTx, err := h.mempool.WrapBundleTransaction(rawRefTx)
		if err != nil {
			return TopOfBlock{}, err
		}

		txBz, err := h.PrepareProposalVerifyTx(ctx, refTx)
		if err != nil {
			return TopOfBlock{}, err
		}

		hashBz := sha256.Sum256(txBz)
		hash := hex.EncodeToString(hashBz[:])

		proposal.Cache[hash] = struct{}{}
		sdkTxBytes[index] = txBz
	}

	// cache the bytes of the bid transaction
	hashBz := sha256.Sum256(bidTxBz)
	hash := hex.EncodeToString(hashBz[:])
	proposal.Cache[hash] = struct{}{}

	txs := [][]byte{bidTxBz}
	txs = append(txs, sdkTxBytes...)

	// Set the top of block transactions and size.
	proposal.Txs = txs
	proposal.Size = int64(len(bidTxBz))

	return proposal, nil
}

// getAuctionTxFromVoteExtension extracts the auction transaction from the vote
// extension.
func (h *ProposalHandler) getAuctionTxFromVoteExtension(voteExtension []byte) (sdk.Tx, error) {
	if len(voteExtension) == 0 {
		return nil, fmt.Errorf("vote extension is empty")
	}

	// Attempt to unmarshal the auction transaction.
	bidTx, err := h.txDecoder(voteExtension)
	if err != nil {
		return nil, err
	}

	// Verify the auction transaction has bid information.
	if bidInfo, err := h.mempool.GetAuctionBidInfo(bidTx); err != nil || bidInfo == nil {
		return nil, fmt.Errorf("vote extension does not contain an auction transaction")
	}

	return bidTx, nil
}
