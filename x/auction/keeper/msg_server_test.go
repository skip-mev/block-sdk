package keeper_test

import (
	"math/rand"
	"time"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	testutils "github.com/skip-mev/block-sdk/testutils"
	"github.com/skip-mev/block-sdk/x/auction/types"
)

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
