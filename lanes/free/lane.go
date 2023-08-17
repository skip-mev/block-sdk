package free

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/skip-mev/pob/block"
	"github.com/skip-mev/pob/block/base"
)

const (
	// LaneName defines the name of the free lane.
	LaneName = "free"
)

var _ block.Lane = (*FreeLane)(nil)

// FreeLane defines the lane that is responsible for processing free transactions.
// By default, transactions that are staking related are considered free.
type FreeLane struct { //nolint
	*base.BaseLane
}

// NewFreeLane returns a new free lane.
func NewFreeLane(
	cfg base.LaneConfig,
	txPriority base.TxPriority[string],
	matchFn block.MatchHandler,
) *FreeLane {
	lane := base.NewBaseLane(
		cfg,
		LaneName,
		base.NewMempool[string](
			txPriority,
			cfg.TxEncoder,
			cfg.MaxTxs,
		),
		matchFn,
	)

	return &FreeLane{
		BaseLane: lane,
	}
}

// DefaultMatchHandler returns the default match handler for the free lane. The
// default implementation matches transactions that are staking related. In particular,
// any transaction that is a MsgDelegate, MsgBeginRedelegate, or MsgCancelUnbondingDelegation.
func DefaultMatchHandler() block.MatchHandler {
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
