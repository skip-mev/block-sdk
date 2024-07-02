package free

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/staking/types"

	"github.com/skip-mev/block-sdk/v2/block/base"
)

const (
	// LaneName defines the name of the free lane.
	LaneName = "free"
)

// NewFreeLane returns a new free lane.
func NewFreeLane[C comparable](
	cfg base.LaneConfig,
	txPriority base.TxPriority[C],
	matchFn base.MatchHandler,
) *base.BaseLane {
	options := []base.LaneOption{
		base.WithMatchHandler(matchFn),
		base.WithMempoolConfigs[C](cfg, txPriority),
	}

	lane, err := base.NewBaseLane(
		cfg,
		LaneName,
		options...,
	)
	if err != nil {
		panic(err)
	}

	return lane
}

// DefaultMatchHandler returns the default match handler for the free lane. The
// default implementation matches transactions that are staking related. In particular,
// any transaction that is a MsgDelegate, MsgBeginRedelegate, or MsgCancelUnbondingDelegation.
func DefaultMatchHandler() base.MatchHandler {
	return func(ctx sdk.Context, tx sdk.Tx) bool {
		for _, msg := range tx.GetMsgs() {
			switch msg.(type) {
			case *types.MsgDelegate:
				return true
			case *types.MsgBeginRedelegate:
				return true
			case *types.MsgCancelUnbondingDelegation:
				return true
			}
		}

		return false
	}
}
