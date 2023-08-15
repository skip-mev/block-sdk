package free

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/skip-mev/pob/blockbuster"
)

const (
	// LaneName defines the name of the free lane.
	LaneName = "free"
)

var _ blockbuster.Lane = (*FreeLane)(nil)

// FreeLane defines the lane that is responsible for processing free transactions.
// By default, transactions that are staking related are considered free.
type FreeLane struct { //nolint
	*blockbuster.LaneConstructor
}

// NewFreeLane returns a new free lane.
func NewFreeLane(
	cfg blockbuster.LaneConfig,
	txPriority blockbuster.TxPriority[string],
	matchFn blockbuster.MatchHandler,
) *FreeLane {
	lane := blockbuster.NewLaneConstructor(
		cfg,
		LaneName,
		blockbuster.NewConstructorMempool[string](
			txPriority,
			cfg.TxEncoder,
			cfg.MaxTxs,
		),
		matchFn,
	)

	return &FreeLane{
		LaneConstructor: lane,
	}
}

// DefaultMatchHandler returns the default match handler for the free lane. The
// default implementation matches transactions that are staking related. In particular,
// any transaction that is a MsgDelegate, MsgBeginRedelegate, or MsgCancelUnbondingDelegation.
func DefaultMatchHandler() blockbuster.MatchHandler {
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
