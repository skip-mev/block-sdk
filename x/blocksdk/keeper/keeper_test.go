package keeper_test

import (
	"testing"

	"cosmossdk.io/math"

	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/stretchr/testify/suite"

	testutils "github.com/skip-mev/block-sdk/v2/testutils"
	"github.com/skip-mev/block-sdk/v2/x/blocksdk/keeper"
	"github.com/skip-mev/block-sdk/v2/x/blocksdk/types"
)

type KeeperTestSuite struct {
	suite.Suite

	blocksdKeeper    keeper.Keeper
	encCfg           testutils.EncodingConfig
	ctx              sdk.Context
	queryServer      types.QueryServer
	msgServer        types.MsgServer
	key              *storetypes.KVStoreKey
	authorityAccount sdk.AccAddress
}

func TestKeeperTestSuite(t *testing.T) {
	suite.Run(t, new(KeeperTestSuite))
}

func (s *KeeperTestSuite) SetupTest() {
	s.encCfg = testutils.CreateTestEncodingConfig()
	s.key = storetypes.NewKVStoreKey(types.StoreKey)
	testCtx := testutil.DefaultContextWithDB(s.T(), s.key, storetypes.NewTransientStoreKey("transient_test"))
	s.ctx = testCtx.Ctx
	s.authorityAccount = []byte("authority")
	s.blocksdKeeper = keeper.NewKeeper(
		s.encCfg.Codec,
		s.key,
		s.authorityAccount.String(),
	)

	s.msgServer = keeper.NewMsgServerImpl(s.blocksdKeeper)
	s.queryServer = keeper.NewQueryServer(s.blocksdKeeper)
}

func (s *KeeperTestSuite) TestLane() {
	const (
		validLaneID1  = "test1"
		validLaneID2  = "test2"
		validLaneID3  = "test3"
		invalidLaneID = "invalid"
	)

	lanes := []types.Lane{
		{
			Id:            validLaneID1,
			MaxBlockSpace: math.LegacyMustNewDecFromStr("0.10"),
			Order:         0,
		},
		{
			Id:            validLaneID2,
			MaxBlockSpace: math.LegacyMustNewDecFromStr("0.10"),
			Order:         1,
		},
	}

	for _, lane := range lanes {
		err := s.blocksdKeeper.AddLane(s.ctx, lane)
		s.Require().NoError(err)
	}

	s.Run("get lane valid", func() {
		gotLane, err := s.blocksdKeeper.GetLane(s.ctx, validLaneID1)
		s.Require().NoError(err)
		s.Require().Equal(lanes[0], gotLane)
	})

	s.Run("get lane invalid", func() {
		_, err := s.blocksdKeeper.GetLane(s.ctx, invalidLaneID)
		s.Require().Error(err)
	})

	s.Run("get lanes", func() {
		gotLanes := s.blocksdKeeper.GetLanes(s.ctx)
		s.Require().Equal(lanes, gotLanes)
	})

	s.Run("add invalid duplicate lane order", func() {
		invalidLane := types.Lane{
			Id:            validLaneID3,
			MaxBlockSpace: math.LegacyMustNewDecFromStr("0.10"),
			Order:         0,
		}

		err := s.blocksdKeeper.AddLane(s.ctx, invalidLane)
		s.Require().Error(err)
	})

	s.Run("add invalid duplicate lane ID", func() {
		invalidLane := types.Lane{
			Id:            validLaneID1,
			MaxBlockSpace: math.LegacyMustNewDecFromStr("0.10"),
			Order:         2,
		}

		err := s.blocksdKeeper.AddLane(s.ctx, invalidLane)
		s.Require().Error(err)
	})

	s.Run("add invalid non-monotonic order", func() {
		invalidLane := types.Lane{
			Id:            validLaneID3,
			MaxBlockSpace: math.LegacyMustNewDecFromStr("0.10"),
			Order:         3,
		}

		err := s.blocksdKeeper.AddLane(s.ctx, invalidLane)
		s.Require().Error(err)
	})

	s.Run("delete valid", func() {
		s.blocksdKeeper.DeleteLane(s.ctx, validLaneID1)
		_, err := s.blocksdKeeper.GetLane(s.ctx, validLaneID1)
		s.Require().Error(err)
	})
}
