package keeper_test

import (
	"cosmossdk.io/math"

	"github.com/skip-mev/block-sdk/x/blocksdk/types"
)

func (s *KeeperTestSuite) TestQueryLane() {
	// pre-register a lane
	registeredLanes := types.Lanes{
		types.Lane{
			Id:            "registered1",
			MaxBlockSpace: math.LegacyMustNewDecFromStr("0.1"),
			Order:         0,
		},
		types.Lane{
			Id:            "registered2",
			MaxBlockSpace: math.LegacyMustNewDecFromStr("0.1"),
			Order:         1,
		},
	}

	for _, lane := range registeredLanes {
		s.Require().NoError(s.blocksdKeeper.AddLane(s.ctx, lane))
	}

	testCases := []struct {
		name     string
		query    *types.QueryLaneRequest
		expected types.Lane
		wantErr  bool
	}{
		{
			name:     "invalid lane does not exist",
			query:    &types.QueryLaneRequest{Id: "invalid"},
			expected: types.Lane{},
			wantErr:  true,
		},
		{
			name:     "valid query 1",
			query:    &types.QueryLaneRequest{Id: registeredLanes[0].Id},
			expected: registeredLanes[0],
			wantErr:  false,
		},
		{
			name:     "valid query 2",
			query:    &types.QueryLaneRequest{Id: registeredLanes[1].Id},
			expected: registeredLanes[1],
			wantErr:  false,
		},
	}
	for _, tc := range testCases {
		s.Run(tc.name, func() {
			resp, err := s.queryServer.Lane(s.ctx, tc.query)
			if tc.wantErr {
				s.Require().Error(err)
				return
			}

			s.Require().NoError(err)
			s.Require().Equal(tc.expected, resp.GetLane())
		})
	}
}

func (s *KeeperTestSuite) TestQueryLanes() {
	// pre-register a lane
	registeredLanes := types.Lanes{
		types.Lane{
			Id:            "registered1",
			MaxBlockSpace: math.LegacyMustNewDecFromStr("0.1"),
			Order:         0,
		},
		types.Lane{
			Id:            "registered2",
			MaxBlockSpace: math.LegacyMustNewDecFromStr("0.1"),
			Order:         1,
		},
	}

	for _, lane := range registeredLanes {
		s.Require().NoError(s.blocksdKeeper.AddLane(s.ctx, lane))
	}

	testCases := []struct {
		name     string
		query    *types.QueryLanesRequest
		expected []types.Lane
	}{
		{
			name:     "valid",
			query:    &types.QueryLanesRequest{},
			expected: registeredLanes,
		},
	}
	for _, tc := range testCases {
		s.Run(tc.name, func() {
			resp, err := s.queryServer.Lanes(s.ctx, tc.query)
			s.Require().NoError(err)
			s.Require().Equal(tc.expected, resp.GetLanes())
		})
	}
}
