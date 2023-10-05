package block

import (
	"context"
	"fmt"
	"strings"

	"cosmossdk.io/log"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkmempool "github.com/cosmos/cosmos-sdk/types/mempool"
)

var _ Mempool = (*LanedMempool)(nil)

type (
	// Mempool defines the Block SDK mempool interface.
	Mempool interface {
		sdkmempool.Mempool

		// Registry returns the mempool's lane registry.
		Registry() []Lane

		// Contains returns the any of the lanes currently contain the transaction.
		Contains(tx sdk.Tx) bool

		// GetTxDistribution returns the number of transactions in each lane.
		GetTxDistribution() map[string]int
	}

	// LanedMempool defines the Block SDK mempool implementation. It contains a registry
	// of lanes, which allows for customizable block proposal construction.
	LanedMempool struct {
		logger log.Logger

		// registry contains the lanes in the mempool. The lanes are ordered
		// according to their priority. The first lane in the registry has the
		// highest priority and the last lane has the lowest priority.
		registry []Lane
	}
)

// NewLanedMempool returns a new Block SDK LanedMempool. The laned mempool is
// comprised of a registry of lanes. Each lane is responsible for selecting
// transactions according to its own selection logic. The lanes are ordered
// according to their priority. The first lane in the registry has the highest
// priority. Proposals are verified according to the order of the lanes in the
// registry. Each transaction should only belong in one lane but this is NOT enforced.
// To enforce that each transaction belong to a single lane, you must configure the
// ignore list of each lane to include all preceding lanes. Basic mempool API will
// attempt to insert, remove transactions from all lanes it belongs to. It is recommended,
// that mutex is set to true when creating the mempool. This will ensure that each
// transaction cannot be inserted into the lanes before it.
func NewLanedMempool(logger log.Logger, mutex bool, lanes ...Lane) Mempool {
	mempool := &LanedMempool{
		logger:   logger,
		registry: lanes,
	}

	if err := mempool.ValidateBasic(); err != nil {
		panic(err)
	}

	// Set the ignore list for each lane
	if mutex {
		registry := mempool.registry
		for index, lane := range mempool.registry {
			if index > 0 {
				lane.SetIgnoreList(registry[:index])
			}
		}
	}

	return mempool
}

// CountTx returns the total number of transactions in the mempool. This will
// be the sum of the number of transactions in each lane.
func (m *LanedMempool) CountTx() int {
	var total int
	for _, lane := range m.registry {
		total += lane.CountTx()
	}

	return total
}

// GetTxDistribution returns the number of transactions in each lane.
func (m *LanedMempool) GetTxDistribution() map[string]int {
	counts := make(map[string]int, len(m.registry))

	for _, lane := range m.registry {
		counts[lane.Name()] = lane.CountTx()
	}

	return counts
}

// Insert will insert a transaction into the mempool. It inserts the transaction
// into the first lane that it matches.
func (m *LanedMempool) Insert(ctx context.Context, tx sdk.Tx) (err error) {
	defer func() {
		if r := recover(); r != nil {
			m.logger.Error("panic in Insert", "err", r)
			err = fmt.Errorf("panic in Insert: %v", r)
		}
	}()

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
func (m *LanedMempool) Select(_ context.Context, _ [][]byte) sdkmempool.Iterator {
	return nil
}

// Remove removes a transaction from all of the lanes it is currently in.
func (m *LanedMempool) Remove(tx sdk.Tx) (err error) {
	defer func() {
		if r := recover(); r != nil {
			m.logger.Error("panic in Remove", "err", r)
			err = fmt.Errorf("panic in Remove: %v", r)
		}
	}()

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
func (m *LanedMempool) Contains(tx sdk.Tx) (contains bool) {
	defer func() {
		if r := recover(); r != nil {
			m.logger.Error("panic in Contains", "err", r)
			contains = false
		}
	}()

	for _, lane := range m.registry {
		if lane.Contains(tx) {
			return true
		}
	}

	return false
}

// Registry returns the mempool's lane registry.
func (m *LanedMempool) Registry() []Lane {
	return m.registry
}

// ValidateBasic validates the mempools configuration. ValidateBasic ensures
// the following:
// - The sum of the lane max block space percentages is less than or equal to 1.
// - There is no unused block space.
func (m *LanedMempool) ValidateBasic() error {
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
