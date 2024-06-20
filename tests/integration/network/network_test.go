package integration_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/skip-mev/chaintestutil/network"

	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	tmcli "github.com/cometbft/cometbft/libs/cli"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc/status"

	"github.com/skip-mev/block-sdk/v2/testutils/networksuite"
	auctiontypes "github.com/skip-mev/block-sdk/v2/x/auction/types"
)

// NetworkTestSuite is a test suite for network integration tests.
type NetworkTestSuite struct {
	networksuite.NetworkTestSuite
}

// TestQueryTestSuite runs test of network integration tests.
func TestNetworkTestSuite(t *testing.T) {
	suite.Run(t, new(NetworkTestSuite))
}

func (s *NetworkTestSuite) TestGetAuctionParams() {
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

func (s *NetworkTestSuite) TestFreeTxNoFees() {
	val := s.NetworkSuite.Network.Validators[0]
	acc := *s.Accounts[0]

	cc, closeConn, err := s.NetworkSuite.GetGRPC()
	s.Require().NoError(err)
	defer closeConn()

	// Get original acc balance
	bankClient := banktypes.NewQueryClient(cc)
	resp, err := bankClient.Balance(context.Background(), &banktypes.QueryBalanceRequest{
		Address: acc.Address().String(),
		Denom:   s.NetworkSuite.Network.Config.BondDenom,
	})
	require.NoError(s.T(), err)
	originalBalance := resp.Balance.Amount

	// Send a free tx (delegation)
	coin := sdk.NewCoin(s.NetworkSuite.Network.Config.BondDenom, math.NewInt(10))
	txBz, err := s.NetworkSuite.CreateTxBytes(
		context.Background(),
		network.TxGenInfo{
			Account:       acc,
			GasLimit:      999999999,
			TimeoutHeight: 999999999,
			Fee:           sdk.NewCoins(coin),
		},
		&stakingtypes.MsgDelegate{
			DelegatorAddress: acc.Address().String(),
			ValidatorAddress: val.ValAddress.String(),
			Amount:           coin,
		},
	)
	require.NoError(s.T(), err)
	bcastResp, err := val.RPCClient.BroadcastTxCommit(context.Background(), txBz)
	require.NoError(s.T(), err)
	require.Equal(s.T(), uint32(0), bcastResp.CheckTx.Code)
	require.Equal(s.T(), uint32(0), bcastResp.TxResult.Code)

	// Get updated acc balance
	resp, err = bankClient.Balance(context.Background(), &banktypes.QueryBalanceRequest{
		Address: acc.Address().String(),
		Denom:   s.NetworkSuite.Network.Config.BondDenom,
	})
	require.NoError(s.T(), err)
	// Assert update acc balance is equal to original balance less the delegation
	require.Equal(s.T(), originalBalance.Sub(coin.Amount), resp.Balance.Amount)
}
