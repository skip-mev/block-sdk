package base

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// DefaultTxPriority
func DefaultTxPriority() TxPriority[int] {
	return TxPriority[int]{
		GetTxPriority: func(goCtx context.Context, tx sdk.Tx) int {
			return 0
		},
		Compare: func(a, b int) int {
			return 0
		},
		MinValue: 0,
	}
}
