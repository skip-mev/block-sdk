package types

import (
	"fmt"

	"cosmossdk.io/math"
)

// Lanes is a type alias for a slice of Lane objects.
type Lanes []Lane

// ValidateBasic performs stateless validation of a Lane.
func (l *Lane) ValidateBasic() error {
	if l.Id == "" {
		return fmt.Errorf("lane must have an id specified")
	}

	if l.MaxBlockSpace.IsNil() || l.MaxBlockSpace.IsNegative() || l.MaxBlockSpace.GT(math.LegacyOneDec()) {
		return fmt.Errorf("max block space must be set to a value between 0 and 1")
	}

	return nil
}

// ValidateBasic performs stateless validation all Lanes.  Performs lane.ValidateBasic()
// for each lane, verifies that order is monotonically increasing for all lanes, and
// checks that IDs are unique.
func (ls Lanes) ValidateBasic() error {
	laneIDMap := make(map[string]struct{})
	encounteredOrders := make(map[uint64]struct{})

	// validate each lane and check that ID and order fields are unique
	for _, l := range ls {
		if err := l.ValidateBasic(); err != nil {
			return err
		}

		_, found := laneIDMap[l.Id]
		if found {
			return fmt.Errorf("duplicate lane ID found: %s", l.Id)
		}

		laneIDMap[l.Id] = struct{}{}

		_, found = encounteredOrders[l.Order]
		if found {
			return fmt.Errorf("duplicate lane order found: %d", l.Order)
		}

		encounteredOrders[l.Order] = struct{}{}
	}

	// check if an order value is outside the expected range
	// since all order values are already unique, if a value
	// is greater than the total number of lanes it must break
	// the rule of monotonicity
	numLanes := uint64(len(ls))
	for order := range encounteredOrders {
		// subtract 1 since orders starts at 0
		if order > numLanes-1 {
			return fmt.Errorf("orders are not set in a monotonically increasing order: order value: %d", order)
		}
	}

	return nil
}
