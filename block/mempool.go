package block

import (
	"context"
	"fmt"

	"cosmossdk.io/log"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkmempool "github.com/cosmos/cosmos-sdk/types/mempool"

	blocksdkmoduletypes "github.com/skip-mev/block-sdk/x/blocksdk/types"
)

var _ Mempool = (*LanedMempool)(nil)

// LaneFetcher defines the interface to get a lane stored in the x/blocksdk module.
type LaneFetcher interface {
	GetLane(ctx sdk.Context, id string) (lane blocksdkmoduletypes.Lane, err error)
	GetLanes(ctx sdk.Context) []blocksdkmoduletypes.Lane
}

type (
	// Mempool defines the Block SDK mempool interface.
	Mempool interface {
		sdkmempool.Mempool

		// Registry returns the mempool's lane registry.
		Registry(ctx sdk.Context) []Lane

		// Contains returns true if any of the lanes currently contain the transaction.
		Contains(tx sdk.Tx) bool

		// GetTxDistribution returns the number of transactions in each lane.
		GetTxDistribution() map[string]uint64
	}

	// LanedMempool defines the Block SDK mempool implementation. It contains a registry
	// of lanes, which allows for customizable block proposal construction.
	LanedMempool struct {
		logger log.Logger

		// registry contains the lanes in the mempool. The lanes are ordered
		// according to their priority. The first lane in the registry has the
		// highest priority and the last lane has the lowest priority.
		registry []Lane

		// moduleLaneFetcher is the mempool's interface to read on-chain lane
		// information in the x/blocksdk module.
		moduleLaneFetcher LaneFetcher
	}
)

// NewLanedMempool returns a new Block SDK LanedMempool. The laned mempool comprises
// a registry of lanes. Each lane is responsible for selecting transactions according
// to its own selection logic. The lanes are ordered according to their priority. The
// first lane in the registry has the highest priority. Proposals are verified according
// to the order of the lanes in the registry. Each transaction SHOULD only belong in one lane.
func NewLanedMempool(
	logger log.Logger,
	lanes []Lane,
	laneFetcher LaneFetcher,
) (*LanedMempool, error) {
	mempool := &LanedMempool{
		logger:            logger,
		registry:          lanes,
		moduleLaneFetcher: laneFetcher,
	}

	if err := mempool.ValidateBasic(); err != nil {
		return nil, err
	}

	return mempool, nil
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
func (m *LanedMempool) GetTxDistribution() map[string]uint64 {
	counts := make(map[string]uint64, len(m.registry))

	for _, lane := range m.registry {
		counts[lane.Name()] = uint64(lane.CountTx())
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

	unwrappedCtx := sdk.UnwrapSDKContext(ctx)
	for _, lane := range m.registry {
		if lane.Match(unwrappedCtx, tx) {
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
func (m *LanedMempool) Select(_ context.Context, _ [][]byte) sdkmempool.Iterator {
	return nil
}

// Remove removes a transaction from the mempool. This assumes that the transaction
// is contained in only one of the lanes.
func (m *LanedMempool) Remove(tx sdk.Tx) (err error) {
	defer func() {
		if r := recover(); r != nil {
			m.logger.Error("panic in Remove", "err", r)
			err = fmt.Errorf("panic in Remove: %v", r)
		}
	}()

	for _, lane := range m.registry {
		if lane.Contains(tx) {
			return lane.Remove(tx)
		}
	}

	return nil
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
//
// CONTRACT: this function panics if it fails.  It should be wrapped in a recover() mechanism.
// This function will NEVER panic if x/blocksdk is disabled.
func (m *LanedMempool) Registry(ctx sdk.Context) []Lane {
	// TODO check enabled param
	// if !enabled fall back to default registry and return

	// TODO add a last block updated check ?
	// potential future optimization
	chainLanes := m.moduleLaneFetcher.GetLanes(ctx)

	// order lanes and populate the necessary fields (maxBlockSize, etc)
	var err error
	m.registry, err = m.OrderLanes(chainLanes)
	if err != nil {
		panic(err)
	}
	return m.registry
}

func (m *LanedMempool) OrderLanes(chainLanes []blocksdkmoduletypes.Lane) (orderedLanes []Lane, err error) {
	orderedLanes = make([]Lane, len(chainLanes))
	for _, chainLane := range chainLanes {
		// panic protect
		if chainLane.GetOrder() >= uint64(len(orderedLanes)) {
			return orderedLanes, fmt.Errorf("lane order %d out  of bounds, invalid configuration", chainLane.GetOrder())
		}

		_, index, found := FindLane(m.registry, chainLane.Id)
		if !found {
			return orderedLanes, fmt.Errorf("lane %s not found in registry, invalid configuration", chainLane.Id)
		}

		lane := m.registry[index]
		lane.SetMaxBlockSpace(chainLane.MaxBlockSpace)
		orderedLanes[chainLane.GetOrder()] = lane

		// remove found lane from registry lanes for quicker find()
		m.registry[index] = m.registry[len(m.registry)-1] // Copy last element to index i.
		m.registry[len(m.registry)-1] = nil               // Erase last element (write zero value).
		m.registry = m.registry[:len(m.registry)-1]       // Truncate slice.
	}

	return orderedLanes, nil
}

// ValidateBasic validates the mempools configuration. ValidateBasic ensures
// the following:
// - The sum of the lane max block space percentages is less than or equal to 1.
// - There is no unused block space.
func (m *LanedMempool) ValidateBasic() error {
	if len(m.registry) == 0 {
		return fmt.Errorf("registry cannot be nil; must configure at least one lane")
	}

	if m.moduleLaneFetcher == nil {
		return fmt.Errorf("module lane fetcher cannot be nil; must be set")
	}

	sum := math.LegacyZeroDec()
	seenZeroMaxBlockSpace := false
	seenLanes := make(map[string]struct{})

	for _, lane := range m.registry {
		name := lane.Name()
		if _, seen := seenLanes[name]; seen {
			return fmt.Errorf("duplicate lane name %s", name)
		}

		maxBlockSpace := lane.GetMaxBlockSpace()
		if maxBlockSpace.IsZero() {
			seenZeroMaxBlockSpace = true
		}

		sum = sum.Add(lane.GetMaxBlockSpace())
		seenLanes[name] = struct{}{}
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

	if m.moduleLaneFetcher == nil {
		return fmt.Errorf("moduleLaneFetcher muset be set on mempool")
	}

	return nil
}
