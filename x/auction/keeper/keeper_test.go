package keeper_test

import (
	"testing"

	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/golang/mock/gomock"
	"github.com/skip-mev/pob/mempool"
	testutils "github.com/skip-mev/pob/testutils"
	"github.com/skip-mev/pob/x/auction/keeper"
	"github.com/skip-mev/pob/x/auction/types"

	"github.com/stretchr/testify/suite"
)

type KeeperTestSuite struct {
	suite.Suite

	auctionKeeper    keeper.Keeper
	bankKeeper       *testutils.MockBankKeeper
	accountKeeper    *testutils.MockAccountKeeper
	distrKeeper      *testutils.MockDistributionKeeper
	stakingKeeper    *testutils.MockStakingKeeper
	encCfg           testutils.EncodingConfig
	ctx              sdk.Context
	msgServer        types.MsgServer
	key              *storetypes.KVStoreKey
	authorityAccount sdk.AccAddress

	mempool *mempool.AuctionMempool
}

func TestKeeperTestSuite(t *testing.T) {
	suite.Run(t, new(KeeperTestSuite))
}

func (suite *KeeperTestSuite) SetupTest() {
	suite.encCfg = testutils.CreateTestEncodingConfig()
	suite.key = storetypes.NewKVStoreKey(types.StoreKey)
	testCtx := testutil.DefaultContextWithDB(suite.T(), suite.key, storetypes.NewTransientStoreKey("transient_test"))
	suite.ctx = testCtx.Ctx

	ctrl := gomock.NewController(suite.T())

	suite.accountKeeper = testutils.NewMockAccountKeeper(ctrl)
	suite.accountKeeper.EXPECT().GetModuleAddress(types.ModuleName).Return(sdk.AccAddress{}).AnyTimes()

	suite.bankKeeper = testutils.NewMockBankKeeper(ctrl)
	suite.distrKeeper = testutils.NewMockDistributionKeeper(ctrl)
	suite.stakingKeeper = testutils.NewMockStakingKeeper(ctrl)
	suite.authorityAccount = sdk.AccAddress([]byte("authority"))
	suite.auctionKeeper = keeper.NewKeeper(
		suite.encCfg.Codec,
		suite.key,
		suite.accountKeeper,
		suite.bankKeeper,
		suite.distrKeeper,
		suite.stakingKeeper,
		suite.authorityAccount.String(),
	)

	err := suite.auctionKeeper.SetParams(suite.ctx, types.DefaultParams())
	suite.Require().NoError(err)

	suite.mempool = mempool.NewAuctionMempool(suite.encCfg.TxConfig.TxDecoder(), 0)
	suite.msgServer = keeper.NewMsgServerImpl(suite.auctionKeeper)
}
