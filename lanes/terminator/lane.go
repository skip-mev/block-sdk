package terminator

import (
	"context"

	"cosmossdk.io/math"
	"github.com/cometbft/cometbft/libs/log"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkmempool "github.com/cosmos/cosmos-sdk/types/mempool"

	"github.com/skip-mev/block-sdk/block"
	"github.com/skip-mev/block-sdk/block/proposals"
)

const (
	LaneName = "Terminator"
)

// Terminator Lane will get added to the chain to simplify chaining code so that we
// don't need to check if next == nil further up the chain
//
// sniped from the sdk
//
//	                      ______
//	                   <((((((\\\
//	                   /      . }\
//	                   ;--..--._|}
//	(\                 '--/\--'  )
//	 \\                | '-'  :'|
//	  \\               . -==- .-|
//	   \\               \.__.'   \--._
//	   [\\          __.--|       //  _/'--.
//	   \ \\       .'-._ ('-----'/ __/      \
//	    \ \\     /   __>|      | '--.       |
//	     \ \\   |   \   |     /    /       /
//	      \ '\ /     \  |     |  _/       /
//	       \  \       \ |     | /        /
//	 snd    \  \      \        /
type Terminator struct{}

var _ block.Lane = (*Terminator)(nil)

// PrepareLane is a no-op
func (t Terminator) PrepareLane(_ sdk.Context, proposal proposals.Proposal, _ block.PrepareLanesHandler) (proposals.Proposal, error) {
	return proposal, nil
}

// ProcessLane is a no-op
func (t Terminator) ProcessLane(_ sdk.Context, p proposals.Proposal, _ [][]byte, _ block.ProcessLanesHandler) (proposals.Proposal, error) {
	return p, nil
}

// GetMaxBlockSpace is a no-op
func (t Terminator) GetMaxBlockSpace() math.LegacyDec {
	return math.LegacyZeroDec()
}

// Logger is a no-op
func (t Terminator) Logger() log.Logger {
	return log.NewNopLogger()
}

// Name returns the name of the lane
func (t Terminator) Name() string {
	return LaneName
}

// SetAnteHandler is a no-op
func (t Terminator) SetAnteHandler(sdk.AnteHandler) {}

// SetIgnoreList is a no-op
func (t Terminator) SetIgnoreList([]block.Lane) {}

// Match is a no-op
func (t Terminator) Match(sdk.Context, sdk.Tx) bool {
	return false
}

// Contains is a no-op
func (t Terminator) Contains(sdk.Tx) bool {
	return false
}

// CountTx is a no-op
func (t Terminator) CountTx() int {
	return 0
}

// Insert is a no-op
func (t Terminator) Insert(context.Context, sdk.Tx) error {
	return nil
}

// Remove is a no-op
func (t Terminator) Remove(sdk.Tx) error {
	return nil
}

// Select is a no-op
func (t Terminator) Select(context.Context, [][]byte) sdkmempool.Iterator {
	return nil
}

// HasHigherPriority is a no-op
func (t Terminator) Compare(sdk.Context, sdk.Tx, sdk.Tx) (int, error) {
	return 0, nil
}
