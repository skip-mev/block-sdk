package ante

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/skip-mev/block-sdk/x/auction/types"
)

// If the current execution mode is FinalizeBlock, we do not need to compare the bid to the
// top bid because the transaction will be executed in the current block. Additionally,
// if the current execution mode is ProcessProposal, we should not be comparing the proposal
// with the validators local mempool as it may be storing a higher bid than the one in the
// proposer's mempool. Lastly, since the MEV lane removes transactions only after it
// has selected it's highest bid, we should not be comparing the bid to the top bid in
// the application-side mempool.
var currentHeightExecModes = map[sdk.ExecMode]struct{}{
	sdk.ExecModePrepareProposal: {},
	sdk.ExecModeProcessProposal: {},
	sdk.ExecModeFinalize:        {},
}

// MEVLane is an interface that defines the methods required to interact with the MEV
// lane.
type MEVLane interface {
	GetAuctionBidInfo(tx sdk.Tx) (*types.BidInfo, error)
	GetTopAuctionTx(ctx context.Context) sdk.Tx
}

// AuctionKeeper is an interface that defines the methods required to interact with the
// auction keeper.
type AuctionKeeper interface {
	ValidateBidInfo(ctx sdk.Context, highestBid sdk.Coin, bidInfo *types.BidInfo) error
}

// ValidateTimeout validates that the timeout is greater than or equal to the expected block height
// the bid transaction will be executed in.
func ValidateTimeout(ctx sdk.Context, timeout int64) error {
	// If the timeout height is set to zero, this means that the searcher wanted next block execution.
	if timeout == 0 {
		// If the block at which the transaction was ingressed has been committed, we cannot
		// include this transaction in the next block.
		if ctx.IsReCheckTx() {
			return fmt.Errorf("timeout height cannot be 0 for recheck tx. please resubmit the transaction")
		}

		return nil
	}

	// Every transaction must have a timeout height greater than or equal to the height at which
	// the bid transaction will be executed.
	height := ctx.BlockHeight()
	if _, ok := currentHeightExecModes[ctx.ExecMode()]; !ok {
		height++
	}

	if height > timeout {
		return fmt.Errorf(
			"timeout height cannot be less than the current block height (timeout: %d, current block height: %d)",
			timeout,
			height,
		)
	}

	return nil
}
