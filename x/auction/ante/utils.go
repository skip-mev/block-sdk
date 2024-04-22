package ante

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/skip-mev/block-sdk/v2/x/auction/types"
)

// nextHeightExecModes is the subset of execution modes that need special treatment. In
// particular, these modes are used to signal that the transaction is being executed in the
// next block. This is used to determine if the current transaction should be compared
// against the top bid and what the expected timeout height should be.
var nextHeightExecModes = map[sdk.ExecMode]struct{}{
	sdk.ExecModeCheck:    {},
	sdk.ExecModeReCheck:  {},
	sdk.ExecModeSimulate: {},
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
	// Every transaction must have a timeout height greater than or equal to the height at which
	// the bid transaction will be executed.
	height := ctx.BlockHeight()
	if _, ok := nextHeightExecModes[ctx.ExecMode()]; ok {
		height++
	}

	if height != timeout {
		return fmt.Errorf(
			"you must set the timeout height to be the next block height got %d, expected %d",
			timeout,
			height,
		)
	}

	return nil
}
