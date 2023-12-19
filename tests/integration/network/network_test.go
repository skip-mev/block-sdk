package integration_test

import (
	"context"
	"fmt"
	"testing"

	tmcli "github.com/cometbft/cometbft/libs/cli"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc/status"

	"github.com/skip-mev/block-sdk/testutils/networksuite"
	auctiontypes "github.com/skip-mev/block-sdk/x/auction/types"
	blocksdktypes "github.com/skip-mev/block-sdk/x/blocksdk/types"
)

// NetworkTestSuite is a test suite for network integration tests.
type NetworkTestSuite struct {
	networksuite.NetworkTestSuite
}

// TestQueryTestSuite runs test of network integration tests.
func TestNetworkTestSuite(t *testing.T) {
	suite.Run(t, new(NetworkTestSuite))
}

func (s *NetworkTestSuite) TestGetLanes() {
	common := []string{
		fmt.Sprintf("--%s=json", tmcli.OutputFlag),
	}
	for _, tc := range []struct {
		name string

		args []string
		err  error
		obj  []blocksdktypes.Lane
	}{
		{
			name: "should return default lanes",
			args: common,
			obj:  s.BlockSDKState.Lanes,
		},
	} {
		s.T().Run(tc.name, func(t *testing.T) {
			tc := tc
			resp, err := s.QueryBlockSDKLanes()
			if tc.err != nil {
				stat, ok := status.FromError(tc.err)
				require.True(t, ok)
				require.ErrorIs(t, stat.Err(), tc.err)
			} else {
				require.NoError(t, err)
				require.NotNil(t, resp.Lanes)
				require.ElementsMatch(t, tc.obj, resp.Lanes)
			}
		})
	}
}

func (s *NetworkTestSuite) TestGetAuctionParams() {
	s.T().Parallel()

	common := []string{
		fmt.Sprintf("--%s=json", tmcli.OutputFlag),
	}
	for _, tc := range []struct {
		name string

		args []string
		err  error
		obj  auctiontypes.Params
	}{
		{
			name: "should return default params",
			args: common,
			obj:  auctiontypes.DefaultParams(),
		},
	} {
		s.T().Run(tc.name, func(t *testing.T) {
			tc := tc
			resp, err := s.QueryAuctionParams()
			if tc.err != nil {
				stat, ok := status.FromError(tc.err)
				require.True(t, ok)
				require.ErrorIs(t, stat.Err(), tc.err)
			} else {
				require.NoError(t, err)
				require.NotNil(t, resp)
				require.Equal(t, tc.obj, resp.Params)
			}
		})
	}
}

func (s *NetworkTestSuite) QueryAuctionParams() (*auctiontypes.QueryParamsResponse, error) {
	s.T().Helper()

	cc, closeConn, err := s.NetworkSuite.GetGRPC()
	s.Require().NoError(err)
	defer closeConn()

	client := auctiontypes.NewQueryClient(cc)
	return client.Params(context.Background(), &auctiontypes.QueryParamsRequest{})
}

func (s *NetworkTestSuite) QueryBlockSDKLanes() (*blocksdktypes.QueryLanesResponse, error) {
	s.T().Helper()

	cc, closeConn, err := s.NetworkSuite.GetGRPC()
	s.Require().NoError(err)
	defer closeConn()

	client := blocksdktypes.NewQueryClient(cc)
	return client.Lanes(context.Background(), &blocksdktypes.QueryLanesRequest{})
}
