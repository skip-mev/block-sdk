package keeper_test

import (
	"testing"

	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"

	testutils "github.com/skip-mev/block-sdk/v2/testutils"
	"github.com/skip-mev/block-sdk/v2/x/auction/keeper"
	"github.com/skip-mev/block-sdk/v2/x/auction/types"
	"github.com/skip-mev/block-sdk/v2/x/auction/types/mocks"

	"github.com/stretchr/testify/suite"
)

type KeeperTestSuite struct {
	suite.Suite

	auctionkeeper    keeper.Keeper
	bankKeeper       *mocks.BankKeeper
	accountKeeper    *mocks.AccountKeeper
	distrKeeper      *mocks.DistributionKeeper
	stakingKeeper    *mocks.StakingKeeper
	encCfg           testutils.EncodingConfig
	ctx              sdk.Context
	key              *storetypes.KVStoreKey
	authorityAccount sdk.AccAddress

	msgServer   types.MsgServer
	queryServer types.QueryServer
}

func TestKeeperTestSuite(t *testing.T) {
	suite.Run(t, new(KeeperTestSuite))
}

func (s *KeeperTestSuite) SetupTest() {
	s.encCfg = testutils.CreateTestEncodingConfig()
	s.key = storetypes.NewKVStoreKey(types.StoreKey)
	testCtx := testutil.DefaultContextWithDB(s.T(), s.key, storetypes.NewTransientStoreKey("transient_test"))
	s.ctx = testCtx.Ctx

	s.accountKeeper = mocks.NewAccountKeeper(s.T())
	s.accountKeeper.On("GetModuleAddress", types.ModuleName).Return(sdk.AccAddress{}).Maybe()

	s.bankKeeper = mocks.NewBankKeeper(s.T())
	s.distrKeeper = mocks.NewDistributionKeeper(s.T())
	s.stakingKeeper = mocks.NewStakingKeeper(s.T())
	s.authorityAccount = sdk.AccAddress([]byte("authority"))
	s.auctionkeeper = keeper.NewKeeper(
		s.encCfg.Codec,
		s.key,
		s.accountKeeper,
		s.bankKeeper,
		s.distrKeeper,
		s.stakingKeeper,
		s.authorityAccount.String(),
	)

	err := s.auctionkeeper.SetParams(s.ctx, types.DefaultParams())
	s.Require().NoError(err)

	s.msgServer = keeper.NewMsgServerImpl(s.auctionkeeper)
	s.queryServer = keeper.NewQueryServer(s.auctionkeeper)
}
