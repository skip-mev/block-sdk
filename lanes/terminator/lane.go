package terminator

import (
	"context"
	"fmt"

	"cosmossdk.io/log"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkmempool "github.com/cosmos/cosmos-sdk/types/mempool"

	"github.com/skip-mev/block-sdk/v2/block"
	"github.com/skip-mev/block-sdk/v2/block/proposals"
	"github.com/skip-mev/block-sdk/v2/block/utils"
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
func (t Terminator) ProcessLane(_ sdk.Context, p proposals.Proposal, txs []sdk.Tx, _ block.ProcessLanesHandler) (proposals.Proposal, error) {
	if len(txs) > 0 {
		return p, fmt.Errorf("terminator lane should not have any transactions")
	}

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

// GetTxInfo is a no-op
func (t Terminator) GetTxInfo(_ sdk.Context, _ sdk.Tx) (utils.TxWithInfo, error) {
	return utils.TxWithInfo{}, fmt.Errorf("terminator lane should not have any transactions")
}

// SetAnteHandler is a no-op
func (t Terminator) SetAnteHandler(sdk.AnteHandler) {}

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

// Compare is a no-op
func (t Terminator) Compare(sdk.Context, sdk.Tx, sdk.Tx) (int, error) {
	return 0, nil
}

// Priority is a no-op
func (t Terminator) Priority(sdk.Context, sdk.Tx) any {
	return 0
}
