package blockbuster

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkmempool "github.com/cosmos/cosmos-sdk/types/mempool"
)

var _ Mempool = (*BBMempool)(nil)

type (
	// Mempool defines the Blockbuster mempool interface.
	Mempool interface {
		sdkmempool.Mempool

		// Registry returns the mempool's lane registry.
		Registry() []Lane

		// Contains returns true if the transaction is contained in the mempool.
		Contains(tx sdk.Tx) (bool, error)

		// GetTxDistribution returns the number of transactions in each lane.
		GetTxDistribution() map[string]int
	}

	// Mempool defines the Blockbuster mempool implement. It contains a registry
	// of lanes, which allows for customizable block proposal construction.
	BBMempool struct {
		registry []Lane
	}
)

func NewMempool(lanes ...Lane) *BBMempool {
	return &BBMempool{
		registry: lanes,
	}
}

// CountTx returns the total number of transactions in the mempool.
func (m *BBMempool) CountTx() int {
	var total int
	for _, lane := range m.registry {
		total += lane.CountTx()
	}

	return total
}

// GetTxDistribution returns the number of transactions in each lane.
func (m *BBMempool) GetTxDistribution() map[string]int {
	counts := make(map[string]int, len(m.registry))

	for _, lane := range m.registry {
		counts[lane.Name()] = lane.CountTx()
	}

	return counts
}

// Insert inserts a transaction into every lane that it matches. Insertion will
// be attempted on all lanes, even if an error is encountered.
func (m *BBMempool) Insert(ctx context.Context, tx sdk.Tx) error {
	for _, lane := range m.registry {
		if lane.Match(tx) {
			return lane.Insert(ctx, tx)
		}
	}

	return nil
}

// Insert returns a nil iterator.
//
// TODO:
// - Determine if it even makes sense to return an iterator. What does that even
// mean in the context where you have multiple lanes?
// - Perhaps consider implementing and returning a no-op iterator?
func (m *BBMempool) Select(_ context.Context, _ [][]byte) sdkmempool.Iterator {
	return nil
}

// Remove removes a transaction from the mempool. It removes the transaction
// from the first lane that it matches.
func (m *BBMempool) Remove(tx sdk.Tx) error {
	for _, lane := range m.registry {
		if lane.Match(tx) {
			return lane.Remove(tx)
		}
	}

	return nil
}

// Contains returns true if the transaction is contained in the mempool.
func (m *BBMempool) Contains(tx sdk.Tx) (bool, error) {
	for _, lane := range m.registry {
		if lane.Match(tx) {
			return lane.Contains(tx)
		}
	}

	return false, nil
}

// Registry returns the mempool's lane registry.
func (m *BBMempool) Registry() []Lane {
	return m.registry
}
