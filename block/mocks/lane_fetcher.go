package mocks

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	blocksdkmoduletypes "github.com/skip-mev/block-sdk/v2/x/blocksdk/types"
)

type MockLaneFetcher struct {
	getLaneHandler   func() (blocksdkmoduletypes.Lane, error)
	getLanesHandler  func() []blocksdkmoduletypes.Lane
	getParamsHandler func() (blocksdkmoduletypes.Params, error)
}

func NewMockLaneFetcher(
	getLane func() (blocksdkmoduletypes.Lane, error),
	getLanes func() []blocksdkmoduletypes.Lane,
	getParams func() (blocksdkmoduletypes.Params, error),
) MockLaneFetcher {
	return MockLaneFetcher{
		getLaneHandler:   getLane,
		getLanesHandler:  getLanes,
		getParamsHandler: getParams,
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

func (m MockLaneFetcher) GetParams(_ sdk.Context) (blocksdkmoduletypes.Params, error) {
	return m.getParamsHandler()
}
