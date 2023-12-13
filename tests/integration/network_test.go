package integration_test

import (
	"fmt"
	"testing"

	tmcli "github.com/cometbft/cometbft/libs/cli"
	clitestutil "github.com/cosmos/cosmos-sdk/testutil/cli"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc/status"

	"github.com/skip-mev/block-sdk/testutils/networksuite"
	auctioncli "github.com/skip-mev/block-sdk/x/auction/client/cli"
	auctiontypes "github.com/skip-mev/block-sdk/x/auction/types"
	blocksdkcli "github.com/skip-mev/block-sdk/x/blocksdk/client/cli"
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
	s.T().Parallel()

	val := s.Network.Validators[0]

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
			out, err := clitestutil.ExecTestCLICmd(val.ClientCtx, blocksdkcli.CmdQueryLanes(), tc.args)
			if tc.err != nil {
				stat, ok := status.FromError(tc.err)
				require.True(t, ok)
				require.ErrorIs(t, stat.Err(), tc.err)
			} else {
				require.NoError(t, err)
				var resp blocksdktypes.QueryLanesResponse
				require.NoError(t, s.Network.Config.Codec.UnmarshalJSON(out.Bytes(), &resp))
				require.NotNil(t, resp.Lanes)
				require.ElementsMatch(t, tc.obj, resp.Lanes)
			}
		})
	}
}

func (s *NetworkTestSuite) TestGetAuctionParams() {
	s.T().Parallel()

	val := s.Network.Validators[0]

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
			out, err := clitestutil.ExecTestCLICmd(val.ClientCtx, auctioncli.CmdQueryParams(), tc.args)
			if tc.err != nil {
				stat, ok := status.FromError(tc.err)
				require.True(t, ok)
				require.ErrorIs(t, stat.Err(), tc.err)
			} else {
				require.NoError(t, err)
				var resp auctiontypes.QueryParamsResponse
				require.NoError(t, s.Network.Config.Codec.UnmarshalJSON(out.Bytes(), &resp.Params))
				require.NotNil(t, resp)
				require.Equal(t, tc.obj, resp.Params)
			}
		})
	}
}
