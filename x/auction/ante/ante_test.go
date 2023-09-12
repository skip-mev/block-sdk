package ante_test

import (
	"math/rand"
	"os"
	"testing"
	"time"

	"cosmossdk.io/math"
	"github.com/cometbft/cometbft/libs/log"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/golang/mock/gomock"
	"github.com/skip-mev/block-sdk/block"
	"github.com/skip-mev/block-sdk/block/base"
	defaultlane "github.com/skip-mev/block-sdk/lanes/base"
	"github.com/skip-mev/block-sdk/lanes/mev"
	testutils "github.com/skip-mev/block-sdk/testutils"
<<<<<<< HEAD:x/builder/ante/ante_test.go
	"github.com/skip-mev/block-sdk/x/builder/ante"
	"github.com/skip-mev/block-sdk/x/builder/keeper"
	buildertypes "github.com/skip-mev/block-sdk/x/builder/types"
	"github.com/stretchr/testify/suite"
=======
	"github.com/skip-mev/block-sdk/x/auction/ante"
	"github.com/skip-mev/block-sdk/x/auction/keeper"
	auctiontypes "github.com/skip-mev/block-sdk/x/auction/types"
>>>>>>> 3c6f319 (feat(docs): rename x/builder -> x/auction (#55)):x/auction/ante/ante_test.go
)

type AnteTestSuite struct {
	suite.Suite
	ctx sdk.Context

	encodingConfig testutils.EncodingConfig
	random         *rand.Rand

	// auction setup
	auctionkeeper    keeper.Keeper
	bankKeeper       *testutils.MockBankKeeper
	accountKeeper    *testutils.MockAccountKeeper
	distrKeeper      *testutils.MockDistributionKeeper
	stakingKeeper    *testutils.MockStakingKeeper
	auctionDecorator ante.AuctionDecorator
	key              *storetypes.KVStoreKey
	authorityAccount sdk.AccAddress

	// mempool and lane set up
	mempool  block.Mempool
	mevLane  *mev.MEVLane
	baseLane *defaultlane.DefaultLane
	lanes    []block.Lane

	// Account set up
	balance sdk.Coin
}

func TestAnteTestSuite(t *testing.T) {
	suite.Run(t, new(AnteTestSuite))
}

func (suite *AnteTestSuite) SetupTest() {
	// General config
	suite.encodingConfig = testutils.CreateTestEncodingConfig()
	suite.random = rand.New(rand.NewSource(time.Now().Unix()))
	suite.key = storetypes.NewKVStoreKey(auctiontypes.StoreKey)
	testCtx := testutil.DefaultContextWithDB(suite.T(), suite.key, storetypes.NewTransientStoreKey("transient_test"))
	suite.ctx = testCtx.Ctx.WithIsCheckTx(true)

	// Keepers set up
	ctrl := gomock.NewController(suite.T())
	suite.accountKeeper = testutils.NewMockAccountKeeper(ctrl)
	suite.accountKeeper.EXPECT().GetModuleAddress(auctiontypes.ModuleName).Return(sdk.AccAddress{}).AnyTimes()
	suite.bankKeeper = testutils.NewMockBankKeeper(ctrl)
	suite.distrKeeper = testutils.NewMockDistributionKeeper(ctrl)
	suite.stakingKeeper = testutils.NewMockStakingKeeper(ctrl)
	suite.authorityAccount = sdk.AccAddress([]byte("authority"))
	suite.auctionkeeper = keeper.NewKeeper(
		suite.encodingConfig.Codec,
		suite.key,
		suite.accountKeeper,
		suite.bankKeeper,
		suite.distrKeeper,
		suite.stakingKeeper,
		suite.authorityAccount.String(),
	)
	err := suite.auctionkeeper.SetParams(suite.ctx, auctiontypes.DefaultParams())
	suite.Require().NoError(err)

	// Lanes configuration
	//
	// TOB lane set up
	mevConfig := base.LaneConfig{
		Logger:        log.NewTMLogger(os.Stdout),
		TxEncoder:     suite.encodingConfig.TxConfig.TxEncoder(),
		TxDecoder:     suite.encodingConfig.TxConfig.TxDecoder(),
		AnteHandler:   suite.anteHandler,
		MaxBlockSpace: math.LegacyZeroDec(),
	}
	suite.mevLane = mev.NewMEVLane(
		mevConfig,
		mev.NewDefaultAuctionFactory(suite.encodingConfig.TxConfig.TxDecoder()),
	)

	// Base lane set up
	baseConfig := base.LaneConfig{
		Logger:        log.NewTMLogger(os.Stdout),
		TxEncoder:     suite.encodingConfig.TxConfig.TxEncoder(),
		TxDecoder:     suite.encodingConfig.TxConfig.TxDecoder(),
		AnteHandler:   suite.anteHandler,
		MaxBlockSpace: math.LegacyZeroDec(),
		IgnoreList:    []block.Lane{suite.mevLane},
	}
	suite.baseLane = defaultlane.NewDefaultLane(baseConfig)

	// Mempool set up
	suite.lanes = []block.Lane{suite.mevLane, suite.baseLane}
	suite.mempool = block.NewLanedMempool(log.NewTMLogger(os.Stdout), true, suite.lanes...)
}

func (suite *AnteTestSuite) anteHandler(ctx sdk.Context, tx sdk.Tx, _ bool) (sdk.Context, error) {
	suite.bankKeeper.EXPECT().GetBalance(ctx, gomock.Any(), suite.balance.Denom).AnyTimes().Return(suite.balance)

	next := func(ctx sdk.Context, tx sdk.Tx, _ bool) (sdk.Context, error) {
		return ctx, nil
	}

	return suite.auctionDecorator.AnteHandle(ctx, tx, false, next)
}

func (suite *AnteTestSuite) TestAnteHandler() {
	var (
		// Bid set up
		bidder  = testutils.RandomAccounts(suite.random, 1)[0]
		bid     = sdk.NewCoin("stake", math.NewInt(1000))
		balance = sdk.NewCoin("stake", math.NewInt(10000))
		signers = []testutils.Account{bidder}

		// Top bidding auction tx set up
		topBidder    = testutils.RandomAccounts(suite.random, 1)[0]
		topBid       = sdk.NewCoin("stake", math.NewInt(100))
		insertTopBid = true
		timeout      = uint64(1000)

		// Auction setup
		maxBundleSize          uint32 = 5
		reserveFee                    = sdk.NewCoin("stake", math.NewInt(100))
		minBidIncrement               = sdk.NewCoin("stake", math.NewInt(100))
		frontRunningProtection        = true
	)

	cases := []struct {
		name     string
		malleate func()
		pass     bool
	}{
		{
			"empty mempool, valid bid",
			func() {
				insertTopBid = false
			},
			true,
		},
		{
			"smaller bid than winning bid, invalid auction tx",
			func() {
				insertTopBid = true
				topBid = sdk.NewCoin("stake", math.NewInt(100000))
			},
			false,
		},
		{
			"bidder has insufficient balance, invalid auction tx",
			func() {
				insertTopBid = false
				balance = sdk.NewCoin("stake", math.NewInt(10))
			},
			false,
		},
		{
			"bid is smaller than reserve fee, invalid auction tx",
			func() {
				balance = sdk.NewCoin("stake", math.NewInt(10000))
				bid = sdk.NewCoin("stake", math.NewInt(101))
				reserveFee = sdk.NewCoin("stake", math.NewInt(1000))
			},
			false,
		},
		{
			"valid mev bid tx",
			func() {
				balance = sdk.NewCoin("stake", math.NewInt(10000))
				bid = sdk.NewCoin("stake", math.NewInt(1000))
				reserveFee = sdk.NewCoin("stake", math.NewInt(100))
			},
			true,
		},
		{
			"invalid mev bid tx with no timeout",
			func() {
				timeout = 0
			},
			false,
		},
		{
			"mev tx is the top bidding tx",
			func() {
				timeout = 1000
				balance = sdk.NewCoin("stake", math.NewInt(10000))
				bid = sdk.NewCoin("stake", math.NewInt(1000))
				reserveFee = sdk.NewCoin("stake", math.NewInt(100))

				insertTopBid = true
				topBidder = bidder
				topBid = bid
				signers = []testutils.Account{}
			},
			true,
		},
		{
			"invalid frontrunning mev bid tx",
			func() {
				randomAccount := testutils.RandomAccounts(suite.random, 2)
				bidder := randomAccount[0]
				otherUser := randomAccount[1]
				insertTopBid = false

				signers = []testutils.Account{bidder, otherUser}
			},
			false,
		},
		{
			"valid frontrunning mev bid tx",
			func() {
				randomAccount := testutils.RandomAccounts(suite.random, 2)
				bidder := randomAccount[0]
				otherUser := randomAccount[1]

				signers = []testutils.Account{bidder, otherUser}
				frontRunningProtection = false
			},
			true,
		},
		{
			"invalid sandwiching mev bid tx",
			func() {
				randomAccount := testutils.RandomAccounts(suite.random, 2)
				bidder := randomAccount[0]
				otherUser := randomAccount[1]

				signers = []testutils.Account{bidder, otherUser, bidder}
				frontRunningProtection = true
			},
			false,
		},
		{
			"invalid mev bid tx with many signers",
			func() {
				signers = testutils.RandomAccounts(suite.random, 10)
				frontRunningProtection = true
			},
			false,
		},
	}

	for _, tc := range cases {
		suite.Run(tc.name, func() {
			suite.SetupTest()
			tc.malleate()

			suite.ctx = suite.ctx.WithBlockHeight(1)

			// Set the mev params
			err := suite.auctionkeeper.SetParams(suite.ctx, auctiontypes.Params{
				MaxBundleSize:          maxBundleSize,
				ReserveFee:             reserveFee,
				MinBidIncrement:        minBidIncrement,
				FrontRunningProtection: frontRunningProtection,
			})
			suite.Require().NoError(err)

			// Insert the top bid into the mempool
			if insertTopBid {
				topAuctionTx, err := testutils.CreateAuctionTxWithSigners(suite.encodingConfig.TxConfig, topBidder, topBid, 0, timeout, []testutils.Account{})
				suite.Require().NoError(err)

				distribution := suite.mempool.GetTxDistribution()
				suite.Require().Equal(0, distribution[mev.LaneName])
				suite.Require().Equal(0, distribution[defaultlane.LaneName])

				suite.Require().NoError(suite.mempool.Insert(suite.ctx, topAuctionTx))

				distribution = suite.mempool.GetTxDistribution()
				suite.Require().Equal(1, distribution[mev.LaneName])
				suite.Require().Equal(0, distribution[defaultlane.LaneName])
			}

			// Create the actual mev tx and insert into the mempool
			mevTx, err := testutils.CreateAuctionTxWithSigners(suite.encodingConfig.TxConfig, bidder, bid, 0, timeout, signers)
			suite.Require().NoError(err)

			// Execute the ante handler
			suite.balance = balance
			suite.auctionDecorator = ante.NewAuctionDecorator(suite.auctionkeeper, suite.encodingConfig.TxConfig.TxEncoder(), suite.mevLane, suite.mempool)
			_, err = suite.anteHandler(suite.ctx, mevTx, false)
			if tc.pass {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}
