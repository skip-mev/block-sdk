package block

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/skip-mev/block-sdk/block/proposals"
)

type (
	// PrepareLanesHandler wraps all of the lanes' PrepareLane function into a single chained
	// function. You can think of it like an AnteHandler, but for preparing proposals in the
	// context of lanes instead of modules.
	PrepareLanesHandler func(ctx sdk.Context, proposal proposals.Proposal) (proposals.Proposal, error)

	// ProcessLanesHandler wraps all of the lanes' ProcessLane functions into a single chained
	// function. You can think of it like an AnteHandler, but for processing proposals in the
	// context of lanes instead of modules.
	ProcessLanesHandler func(ctx sdk.Context) (sdk.Context, error)
)

// NoOpPrepareLanesHandler returns a no-op prepare lanes handler.
// This should only be used for testing.
func NoOpPrepareLanesHandler() PrepareLanesHandler {
	return func(ctx sdk.Context, proposal proposals.Proposal) (proposals.Proposal, error) {
		return proposal, nil
	}
}

// NoOpProcessLanesHandler returns a no-op process lanes handler.
// This should only be used for testing.
func NoOpProcessLanesHandler() ProcessLanesHandler {
	return func(ctx sdk.Context) (sdk.Context, error) {
		return ctx, nil
	}
}
