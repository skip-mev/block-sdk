package keeper_test

import (
	"cosmossdk.io/math"
	"github.com/skip-mev/chaintestutil/sample"

	testutils "github.com/skip-mev/block-sdk/v2/testutils"

	"github.com/skip-mev/block-sdk/v2/x/blocksdk/types"
)

func (s *KeeperTestSuite) TestMsgUpdateLane() {
	rng := sample.Rand()
	account := testutils.RandomAccounts(rng, 1)[0]

	// pre-register a lane
	registeredLane := types.Lane{
		Id:            "registered",
		MaxBlockSpace: math.LegacyMustNewDecFromStr("0.1"),
		Order:         0,
	}

	s.Require().NoError(s.blocksdKeeper.AddLane(s.ctx, registeredLane))
	testCases := []struct {
		name      string
		msg       *types.MsgUpdateLane
		pass      bool
		passBasic bool
	}{
		{
			name: "invalid authority",
			msg: &types.MsgUpdateLane{
				Authority: "invalid",
				Lane: types.Lane{
					Id:            "test",
					MaxBlockSpace: math.LegacyMustNewDecFromStr("0.1"),
					Order:         0,
				},
			},
			passBasic: false,
			pass:      false,
		},
		{
			name: "invalid unauthorized authority",
			msg: &types.MsgUpdateLane{
				Authority: account.Address.String(),
				Lane: types.Lane{
					Id:            "test",
					MaxBlockSpace: math.LegacyMustNewDecFromStr("0.1"),
					Order:         0,
				},
			},
			passBasic: true,
			pass:      false,
		},
		{
			name: "invalid lane ID",
			msg: &types.MsgUpdateLane{
				Authority: s.authorityAccount.String(),
				Lane: types.Lane{
					Id:            "",
					MaxBlockSpace: math.LegacyMustNewDecFromStr("0.1"),
					Order:         0,
				},
			},
			passBasic: false,
			pass:      false,
		},

		{
			name: "invalid MaxBlockSpace",
			msg: &types.MsgUpdateLane{
				Authority: s.authorityAccount.String(),
				Lane: types.Lane{
					Id:            "",
					MaxBlockSpace: math.LegacyMustNewDecFromStr("1.1"),
					Order:         0,
				},
			},
			passBasic: false,
			pass:      false,
		},
		{
			name: "invalid lane does not exist",
			msg: &types.MsgUpdateLane{
				Authority: s.authorityAccount.String(),
				Lane: types.Lane{
					Id:            "invalid",
					MaxBlockSpace: math.LegacyMustNewDecFromStr("0.1"),
					Order:         0,
				},
			},
			passBasic: true,
			pass:      false,
		},
		{
			name: "invalid order modification",
			msg: &types.MsgUpdateLane{
				Authority: s.authorityAccount.String(),
				Lane: types.Lane{
					Id:            "registered",
					MaxBlockSpace: math.LegacyMustNewDecFromStr("0.2"),
					Order:         1,
				},
			},
			passBasic: true,
			pass:      false,
		},
		{
			name: "valid",
			msg: &types.MsgUpdateLane{
				Authority: s.authorityAccount.String(),
				Lane: types.Lane{
					Id:            "registered",
					MaxBlockSpace: math.LegacyMustNewDecFromStr("0.2"),
					Order:         0,
				},
			},
			passBasic: true,
			pass:      true,
		},
	}
	for _, tc := range testCases {
		s.Run(tc.name, func() {
			if !tc.passBasic {
				s.Require().Error(tc.msg.ValidateBasic())
				return
			}

			_, err := s.msgServer.UpdateLane(s.ctx, tc.msg)
			if tc.pass {
				s.Require().NoError(err)
				lane, err := s.blocksdKeeper.GetLane(s.ctx, tc.msg.Lane.Id)
				s.Require().NoError(err)
				s.Require().Equal(tc.msg.Lane, lane)

			} else {
				s.Require().Error(err)
			}
		})
	}
}

func (s *KeeperTestSuite) TestMsgUpdateParams() {
	rng := sample.Rand()

	testCases := []struct {
		name string
		msg  *types.MsgUpdateParams

		pass      bool
		passBasic bool
	}{
		{
			name: "valid",
			msg: &types.MsgUpdateParams{
				Authority: s.authorityAccount.String(),
				Params: types.Params{
					Enabled: true,
				},
			},
			passBasic: true,
			pass:      true,
		},
		{
			name: "invalid authority address non bech32",
			msg: &types.MsgUpdateParams{
				Authority: "invalid",
				Params: types.Params{
					Enabled: true,
				},
			},
			passBasic: false,
			pass:      true,
		},
		{
			name: "invalid authority address",
			msg: &types.MsgUpdateParams{
				Authority: sample.Address(rng),
				Params: types.Params{
					Enabled: true,
				},
			},
			passBasic: true,
			pass:      false,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			if !tc.passBasic {
				s.Require().Error(tc.msg.ValidateBasic())
				return
			}

			_, err := s.msgServer.UpdateParams(s.ctx, tc.msg)
			if tc.pass {
				s.Require().NoError(err)
			} else {
				s.Require().Error(err)
			}
		})
	}
}
