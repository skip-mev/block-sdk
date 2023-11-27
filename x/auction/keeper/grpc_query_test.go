package keeper_test

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/skip-mev/block-sdk/x/auction/types"
)

func (s *KeeperTestSuite) TestQueryParams() {
	s.Run("can query module params", func() {
		params, err := s.auctionkeeper.GetParams(s.ctx)
		s.Require().NoError(err)

		escrowAddress := sdk.AccAddress(params.EscrowAccountAddress)

		res, err := s.queryServer.Params(s.ctx, &types.QueryParamsRequest{})
		s.Require().NoError(err)
		s.Require().Equal(params, res.Params)
		s.Require().Equal(escrowAddress.String(), res.EscrowAddressString)
	})
}
