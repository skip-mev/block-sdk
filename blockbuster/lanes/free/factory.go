package free

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/staking/types"
)

type (
	// Factory defines the interface for processing free transactions. It is
	// a wrapper around all of the functionality that each application chain must implement
	// in order for free processing to work.
	Factory interface {
		// IsFreeTx defines a function that checks if a transaction qualifies as free.
		IsFreeTx(tx sdk.Tx) bool
	}

	// DefaultFreeFactory defines a default implmentation for the free factory interface for processing free transactions.
	DefaultFreeFactory struct {
		txDecoder sdk.TxDecoder
	}
)

var _ Factory = (*DefaultFreeFactory)(nil)

// NewDefaultFreeFactory returns a default free factory interface implementation.
func NewDefaultFreeFactory(txDecoder sdk.TxDecoder) Factory {
	return &DefaultFreeFactory{
		txDecoder: txDecoder,
	}
}

// IsFreeTx defines a default function that checks if a transaction is free. In this case,
// any transaction that is a delegation/redelegation transaction is free.
func (config *DefaultFreeFactory) IsFreeTx(tx sdk.Tx) bool {
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
