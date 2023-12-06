package base

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// DefaultMatchHandler returns a default implementation of the MatchHandler. It matches all
// transactions.
func DefaultMatchHandler() MatchHandler {
	return func(ctx sdk.Context, tx sdk.Tx) bool {
		return true
	}
}

// VerifyNoMatches returns an error if any of the transactions match the lane.
func (l *BaseLane) VerifyNoMatches(ctx sdk.Context, txs []sdk.Tx) error {
	for _, tx := range txs {
		if l.Match(ctx, tx) {
			return fmt.Errorf("transaction belongs to lane when it should not")
		}
	}

	return nil
}

// NewMatchHandler returns a match handler that matches transactions
// that match the lane and do not match with any of the provided match handlers.
// In the context of building an application, you would want to use this to
// ignore the match handlers of other lanes in the application.
func NewMatchHandler(mh MatchHandler, ignoreMHs ...MatchHandler) MatchHandler {
	return func(ctx sdk.Context, tx sdk.Tx) bool {
		for _, ignoreMH := range ignoreMHs {
			if ignoreMH(ctx, tx) {
				return false
			}
		}

		return mh(ctx, tx)
	}
}
