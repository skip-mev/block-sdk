package ante_test

import (
	"math/rand"
	"testing"
	"time"

	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/golang/mock/gomock"
	"github.com/skip-mev/pob/mempool"
	"github.com/skip-mev/pob/x/auction/ante"
	"github.com/skip-mev/pob/x/auction/keeper"
	auctiontypes "github.com/skip-mev/pob/x/auction/types"
	"github.com/stretchr/testify/suite"
)

type AnteTestSuite struct {
	suite.Suite
	ctx sdk.Context

	// mempool setup
	encodingConfig encodingConfig
	random         *rand.Rand

	// auction setup
	auctionKeeper    keeper.Keeper
	bankKeeper       *MockBankKeeper
	accountKeeper    *MockAccountKeeper
	distrKeeper      *MockDistributionKeeper
	stakingKeeper    *MockStakingKeeper
	auctionDecorator ante.AuctionDecorator
	key              *storetypes.KVStoreKey
	authorityAccount sdk.AccAddress
}

func TestAnteTestSuite(t *testing.T) {
	suite.Run(t, new(AnteTestSuite))
}

func (suite *AnteTestSuite) SetupTest() {
	// General config
	suite.encodingConfig = createTestEncodingConfig()
	suite.random = rand.New(rand.NewSource(time.Now().Unix()))
	suite.key = sdk.NewKVStoreKey(auctiontypes.StoreKey)
	testCtx := testutil.DefaultContextWithDB(suite.T(), suite.key, sdk.NewTransientStoreKey("transient_test"))
	suite.ctx = testCtx.Ctx

	// Keepers set up
	ctrl := gomock.NewController(suite.T())
	suite.accountKeeper = NewMockAccountKeeper(ctrl)
	suite.accountKeeper.EXPECT().GetModuleAddress(auctiontypes.ModuleName).Return(sdk.AccAddress{}).AnyTimes()
	suite.bankKeeper = NewMockBankKeeper(ctrl)
	suite.distrKeeper = NewMockDistributionKeeper(ctrl)
	suite.stakingKeeper = NewMockStakingKeeper(ctrl)
	suite.authorityAccount = sdk.AccAddress([]byte("authority"))
	suite.auctionKeeper = keeper.NewKeeper(
		suite.encodingConfig.Codec,
		suite.key,
		suite.accountKeeper,
		suite.bankKeeper,
		suite.distrKeeper,
		suite.stakingKeeper,
		suite.authorityAccount.String(),
	)
	err := suite.auctionKeeper.SetParams(suite.ctx, auctiontypes.DefaultParams())
	suite.Require().NoError(err)
}

func (suite *AnteTestSuite) executeAnteHandler(tx sdk.Tx, balance sdk.Coins) (sdk.Context, error) {
	signer := tx.GetMsgs()[0].GetSigners()[0]
	suite.bankKeeper.EXPECT().GetAllBalances(suite.ctx, signer).AnyTimes().Return(balance)

	next := func(ctx sdk.Context, tx sdk.Tx, simulate bool) (sdk.Context, error) {
		return ctx, nil
	}

	return suite.auctionDecorator.AnteHandle(suite.ctx, tx, false, next)
}

func (suite *AnteTestSuite) TestAnteHandler() {
	var (
		// Bid set up
		bidder  = RandomAccounts(suite.random, 1)[0]
		bid     = sdk.NewCoins(sdk.NewCoin("foo", sdk.NewInt(1000)))
		balance = sdk.NewCoins(sdk.NewCoin("foo", sdk.NewInt(10000)))
		signers = []Account{bidder}

		// Top bidding auction tx set up
		topBidder    = RandomAccounts(suite.random, 1)[0]
		topBid       = sdk.NewCoins(sdk.NewCoin("foo", sdk.NewInt(100)))
		insertTopBid = true

		// Auction setup
		maxBundleSize          uint32 = 5
		reserveFee                    = sdk.NewCoins(sdk.NewCoin("foo", sdk.NewInt(100)))
		minBuyInFee                   = sdk.NewCoins(sdk.NewCoin("foo", sdk.NewInt(100)))
		minBidIncrement               = sdk.NewCoins(sdk.NewCoin("foo", sdk.NewInt(100)))
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
				topBid = sdk.NewCoins(sdk.NewCoin("foo", sdk.NewInt(100000)))
			},
			false,
		},
		{
			"bidder has insufficient balance, invalid auction tx",
			func() {
				insertTopBid = false
				balance = sdk.NewCoins(sdk.NewCoin("foo", sdk.NewInt(10)))
			},
			false,
		},
		{
			"bid is smaller than reserve fee, invalid auction tx",
			func() {
				balance = sdk.NewCoins(sdk.NewCoin("foo", sdk.NewInt(10000)))
				bid = sdk.NewCoins(sdk.NewCoin("foo", sdk.NewInt(101)))
				reserveFee = sdk.NewCoins(sdk.NewCoin("foo", sdk.NewInt(1000)))
			},
			false,
		},
		{
			"bid is greater than reserve fee but has insufficient balance to pay the buy in fee",
			func() {
				balance = sdk.NewCoins(sdk.NewCoin("foo", sdk.NewInt(1000)))
				bid = sdk.NewCoins(sdk.NewCoin("foo", sdk.NewInt(101)))
				reserveFee = sdk.NewCoins(sdk.NewCoin("foo", sdk.NewInt(100)))
				minBuyInFee = sdk.NewCoins(sdk.NewCoin("foo", sdk.NewInt(1000)))
			},
			false,
		},
		{
			"valid auction bid tx",
			func() {
				balance = sdk.NewCoins(sdk.NewCoin("foo", sdk.NewInt(10000)))
				bid = sdk.NewCoins(sdk.NewCoin("foo", sdk.NewInt(1000)))
				reserveFee = sdk.NewCoins(sdk.NewCoin("foo", sdk.NewInt(100)))
				minBuyInFee = sdk.NewCoins(sdk.NewCoin("foo", sdk.NewInt(100)))
			},
			true,
		},
		{
			"auction tx is the top bidding tx",
			func() {
				balance = sdk.NewCoins(sdk.NewCoin("foo", sdk.NewInt(10000)))
				bid = sdk.NewCoins(sdk.NewCoin("foo", sdk.NewInt(1000)))
				reserveFee = sdk.NewCoins(sdk.NewCoin("foo", sdk.NewInt(100)))
				minBuyInFee = sdk.NewCoins(sdk.NewCoin("foo", sdk.NewInt(100)))

				insertTopBid = true
				topBidder = bidder
				topBid = bid
				signers = []Account{}
			},
			true,
		},
		{
			"invalid frontrunning auction bid tx",
			func() {
				randomAccount := RandomAccounts(suite.random, 2)
				bidder := randomAccount[0]
				otherUser := randomAccount[1]
				insertTopBid = false

				signers = []Account{bidder, otherUser}
			},
			false,
		},
		{
			"valid frontrunning auction bid tx",
			func() {
				randomAccount := RandomAccounts(suite.random, 2)
				bidder := randomAccount[0]
				otherUser := randomAccount[1]

				signers = []Account{bidder, otherUser}
				frontRunningProtection = false
			},
			true,
		},
		{
			"invalid sandwiching auction bid tx",
			func() {
				randomAccount := RandomAccounts(suite.random, 2)
				bidder := randomAccount[0]
				otherUser := randomAccount[1]

				signers = []Account{bidder, otherUser, bidder}
				frontRunningProtection = true
			},
			false,
		},
		{
			"invalid auction bid tx with many signers",
			func() {
				signers = RandomAccounts(suite.random, 10)
				frontRunningProtection = true
			},
			false,
		},
	}

	for _, tc := range cases {
		suite.Run(tc.name, func() {
			suite.SetupTest()
			tc.malleate()

			// Set the auction params
			err := suite.auctionKeeper.SetParams(suite.ctx, auctiontypes.Params{
				MaxBundleSize:          maxBundleSize,
				ReserveFee:             reserveFee,
				MinBuyInFee:            minBuyInFee,
				MinBidIncrement:        minBidIncrement,
				FrontRunningProtection: frontRunningProtection,
			})
			suite.Require().NoError(err)

			// Insert the top bid into the mempool
			mempool := mempool.NewAuctionMempool(suite.encodingConfig.TxConfig.TxDecoder(), 0)
			if insertTopBid {
				topAuctionTx, err := createAuctionTxWithSigners(suite.encodingConfig.TxConfig, topBidder, topBid, 0, []Account{})
				suite.Require().NoError(err)
				suite.Require().Equal(0, mempool.CountTx())
				suite.Require().Equal(0, mempool.CountAuctionTx())
				suite.Require().NoError(mempool.Insert(suite.ctx, topAuctionTx))
				suite.Require().Equal(1, mempool.CountTx())
				suite.Require().Equal(1, mempool.CountAuctionTx())
			}

			// Create the actual auction tx and insert into the mempool
			auctionTx, err := createAuctionTxWithSigners(suite.encodingConfig.TxConfig, bidder, bid, 0, signers)
			suite.Require().NoError(err)

			// Execute the ante handler
			suite.auctionDecorator = ante.NewAuctionDecorator(suite.auctionKeeper, suite.encodingConfig.TxConfig.TxDecoder(), suite.encodingConfig.TxConfig.TxEncoder(), mempool)
			_, err = suite.executeAnteHandler(auctionTx, balance)
			if tc.pass {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}
