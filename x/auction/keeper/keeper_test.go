package keeper_test

import (
	"testing"

	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/golang/mock/gomock"
	"github.com/skip-mev/pob/mempool"
	"github.com/skip-mev/pob/x/auction/ante"
	"github.com/skip-mev/pob/x/auction/keeper"
	"github.com/skip-mev/pob/x/auction/types"

	"github.com/stretchr/testify/suite"
)

type KeeperTestSuite struct {
	suite.Suite

	auctionKeeper    keeper.Keeper
	bankKeeper       *MockBankKeeper
	accountKeeper    *MockAccountKeeper
	distrKeeper      *MockDistributionKeeper
	stakingKeeper    *MockStakingKeeper
	encCfg           encodingConfig
	AuctionDecorator sdk.AnteDecorator
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
	suite.encCfg = createTestEncodingConfig()
	suite.key = sdk.NewKVStoreKey(types.StoreKey)
	testCtx := testutil.DefaultContextWithDB(suite.T(), suite.key, sdk.NewTransientStoreKey("transient_test"))
	suite.ctx = testCtx.Ctx

	ctrl := gomock.NewController(suite.T())

	suite.accountKeeper = NewMockAccountKeeper(ctrl)
	suite.accountKeeper.EXPECT().GetModuleAddress(types.ModuleName).Return(sdk.AccAddress{}).AnyTimes()

	suite.bankKeeper = NewMockBankKeeper(ctrl)
	suite.distrKeeper = NewMockDistributionKeeper(ctrl)
	suite.stakingKeeper = NewMockStakingKeeper(ctrl)
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
	suite.AuctionDecorator = ante.NewAuctionDecorator(suite.auctionKeeper, suite.encCfg.TxConfig.TxDecoder(), suite.mempool)
	suite.msgServer = keeper.NewMsgServerImpl(suite.auctionKeeper)
}
