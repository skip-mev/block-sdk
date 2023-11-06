package mocks

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	blocksdkmoduletypes "github.com/skip-mev/block-sdk/x/blocksdk/types"
)

type MockLaneFetcher struct {
	getLaneHandler  func() (blocksdkmoduletypes.Lane, error)
	getLanesHandler func() []blocksdkmoduletypes.Lane
}

func NewMockLaneFetcher(getLane func() (blocksdkmoduletypes.Lane, error), getLanes func() []blocksdkmoduletypes.Lane) MockLaneFetcher {
	return MockLaneFetcher{
		getLaneHandler:  getLane,
		getLanesHandler: getLanes,
	}
}

func (m *MockLaneFetcher) SetGetLaneHandler(h func() (blocksdkmoduletypes.Lane, error)) {
	m.getLaneHandler = h
}

func (m MockLaneFetcher) GetLane(_ sdk.Context, _ string) (blocksdkmoduletypes.Lane, error) {
	return m.getLaneHandler()
}

func (m *MockLaneFetcher) SetGetLanesHandler(h func() []blocksdkmoduletypes.Lane) {
	m.getLanesHandler = h
}

func (m MockLaneFetcher) GetLanes(_ sdk.Context) []blocksdkmoduletypes.Lane {
	return m.getLanesHandler()
}
