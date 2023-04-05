package keeper_test

import (
	"math/rand"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	testutils "github.com/skip-mev/pob/testutils"
	"github.com/skip-mev/pob/x/builder/types"
)

func (suite *KeeperTestSuite) TestMsgAuctionBid() {
	rng := rand.New(rand.NewSource(time.Now().Unix()))
	accounts := testutils.RandomAccounts(rng, 4)

	bidder := accounts[0]
	escrow := accounts[1]

	proposerCons := accounts[2]
	proposerOperator := accounts[3]
	proposer := stakingtypes.Validator{
		OperatorAddress: sdk.ValAddress(proposerOperator.Address).String(),
	}

	testCases := []struct {
		name      string
		msg       *types.MsgAuctionBid
		malleate  func()
		expectErr bool
	}{
		{
			name: "invalid bidder address",
			msg: &types.MsgAuctionBid{
				Bidder: "foo",
			},
			malleate:  func() {},
			expectErr: true,
		},
		{
			name: "too many bundled transactions",
			msg: &types.MsgAuctionBid{
				Bidder:       bidder.Address.String(),
				Transactions: [][]byte{{0xFF}, {0xFF}, {0xFF}},
			},
			malleate: func() {
				params := types.DefaultParams()
				params.MaxBundleSize = 2
				suite.builderKeeper.SetParams(suite.ctx, params)
			},
			expectErr: true,
		},
		{
			name: "valid bundle with no proposer fee",
			msg: &types.MsgAuctionBid{
				Bidder:       bidder.Address.String(),
				Bid:          sdk.NewInt64Coin("foo", 1024),
				Transactions: [][]byte{{0xFF}, {0xFF}},
			},
			malleate: func() {
				params := types.DefaultParams()
				params.ProposerFee = sdk.ZeroDec()
				params.EscrowAccountAddress = escrow.Address.String()
				suite.builderKeeper.SetParams(suite.ctx, params)

				suite.bankKeeper.EXPECT().
					SendCoins(
						suite.ctx,
						bidder.Address,
						escrow.Address,
						sdk.NewCoins(sdk.NewInt64Coin("foo", 1024)),
					).
					Return(nil).
					AnyTimes()
			},
			expectErr: false,
		},
		{
			name: "valid bundle with proposer fee",
			msg: &types.MsgAuctionBid{
				Bidder:       bidder.Address.String(),
				Bid:          sdk.NewInt64Coin("foo", 3416),
				Transactions: [][]byte{{0xFF}, {0xFF}},
			},
			malleate: func() {
				params := types.DefaultParams()
				params.ProposerFee = sdk.MustNewDecFromStr("0.30")
				params.EscrowAccountAddress = escrow.Address.String()
				suite.builderKeeper.SetParams(suite.ctx, params)

				suite.distrKeeper.EXPECT().
					GetPreviousProposerConsAddr(suite.ctx).
					Return(proposerCons.ConsKey.PubKey().Address().Bytes())

				suite.stakingKeeper.EXPECT().
					ValidatorByConsAddr(suite.ctx, sdk.ConsAddress(proposerCons.ConsKey.PubKey().Address().Bytes())).
					Return(proposer).
					AnyTimes()

				suite.bankKeeper.EXPECT().
					SendCoins(suite.ctx, bidder.Address, proposerOperator.Address, sdk.NewCoins(sdk.NewInt64Coin("foo", 1024))).
					Return(nil)

				suite.bankKeeper.EXPECT().
					SendCoins(suite.ctx, bidder.Address, escrow.Address, sdk.NewCoins(sdk.NewInt64Coin("foo", 2392))).
					Return(nil)
			},
			expectErr: false,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			tc.malleate()

			_, err := suite.msgServer.AuctionBid(suite.ctx, tc.msg)
			if tc.expectErr {
				suite.Require().Error(err)
			} else {
				suite.Require().NoError(err)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestMsgUpdateParams() {
	suite.T().SkipNow()
}
