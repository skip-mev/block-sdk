package keeper_test

import (
	"testing"

	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"

	testutils "github.com/skip-mev/block-sdk/testutils"
	"github.com/skip-mev/block-sdk/x/auction/keeper"
	"github.com/skip-mev/block-sdk/x/auction/types"
	"github.com/skip-mev/block-sdk/x/auction/types/mocks"

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
	msgServer        types.MsgServer
	key              *storetypes.KVStoreKey
	authorityAccount sdk.AccAddress
}

func TestKeeperTestSuite(t *testing.T) {
	suite.Run(t, new(KeeperTestSuite))
}

func (suite *KeeperTestSuite) SetupTest() {
	suite.encCfg = testutils.CreateTestEncodingConfig()
	suite.key = storetypes.NewKVStoreKey(types.StoreKey)
	testCtx := testutil.DefaultContextWithDB(suite.T(), suite.key, storetypes.NewTransientStoreKey("transient_test"))
	suite.ctx = testCtx.Ctx

	suite.accountKeeper = mocks.NewAccountKeeper(suite.T())
	suite.accountKeeper.On("GetModuleAddress", types.ModuleName).Return(sdk.AccAddress{}).Maybe()

	suite.bankKeeper = mocks.NewBankKeeper(suite.T())
	suite.distrKeeper = mocks.NewDistributionKeeper(suite.T())
	suite.stakingKeeper = mocks.NewStakingKeeper(suite.T())
	suite.authorityAccount = sdk.AccAddress([]byte("authority"))
	suite.auctionkeeper = keeper.NewKeeper(
		suite.encCfg.Codec,
		suite.key,
		suite.accountKeeper,
		suite.bankKeeper,
		suite.distrKeeper,
		suite.stakingKeeper,
		suite.authorityAccount.String(),
	)

	err := suite.auctionkeeper.SetParams(suite.ctx, types.DefaultParams())
	suite.Require().NoError(err)

	suite.msgServer = keeper.NewMsgServerImpl(suite.auctionkeeper)
}
