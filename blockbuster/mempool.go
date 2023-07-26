package blockbuster

import (
	"context"
	"fmt"

	"cosmossdk.io/math"
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

		// Match will return the lane that the transaction belongs to.
		Match(tx sdk.Tx) (Lane, error)

		// GetLane returns the lane with the given name.
		GetLane(name string) (Lane, error)
	}

	// Mempool defines the Blockbuster mempool implement. It contains a registry
	// of lanes, which allows for customizable block proposal construction.
	BBMempool struct {
		registry []Lane
	}
)

// NewMempool returns a new Blockbuster mempool. The blockbuster mempool is
// comprised of a registry of lanes. Each lane is responsible for selecting
// transactions according to its own selection logic. The lanes are ordered
// according to their priority. The first lane in the registry has the highest
// priority. Proposals are verified according to the order of the lanes in the
// registry. Basic mempool API, such as insertion, removal, and contains, are
// delegated to the first lane that matches the transaction. Each transaction
// should only belong in one lane.
func NewMempool(lanes ...Lane) *BBMempool {
	mempool := &BBMempool{
		registry: lanes,
	}

	if err := mempool.ValidateBasic(); err != nil {
		panic(err)
	}

	return mempool
}

// CountTx returns the total number of transactions in the mempool. This will
// be the sum of the number of transactions in each lane.
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

// Match will return the lane that the transaction belongs to. It matches to
// the first lane where lane.Match(tx) is true.
func (m *BBMempool) Match(tx sdk.Tx) (Lane, error) {
	for _, lane := range m.registry {
		if lane.Match(tx) {
			return lane, nil
		}
	}

	return nil, fmt.Errorf("no lane matched transaction")
}

// Insert will insert a transaction into the mempool. It inserts the transaction
// into the first lane that it matches.
func (m *BBMempool) Insert(ctx context.Context, tx sdk.Tx) error {
	lane, err := m.Match(tx)
	if err != nil {
		return err
	}

	return lane.Insert(ctx, tx)
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

// Remove removes a transaction from the mempool based on the first lane that
// it matches.
func (m *BBMempool) Remove(tx sdk.Tx) error {
	lane, err := m.Match(tx)
	if err != nil {
		return err
	}

	return lane.Remove(tx)
}

// Contains returns true if the transaction is contained in the mempool. It
// checks the first lane that it matches to.
func (m *BBMempool) Contains(tx sdk.Tx) (bool, error) {
	lane, err := m.Match(tx)
	if err != nil {
		return false, err
	}

	return lane.Contains(tx)
}

// Registry returns the mempool's lane registry.
func (m *BBMempool) Registry() []Lane {
	return m.registry
}

// ValidateBasic validates the mempools configuration.
func (m *BBMempool) ValidateBasic() error {
	sum := math.LegacyZeroDec()
	seenZeroMaxBlockSpace := false

	for _, lane := range m.registry {
		maxBlockSpace := lane.GetMaxBlockSpace()
		if maxBlockSpace.IsZero() {
			seenZeroMaxBlockSpace = true
		}

		sum = sum.Add(lane.GetMaxBlockSpace())
	}

	switch {
	// Ensure that the sum of the lane max block space percentages is less than
	// or equal to 1.
	case sum.GT(math.LegacyOneDec()):
		return fmt.Errorf("sum of lane max block space percentages must be less than or equal to 1, got %s", sum)
	// Ensure that there is no unused block space.
	case sum.LT(math.LegacyOneDec()) && !seenZeroMaxBlockSpace:
		return fmt.Errorf("sum of total block space percentages will be less than 1")
	}

	return nil
}

// GetLane returns the lane with the given name.
func (m *BBMempool) GetLane(name string) (Lane, error) {
	for _, lane := range m.registry {
		if lane.Name() == name {
			return lane, nil
		}
	}

	return nil, fmt.Errorf("lane %s not found", name)
}
