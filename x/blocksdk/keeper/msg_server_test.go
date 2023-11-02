package keeper_test

import (
	"math/rand"
	"time"

	"cosmossdk.io/math"

	testutils "github.com/skip-mev/block-sdk/testutils"

	"github.com/skip-mev/block-sdk/x/blocksdk/types"
)

func (s *KeeperTestSuite) TestMsgUpdateLane() {
	rng := rand.New(rand.NewSource(time.Now().Unix()))
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
