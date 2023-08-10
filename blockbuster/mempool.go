package blockbuster

import (
	"context"
	"fmt"
	"strings"

	"cosmossdk.io/log"
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

		// Contains returns the any of the lanes currently contain the transaction.
		Contains(tx sdk.Tx) bool

		// GetTxDistribution returns the number of transactions in each lane.
		GetTxDistribution() map[string]int

		// GetLane returns the lane with the given name.
		GetLane(name string) (Lane, error)
	}

	// Mempool defines the Blockbuster mempool implement. It contains a registry
	// of lanes, which allows for customizable block proposal construction.
	BBMempool struct {
		registry []Lane
		logger   log.Logger
	}
)

// NewMempool returns a new Blockbuster mempool. The blockbuster mempool is
// comprised of a registry of lanes. Each lane is responsible for selecting
// transactions according to its own selection logic. The lanes are ordered
// according to their priority. The first lane in the registry has the highest
// priority. Proposals are verified according to the order of the lanes in the
// registry. Each transaction should only belong in one lane but this is NOT enforced.
// To enforce that each transaction belong to a single lane, you must configure the
// ignore list of each lane to include all preceding lanes. Basic mempool API will
// attempt to insert, remove transactions from all lanes it belongs to.
func NewMempool(logger log.Logger, lanes ...Lane) *BBMempool {
	mempool := &BBMempool{
		logger:   logger,
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

// Insert will insert a transaction into the mempool. It inserts the transaction
// into the first lane that it matches.
func (m *BBMempool) Insert(ctx context.Context, tx sdk.Tx) error {
	var errors []string

	unwrappedCtx := sdk.UnwrapSDKContext(ctx)
	for _, lane := range m.registry {
		if !lane.Match(unwrappedCtx, tx) {
			continue
		}

		if err := lane.Insert(ctx, tx); err != nil {
			m.logger.Debug("failed to insert tx into lane", "lane", lane.Name(), "err", err)
			errors = append(errors, fmt.Sprintf("failed to insert tx into lane %s: %s", lane.Name(), err.Error()))
		}
	}

	if len(errors) == 0 {
		return nil
	}

	return fmt.Errorf(strings.Join(errors, ";"))
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

// Remove removes a transaction from all of the lanes it is currently in.
func (m *BBMempool) Remove(tx sdk.Tx) error {
	var errors []string

	for _, lane := range m.registry {
		if !lane.Contains(tx) {
			continue
		}

		if err := lane.Remove(tx); err != nil {
			m.logger.Debug("failed to remove tx from lane", "lane", lane.Name(), "err", err)

			// We only care about errors that are not "tx not found" errors.
			//
			// TODO: Figure out whether we should be erroring in the mempool if
			// the tx is not found in the lane. Downstream, if the removal fails runTx will
			// error out and will NOT execute runMsgs (which is where the tx is actually
			// executed).
			if err != sdkmempool.ErrTxNotFound {
				errors = append(errors, fmt.Sprintf("failed to remove tx from lane %s: %s;", lane.Name(), err.Error()))
			}
		}
	}

	if len(errors) == 0 {
		return nil
	}

	return fmt.Errorf(strings.Join(errors, ";"))
}

// Contains returns true if the transaction is contained in any of the lanes.
func (m *BBMempool) Contains(tx sdk.Tx) bool {
	for _, lane := range m.registry {
		if lane.Contains(tx) {
			return true
		}
	}

	return false
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
