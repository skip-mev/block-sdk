package integration_test

import (
	"context"
	"fmt"
	"testing"

	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

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
	s.T().Parallel()

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

func (s *NetworkTestSuite) TestFreeTxNoFees() {
	s.T().Parallel()

	val := s.NetworkSuite.Network.Validators[0]

	cc, closeConn, err := s.NetworkSuite.GetGRPC()
	s.Require().NoError(err)
	defer closeConn()

	bankClient := banktypes.NewQueryClient(cc)
	resp, err := bankClient.Balance(context.Background(), &banktypes.QueryBalanceRequest{
		Address: val.Address.String(),
		Denom:   s.NetworkSuite.Network.Config.BondDenom,
	})
	require.NoError(s.T(), err)
	originalBalance := resp.Balance.Amount

	coin := sdk.NewCoin(s.NetworkSuite.Network.Config.BondDenom, math.NewInt(10))
	txBz, err := s.NetworkSuite.CreateTxBytes(
		coin,
		999999999,
		[]sdk.Msg{
			&stakingtypes.MsgDelegate{
				DelegatorAddress: val.Address.String(),
				ValidatorAddress: val.ValAddress.String(),
				Amount:           coin,
			},
		},
	)
	bcastResp, err := val.RPCClient.BroadcastTxCommit(context.Background(), txBz)
	require.NoError(s.T(), err)
	require.Equal(s.T(), uint32(0), bcastResp.CheckTx.Code)
	require.Equal(s.T(), uint32(0), bcastResp.TxResult.Code)

	resp, err = bankClient.Balance(context.Background(), &banktypes.QueryBalanceRequest{
		Address: val.Address.String(),
		Denom:   s.NetworkSuite.Network.Config.BondDenom,
	})
	require.NoError(s.T(), err)
	require.Equal(s.T(), originalBalance, resp.Balance.Amount)
}
