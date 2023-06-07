package terminator

import (
	"context"
	"fmt"

	"github.com/cometbft/cometbft/libs/log"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkmempool "github.com/cosmos/cosmos-sdk/types/mempool"
	"github.com/skip-mev/pob/blockbuster"
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

var _ blockbuster.Lane = (*Terminator)(nil)

// PrepareLane is a no-op
func (t Terminator) PrepareLane(_ sdk.Context, proposal *blockbuster.Proposal, _ int64, _ blockbuster.PrepareLanesHandler) *blockbuster.Proposal {
	return proposal
}

// ProcessLane is a no-op
func (t Terminator) ProcessLane(ctx sdk.Context, _ [][]byte, _ blockbuster.ProcessLanesHandler) (sdk.Context, error) {
	return ctx, nil
}

// Name returns the name of the lane
func (t Terminator) Name() string {
	return "Terminator"
}

// Match is a no-op
func (t Terminator) Match(sdk.Tx) bool {
	return false
}

// VerifyTx is a no-op
func (t Terminator) VerifyTx(sdk.Context, sdk.Tx) error {
	return fmt.Errorf("Terminator lane should not be called")
}

// Contains is a no-op
func (t Terminator) Contains(sdk.Tx) (bool, error) {
	return false, nil
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

// ValidateLaneBasic is a no-op
func (t Terminator) ProcessLaneBasic([][]byte) error {
	return nil
}

// SetLaneConfig is a no-op
func (t Terminator) SetAnteHandler(sdk.AnteHandler) {}

// Logger is a no-op
func (t Terminator) Logger() log.Logger {
	return log.NewNopLogger()
}

// GetMaxBlockSpace is a no-op
func (t Terminator) GetMaxBlockSpace() sdk.Dec {
	return sdk.ZeroDec()
}
