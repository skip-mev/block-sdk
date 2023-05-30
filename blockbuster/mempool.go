package blockbuster

import (
	"context"
	"errors"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkmempool "github.com/cosmos/cosmos-sdk/types/mempool"
)

var _ sdkmempool.Mempool = (*Mempool)(nil)

// Mempool defines the Blockbuster mempool implement. It contains a registry
// of lanes, which allows for customizable block proposal construction.
type Mempool struct {
	registry []Lane
}

func NewMempool(lanes ...Lane) *Mempool {
	return &Mempool{
		registry: lanes,
	}
}

// TODO: Consider using a tx cache in Mempool and returning the length of that
// cache instead of relying on lane count tracking.
func (m *Mempool) CountTx() int {
	var total int
	for _, lane := range m.registry {
		// TODO: If a global lane exists, we assume that lane has all transactions
		// and we return the total.
		//
		// if lane.Name() == LaneNameGlobal {
		// 	return lane.CountTx()
		// }

		total += lane.CountTx()
	}

	return total
}

// Insert inserts a transaction into every lane that it matches. Insertion will
// be attempted on all lanes, even if an error is encountered.
func (m *Mempool) Insert(ctx context.Context, tx sdk.Tx) error {
	errs := make([]error, 0, len(m.registry))

	for _, lane := range m.registry {
		if lane.Match(tx) {
			err := lane.Insert(ctx, tx)
			errs = append(errs, err)
		}
	}

	return errors.Join(errs...)
}

// Insert returns a nil iterator.
//
// TODO:
// - Determine if it even makes sense to return an iterator. What does that even
// mean in the context where you have multiple lanes?
// - Perhaps consider implementing and returning a no-op iterator?
func (m *Mempool) Select(_ context.Context, _ [][]byte) sdkmempool.Iterator {
	return nil
}

// Remove removes a transaction from every lane that it matches. Removal will be
// attempted on all lanes, even if an error is encountered.
func (m *Mempool) Remove(tx sdk.Tx) error {
	errs := make([]error, 0, len(m.registry))

	for _, lane := range m.registry {
		if lane.Match(tx) {
			err := lane.Remove(tx)
			errs = append(errs, err)
		}
	}

	return errors.Join(errs...)
}
