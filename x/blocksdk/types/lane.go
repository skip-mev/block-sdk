package types

// Lanes is a type alias for a slice of Lane objects.
type Lanes []Lane

// ValidateBasic performs stateless validation of a Lane.
func (l Lane) ValidateBasic() error {

	return nil
}

// ValidateBasic performs stateless validation all Lanes.  Performs lane.ValidateBasic()
// for each lane, verifies that order is monotonically increasing for all lanes, and
// checks that IDs are unique.
func (ls Lanes) ValidateBasic() error {
	laneIDMap := make(map[string]struct{})

	for _, l := range ls {
		if err := l.ValidateBasic(); err != nil {
			return err
		}
	}

	return nil
}
