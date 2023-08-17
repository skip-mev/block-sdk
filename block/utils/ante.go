package utils

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type (
	// Lane defines the required API dependencies for the IgnoreDecorator. The ignore decorator
	// will check if a transaction belongs to a lane by calling the Match function.
	Lane interface {
		Match(ctx sdk.Context, tx sdk.Tx) bool
	}

	// IgnoreDecorator is an AnteDecorator that wraps an existing AnteDecorator. It allows
	// for the AnteDecorator to be ignored for specified lanes.
	IgnoreDecorator struct {
		decorator sdk.AnteDecorator
		lanes     []Lane
	}
)

// NewIgnoreDecorator returns a new IgnoreDecorator instance.
func NewIgnoreDecorator(decorator sdk.AnteDecorator, lanes ...Lane) *IgnoreDecorator {
	return &IgnoreDecorator{
		decorator: decorator,
		lanes:     lanes,
	}
}

// AnteHandle implements the sdk.AnteDecorator interface. If the transaction belongs to
// one of the lanes, the next AnteHandler is called. Otherwise, the decorator's AnteHandler
// is called.
func (sd IgnoreDecorator) AnteHandle(
	ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler,
) (sdk.Context, error) {
	for _, lane := range sd.lanes {
		if lane.Match(ctx, tx) {
			return next(ctx, tx, simulate)
		}
	}

	return sd.decorator.AnteHandle(ctx, tx, simulate, next)
}
