package free

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/skip-mev/pob/blockbuster"
	"github.com/skip-mev/pob/blockbuster/lanes/base"
)

const (
	// LaneName defines the name of the free lane.
	LaneName = "free"
)

var _ blockbuster.Lane = (*Lane)(nil)

// FreeLane defines the lane that is responsible for processing free transactions.
type Lane struct {
	*base.DefaultLane
	Factory
}

// NewFreeLane returns a new free lane.
func NewFreeLane(cfg blockbuster.BaseLaneConfig, factory Factory) *Lane {
	if err := cfg.ValidateBasic(); err != nil {
		panic(err)
	}

	return &Lane{
		DefaultLane: base.NewDefaultLane(cfg).WithName(LaneName),
		Factory:     factory,
	}
}

// Match returns true if the transaction is a free transaction.
func (l *Lane) Match(ctx sdk.Context, tx sdk.Tx) bool {
	if l.MatchIgnoreList(ctx, tx) {
		return false
	}

	return l.IsFreeTx(tx)
}
