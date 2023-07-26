package abci

import (
	"fmt"
	"reflect"
	"sort"

	abci "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/skip-mev/pob/blockbuster"
	"github.com/skip-mev/pob/blockbuster/utils"
)

// BuildTOB inputs all of the vote extensions and outputs a top of block proposal
// that includes the highest bidding valid transaction along with all the bundled
// transactions.
func (h *ProposalHandler) BuildTOB(ctx sdk.Context, voteExtensionInfo abci.ExtendedCommitInfo, maxBytes int64) *blockbuster.Proposal {
	// Get the bid transactions from the vote extensions.
	sortedBidTxs := h.GetBidsFromVoteExtensions(voteExtensionInfo.Votes)

	// Track the transactions we can remove from the mempool
	txsToRemove := make(map[sdk.Tx]struct{})

	// Attempt to select the highest bid transaction that is valid and whose
	// bundled transactions are valid.
	topOfBlock := blockbuster.NewProposal(maxBytes)
	for _, bidTx := range sortedBidTxs {
		// Cache the context so that we can write it back to the original context
		// when we know we have a valid top of block bundle.
		cacheCtx, write := ctx.CacheContext()

		// Attempt to build the top of block using the bid transaction.
		proposal, err := h.buildTOB(cacheCtx, bidTx, maxBytes)
		if err != nil {
			h.logger.Info(
				"vote extension auction failed to verify auction tx",
				"err", err,
			)
			txsToRemove[bidTx] = struct{}{}
			continue
		}

		// At this point, both the bid transaction itself and all the bundled
		// transactions are valid. So we select the bid transaction along with
		// all the bundled transactions and apply the state changes to the cache
		// context.
		topOfBlock = proposal
		write()

		break
	}

	// Remove all of the transactions that were not valid.
	if err := utils.RemoveTxsFromLane(txsToRemove, h.tobLane); err != nil {
		h.logger.Error(
			"failed to remove transactions from lane",
			"err", err,
		)
	}

	return topOfBlock
}

// VerifyTOB verifies that the set of vote extensions used in prepare proposal deterministically
// produce the same top of block proposal.
func (h *ProposalHandler) VerifyTOB(ctx sdk.Context, proposalTxs [][]byte) (*AuctionInfo, error) {
	// Proposal must include at least the auction info.
	if len(proposalTxs) < NumInjectedTxs {
		return nil, fmt.Errorf("proposal is too small; expected at least %d slots", NumInjectedTxs)
	}

	// Extract the auction info from the proposal.
	auctionInfo := &AuctionInfo{}
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

	// verify that the included vote extensions are valid in accordance with the
	// the preferences of the application
	cacheCtx, _ := ctx.CacheContext()
	if err := h.validateVoteExtensionsFn(cacheCtx, cacheCtx.BlockHeight(), lastCommitInfo); err != nil {
		return nil, fmt.Errorf("failed to validate vote extensions: %w", err)
	}

	// Build the top of block proposal from the auction info.
	expectedTOB := h.BuildTOB(cacheCtx, lastCommitInfo, auctionInfo.MaxTxBytes)

	// Verify that the top of block txs matches the top of block proposal txs.
	actualTOBTxs := proposalTxs[NumInjectedTxs : auctionInfo.NumTxs+NumInjectedTxs]
	if !reflect.DeepEqual(actualTOBTxs, expectedTOB.GetTxs()) {
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
		bidInfoI, err := h.tobLane.GetAuctionBidInfo(bidTxs[i])
		if err != nil {
			return false
		}

		bidInfoJ, err := h.tobLane.GetAuctionBidInfo(bidTxs[j])
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
func (h *ProposalHandler) buildTOB(ctx sdk.Context, bidTx sdk.Tx, maxBytes int64) (*blockbuster.Proposal, error) {
	proposal := blockbuster.NewProposal(maxBytes)

	// cache the bytes of the bid transaction
	txBz, _, err := utils.GetTxHashStr(h.txEncoder, bidTx)
	if err != nil {
		return proposal, err
	}

	// Ensure that the bid transaction is valid
	if err := h.tobLane.VerifyTx(ctx, bidTx); err != nil {
		return proposal, err
	}

	bidInfo, err := h.tobLane.GetAuctionBidInfo(bidTx)
	if err != nil {
		return proposal, err
	}

	// store the bytes of each ref tx as sdk.Tx bytes in order to build a valid proposal
	txs := [][]byte{txBz}

	// Ensure that the bundled transactions are valid
	for _, rawRefTx := range bidInfo.Transactions {
		// convert the bundled raw transaction to a sdk.Tx
		refTx, err := h.tobLane.WrapBundleTransaction(rawRefTx)
		if err != nil {
			return proposal, err
		}

		// convert the sdk.Tx to a hash and bytes
		txBz, _, err := utils.GetTxHashStr(h.txEncoder, refTx)
		if err != nil {
			return proposal, err
		}

		txs = append(txs, txBz)
	}

	// Add the bundled transactions to the proposal.
	if err := proposal.UpdateProposal(h.tobLane, txs); err != nil {
		return proposal, err
	}

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
	if bidInfo, err := h.tobLane.GetAuctionBidInfo(bidTx); err != nil || bidInfo == nil {
		return nil, fmt.Errorf("vote extension does not contain an auction transaction")
	}

	return bidTx, nil
}
