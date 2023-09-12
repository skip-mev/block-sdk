package keeper_test

import (
	"math/rand"
	"time"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	testutils "github.com/skip-mev/block-sdk/testutils"
	"github.com/skip-mev/block-sdk/x/auction/types"
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
				Bidder: "stake",
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
				suite.auctionkeeper.SetParams(suite.ctx, params)
			},
			expectErr: true,
		},
		{
			name: "valid bundle with no proposer fee",
			msg: &types.MsgAuctionBid{
				Bidder:       bidder.Address.String(),
				Bid:          sdk.NewInt64Coin("stake", 1024),
				Transactions: [][]byte{{0xFF}, {0xFF}},
			},
			malleate: func() {
				params := types.DefaultParams()
				params.ProposerFee = math.LegacyZeroDec()
				params.EscrowAccountAddress = escrow.Address
				suite.auctionkeeper.SetParams(suite.ctx, params)

				suite.bankKeeper.EXPECT().
					SendCoins(
						suite.ctx,
						bidder.Address,
						escrow.Address,
						sdk.NewCoins(sdk.NewInt64Coin("stake", 1024)),
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
				Bid:          sdk.NewInt64Coin("stake", 3416),
				Transactions: [][]byte{{0xFF}, {0xFF}},
			},
			malleate: func() {
				params := types.DefaultParams()
				params.ProposerFee = math.LegacyMustNewDecFromStr("0.30")
				params.EscrowAccountAddress = escrow.Address
				suite.auctionkeeper.SetParams(suite.ctx, params)

				suite.distrKeeper.EXPECT().
					GetPreviousProposerConsAddr(suite.ctx).
					Return(proposerCons.ConsKey.PubKey().Address().Bytes(), nil)

				suite.stakingKeeper.EXPECT().
					GetValidatorByConsAddr(suite.ctx, sdk.ConsAddress(proposerCons.ConsKey.PubKey().Address().Bytes())).
					Return(proposer, nil).
					AnyTimes()

				suite.bankKeeper.EXPECT().
					SendCoins(suite.ctx, bidder.Address, proposerOperator.Address, sdk.NewCoins(sdk.NewInt64Coin("stake", 1024))).
					Return(nil).AnyTimes()

				suite.bankKeeper.EXPECT().
					SendCoins(suite.ctx, bidder.Address, escrow.Address, sdk.NewCoins(sdk.NewInt64Coin("stake", 2392))).
					Return(nil).AnyTimes()
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
	rng := rand.New(rand.NewSource(time.Now().Unix()))
	account := testutils.RandomAccounts(rng, 1)[0]

	testCases := []struct {
		name string
		msg  *types.MsgUpdateParams

		pass      bool
		passBasic bool
	}{
		{
			name: "invalid proposer fee",
			msg: &types.MsgUpdateParams{
				Authority: suite.authorityAccount.String(),
				Params: types.Params{
					ProposerFee: math.LegacyMustNewDecFromStr("1.1"),
				},
			},
			passBasic: false,
			pass:      true,
		},
		{
			name: "invalid auction fees",
			msg: &types.MsgUpdateParams{
				Authority: suite.authorityAccount.String(),
				Params: types.Params{
					ProposerFee: math.LegacyMustNewDecFromStr("0.1"),
				},
			},
			passBasic: false,
			pass:      true,
		},
		{
			name: "invalid authority address",
			msg: &types.MsgUpdateParams{
				Authority: account.Address.String(),
				Params: types.Params{
					ProposerFee:          math.LegacyMustNewDecFromStr("0.1"),
					MaxBundleSize:        2,
					EscrowAccountAddress: suite.authorityAccount,
					MinBidIncrement:      sdk.NewInt64Coin("stake", 100),
					ReserveFee:           sdk.NewInt64Coin("stake", 100),
				},
			},
			passBasic: true,
			pass:      false,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			if !tc.passBasic {
				suite.Require().Error(tc.msg.ValidateBasic())
			}

			_, err := suite.msgServer.UpdateParams(suite.ctx, tc.msg)
			if tc.pass {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}
