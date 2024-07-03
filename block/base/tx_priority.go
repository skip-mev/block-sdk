package base

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// DefaultTxPriority
func DefaultTxPriority() TxPriority[int] {
	return TxPriority[int]{
		GetTxPriority: func(_ context.Context, _ sdk.Tx) int {
			return 0
		},
		Compare: func(_, _ int) int {
			return 0
		},
		MinValue: 0,
	}
}
