package abci_test

import (
	"math/rand"
	"testing"
	"time"

	"cosmossdk.io/log"
	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/golang/mock/gomock"
	"github.com/skip-mev/pob/blockbuster"
	"github.com/skip-mev/pob/blockbuster/abci"
	"github.com/skip-mev/pob/blockbuster/lanes/auction"
	"github.com/skip-mev/pob/blockbuster/lanes/base"
	"github.com/skip-mev/pob/blockbuster/lanes/free"
	testutils "github.com/skip-mev/pob/testutils"
	"github.com/skip-mev/pob/x/builder/ante"
	"github.com/skip-mev/pob/x/builder/keeper"
	buildertypes "github.com/skip-mev/pob/x/builder/types"
	"github.com/stretchr/testify/suite"

	abcitypes "github.com/cometbft/cometbft/abci/types"
)

type ABCITestSuite struct {
	suite.Suite
	ctx sdk.Context

	// Define basic tx configuration
	encodingConfig testutils.EncodingConfig

	// Define all of the lanes utilized in the test suite
	tobConfig blockbuster.BaseLaneConfig
	tobLane   *auction.TOBLane

	freeConfig blockbuster.BaseLaneConfig
	freeLane   *free.Lane

	baseConfig blockbuster.BaseLaneConfig
	baseLane   *base.DefaultLane

	lanes   []blockbuster.Lane
	mempool blockbuster.Mempool

	// Proposal handler set up
	proposalHandler *abci.ProposalHandler

	// account set up
	accounts []testutils.Account
	random   *rand.Rand
	nonces   map[string]uint64

	// Keeper set up
	builderKeeper    keeper.Keeper
	bankKeeper       *testutils.MockBankKeeper
	accountKeeper    *testutils.MockAccountKeeper
	distrKeeper      *testutils.MockDistributionKeeper
	stakingKeeper    *testutils.MockStakingKeeper
	builderDecorator ante.BuilderDecorator
}

func TestBlockBusterTestSuite(t *testing.T) {
	suite.Run(t, new(ABCITestSuite))
}

func (suite *ABCITestSuite) SetupTest() {
	// General config for transactions and randomness for the test suite
	suite.encodingConfig = testutils.CreateTestEncodingConfig()
	suite.random = rand.New(rand.NewSource(time.Now().Unix()))
	key := storetypes.NewKVStoreKey(buildertypes.StoreKey)
	testCtx := testutil.DefaultContextWithDB(suite.T(), key, storetypes.NewTransientStoreKey("transient_test"))
	suite.ctx = testCtx.Ctx.WithBlockHeight(1)

	// Lanes configuration
	// Top of block lane set up
	suite.tobConfig = blockbuster.BaseLaneConfig{
		Logger:        log.NewTestLogger(suite.T()),
		TxEncoder:     suite.encodingConfig.TxConfig.TxEncoder(),
		TxDecoder:     suite.encodingConfig.TxConfig.TxDecoder(),
		AnteHandler:   suite.anteHandler,
		MaxBlockSpace: math.LegacyZeroDec(), // It can be as big as it wants (up to maxTxBytes)
	}
	suite.tobLane = auction.NewTOBLane(
		suite.tobConfig,
		0, // No bound on the number of transactions in the lane
		auction.NewDefaultAuctionFactory(suite.encodingConfig.TxConfig.TxDecoder()),
	)

	// Free lane set up
	suite.freeConfig = blockbuster.BaseLaneConfig{
		Logger:        log.NewTestLogger(suite.T()),
		TxEncoder:     suite.encodingConfig.TxConfig.TxEncoder(),
		TxDecoder:     suite.encodingConfig.TxConfig.TxDecoder(),
		AnteHandler:   suite.anteHandler,
		MaxBlockSpace: math.LegacyZeroDec(), // It can be as big as it wants (up to maxTxBytes)
		IgnoreList:    []blockbuster.Lane{suite.tobLane},
	}
	suite.freeLane = free.NewFreeLane(
		suite.freeConfig,
		free.NewDefaultFreeFactory(suite.encodingConfig.TxConfig.TxDecoder()),
	)

	// Base lane set up
	suite.baseConfig = blockbuster.BaseLaneConfig{
		Logger:        log.NewTestLogger(suite.T()),
		TxEncoder:     suite.encodingConfig.TxConfig.TxEncoder(),
		TxDecoder:     suite.encodingConfig.TxConfig.TxDecoder(),
		AnteHandler:   suite.anteHandler,
		MaxBlockSpace: math.LegacyZeroDec(), // It can be as big as it wants (up to maxTxBytes)
		IgnoreList:    []blockbuster.Lane{suite.tobLane, suite.freeLane},
	}
	suite.baseLane = base.NewDefaultLane(
		suite.baseConfig,
	)

	// Mempool set up
	suite.lanes = []blockbuster.Lane{suite.tobLane, suite.freeLane, suite.baseLane}
	suite.mempool = blockbuster.NewMempool(log.NewTestLogger(suite.T()), suite.lanes...)

	// Accounts set up
	suite.accounts = testutils.RandomAccounts(suite.random, 10)
	suite.nonces = make(map[string]uint64)
	for _, acc := range suite.accounts {
		suite.nonces[acc.Address.String()] = 0
	}

	// Set up the keepers and decorators
	// Mock keepers set up
	ctrl := gomock.NewController(suite.T())
	suite.accountKeeper = testutils.NewMockAccountKeeper(ctrl)
	suite.accountKeeper.EXPECT().GetModuleAddress(buildertypes.ModuleName).Return(sdk.AccAddress{}).AnyTimes()
	suite.bankKeeper = testutils.NewMockBankKeeper(ctrl)
	suite.distrKeeper = testutils.NewMockDistributionKeeper(ctrl)
	suite.stakingKeeper = testutils.NewMockStakingKeeper(ctrl)

	// Builder keeper / decorator set up
	suite.builderKeeper = keeper.NewKeeper(
		suite.encodingConfig.Codec,
		key,
		suite.accountKeeper,
		suite.bankKeeper,
		suite.distrKeeper,
		suite.stakingKeeper,
		sdk.AccAddress([]byte("authority")).String(),
	)

	// Set the default params for the builder keeper
	err := suite.builderKeeper.SetParams(suite.ctx, buildertypes.DefaultParams())
	suite.Require().NoError(err)

	// Set up the ante handler
	suite.builderDecorator = ante.NewBuilderDecorator(suite.builderKeeper, suite.encodingConfig.TxConfig.TxEncoder(), suite.tobLane, suite.mempool)

	// Proposal handler set up
	suite.proposalHandler = abci.NewProposalHandler(log.NewTestLogger(suite.T()), suite.encodingConfig.TxConfig.TxDecoder(), suite.mempool)
}

func (suite *ABCITestSuite) anteHandler(ctx sdk.Context, tx sdk.Tx, _ bool) (sdk.Context, error) {
	suite.bankKeeper.EXPECT().GetBalance(ctx, gomock.Any(), "stake").AnyTimes().Return(
		sdk.NewCoin("stake", math.NewInt(100000000000000)),
	)

	next := func(ctx sdk.Context, tx sdk.Tx, simulate bool) (sdk.Context, error) {
		return ctx, nil
	}

	return suite.builderDecorator.AnteHandle(ctx, tx, false, next)
}

func (suite *ABCITestSuite) resetLanesWithNewConfig() {
	// Top of block lane set up
	suite.tobLane = auction.NewTOBLane(
		suite.tobConfig,
		0, // No bound on the number of transactions in the lane
		auction.NewDefaultAuctionFactory(suite.encodingConfig.TxConfig.TxDecoder()),
	)

	// Free lane set up
	suite.freeLane = free.NewFreeLane(
		suite.freeConfig,
		free.NewDefaultFreeFactory(suite.encodingConfig.TxConfig.TxDecoder()),
	)

	// Base lane set up
	suite.baseLane = base.NewDefaultLane(
		suite.baseConfig,
	)

	suite.lanes = []blockbuster.Lane{suite.tobLane, suite.freeLane, suite.baseLane}

	suite.mempool = blockbuster.NewMempool(log.NewTestLogger(suite.T()), suite.lanes...)
}

func (suite *ABCITestSuite) TestPrepareProposal() {
	var (
		// the modified transactions cannot exceed this size
		maxTxBytes int64 = 1000000000000000000

		// mempool configuration
		txs              []sdk.Tx
		auctionTxs       []sdk.Tx
		winningBidTx     sdk.Tx
		insertBundledTxs = false

		// auction configuration
		maxBundleSize          uint32 = 10
		reserveFee                    = sdk.NewCoin("stake", math.NewInt(1000))
		minBidIncrement               = sdk.NewCoin("stake", math.NewInt(100))
		frontRunningProtection        = true
	)

	cases := []struct {
		name                        string
		malleate                    func()
		expectedNumberProposalTxs   int
		expectedMempoolDistribution map[string]int
	}{
		{
			"empty mempool",
			func() {
				txs = []sdk.Tx{}
				auctionTxs = []sdk.Tx{}
				winningBidTx = nil
				insertBundledTxs = false
			},
			0,
			map[string]int{
				base.LaneName:    0,
				auction.LaneName: 0,
				free.LaneName:    0,
			},
		},
		{
			"maxTxBytes is less than any transaction in the mempool",
			func() {
				// Create a tob tx
				bidder := suite.accounts[0]
				bid := sdk.NewCoin("stake", math.NewInt(1000))
				nonce := suite.nonces[bidder.Address.String()]
				timeout := uint64(100)
				signers := []testutils.Account{bidder}
				bidTx, err := testutils.CreateAuctionTxWithSigners(suite.encodingConfig.TxConfig, bidder, bid, nonce, timeout, signers)
				suite.Require().NoError(err)

				// Create a free tx
				account := suite.accounts[1]
				nonce = suite.nonces[account.Address.String()]
				freeTx, err := testutils.CreateFreeTx(suite.encodingConfig.TxConfig, account, nonce, timeout, "val1", bid)
				suite.Require().NoError(err)

				// Create a normal tx
				account = suite.accounts[2]
				nonce = suite.nonces[account.Address.String()]
				numberMsgs := uint64(3)
				normalTx, err := testutils.CreateRandomTx(suite.encodingConfig.TxConfig, account, nonce, numberMsgs, timeout)
				suite.Require().NoError(err)

				txs = []sdk.Tx{freeTx, normalTx}
				auctionTxs = []sdk.Tx{bidTx}
				winningBidTx = nil
				insertBundledTxs = false
				maxTxBytes = 10
			},
			0,
			map[string]int{
				base.LaneName:    1,
				auction.LaneName: 1,
				free.LaneName:    1,
			},
		},
		{
			"valid tob tx but maxTxBytes is less for the tob lane so only the free tx should be included",
			func() {
				// Create a tob tx
				bidder := suite.accounts[0]
				bid := sdk.NewCoin("stake", math.NewInt(1000))
				nonce := suite.nonces[bidder.Address.String()]
				timeout := uint64(100)
				signers := []testutils.Account{suite.accounts[2], bidder}
				bidTx, err := testutils.CreateAuctionTxWithSigners(suite.encodingConfig.TxConfig, bidder, bid, nonce, timeout, signers)
				suite.Require().NoError(err)

				// Create a free tx
				account := suite.accounts[1]
				nonce = suite.nonces[account.Address.String()]
				freeTx, err := testutils.CreateFreeTx(suite.encodingConfig.TxConfig, account, nonce, timeout, "val1", bid)
				suite.Require().NoError(err)

				// Get the size of the tob tx
				bidTxBytes, err := suite.encodingConfig.TxConfig.TxEncoder()(bidTx)
				suite.Require().NoError(err)
				tobSize := int64(len(bidTxBytes))

				// Get the size of the free tx
				freeTxBytes, err := suite.encodingConfig.TxConfig.TxEncoder()(freeTx)
				suite.Require().NoError(err)
				freeSize := int64(len(freeTxBytes))

				maxTxBytes = tobSize + freeSize
				suite.tobConfig.MaxBlockSpace = math.LegacyMustNewDecFromStr("0.1")

				txs = []sdk.Tx{freeTx}
				auctionTxs = []sdk.Tx{bidTx}
				winningBidTx = nil
				insertBundledTxs = false
			},
			1,
			map[string]int{
				base.LaneName:    0,
				auction.LaneName: 1,
				free.LaneName:    1,
			},
		},
		{
			"valid tob tx with sufficient space for only tob tx",
			func() {
				// Create a tob tx
				bidder := suite.accounts[0]
				bid := sdk.NewCoin("stake", math.NewInt(1000))
				nonce := suite.nonces[bidder.Address.String()]
				timeout := uint64(100)
				signers := []testutils.Account{suite.accounts[2]}
				bidTx, err := testutils.CreateAuctionTxWithSigners(suite.encodingConfig.TxConfig, bidder, bid, nonce, timeout, signers)
				suite.Require().NoError(err)

				// Create a free tx
				account := suite.accounts[1]
				nonce = suite.nonces[account.Address.String()]
				freeTx, err := testutils.CreateFreeTx(suite.encodingConfig.TxConfig, account, nonce, timeout, "val1", bid)
				suite.Require().NoError(err)

				// Get the size of the tob tx
				bidTxBytes, err := suite.encodingConfig.TxConfig.TxEncoder()(bidTx)
				suite.Require().NoError(err)
				tobSize := int64(len(bidTxBytes))

				// Get the size of the free tx
				freeTxBytes, err := suite.encodingConfig.TxConfig.TxEncoder()(freeTx)
				suite.Require().NoError(err)
				freeSize := int64(len(freeTxBytes))

				maxTxBytes = tobSize*2 + freeSize - 1
				suite.tobConfig.MaxBlockSpace = math.LegacyZeroDec()
				suite.freeConfig.MaxBlockSpace = math.LegacyMustNewDecFromStr("0.1")

				txs = []sdk.Tx{freeTx}
				auctionTxs = []sdk.Tx{bidTx}
				winningBidTx = bidTx
				insertBundledTxs = false
			},
			2,
			map[string]int{
				base.LaneName:    0,
				auction.LaneName: 1,
				free.LaneName:    1,
			},
		},
		{
			"tob, free, and normal tx but only space for tob and normal tx",
			func() {
				// Create a tob tx
				bidder := suite.accounts[0]
				bid := sdk.NewCoin("stake", math.NewInt(1000))
				nonce := suite.nonces[bidder.Address.String()]
				timeout := uint64(100)
				signers := []testutils.Account{suite.accounts[2], bidder}
				bidTx, err := testutils.CreateAuctionTxWithSigners(suite.encodingConfig.TxConfig, bidder, bid, nonce, timeout, signers)
				suite.Require().NoError(err)

				// Create a free tx
				account := suite.accounts[1]
				nonce = suite.nonces[account.Address.String()]
				freeTx, err := testutils.CreateFreeTx(suite.encodingConfig.TxConfig, account, nonce, timeout, "val1", bid)
				suite.Require().NoError(err)

				// Create a normal tx
				account = suite.accounts[3]
				nonce = suite.nonces[account.Address.String()]
				numberMsgs := uint64(3)
				normalTx, err := testutils.CreateRandomTx(suite.encodingConfig.TxConfig, account, nonce, numberMsgs, timeout)
				suite.Require().NoError(err)

				// Get the size of the tob tx
				bidTxBytes, err := suite.encodingConfig.TxConfig.TxEncoder()(bidTx)
				suite.Require().NoError(err)
				tobSize := int64(len(bidTxBytes))

				// Get the size of the free tx
				freeTxBytes, err := suite.encodingConfig.TxConfig.TxEncoder()(freeTx)
				suite.Require().NoError(err)
				freeSize := int64(len(freeTxBytes))

				// Get the size of the normal tx
				normalTxBytes, err := suite.encodingConfig.TxConfig.TxEncoder()(normalTx)
				suite.Require().NoError(err)
				normalSize := int64(len(normalTxBytes))

				maxTxBytes = tobSize*2 + freeSize + normalSize + 1

				// Tob can take up as much space as it wants
				suite.tobConfig.MaxBlockSpace = math.LegacyZeroDec()

				// Free can take up less space than the tx
				suite.freeConfig.MaxBlockSpace = math.LegacyMustNewDecFromStr("0.01")

				// Default can take up as much space as it wants
				suite.baseConfig.MaxBlockSpace = math.LegacyZeroDec()

				txs = []sdk.Tx{freeTx, normalTx}
				auctionTxs = []sdk.Tx{bidTx}
				winningBidTx = bidTx
				insertBundledTxs = false
			},
			4,
			map[string]int{
				base.LaneName:    1,
				auction.LaneName: 1,
				free.LaneName:    1,
			},
		},
		{
			"single valid tob transaction in the mempool",
			func() {
				// reset the configs
				suite.tobConfig.MaxBlockSpace = math.LegacyZeroDec()
				suite.freeConfig.MaxBlockSpace = math.LegacyZeroDec()
				suite.baseConfig.MaxBlockSpace = math.LegacyZeroDec()

				bidder := suite.accounts[0]
				bid := sdk.NewCoin("stake", math.NewInt(1000))
				nonce := suite.nonces[bidder.Address.String()]
				timeout := uint64(100)
				signers := []testutils.Account{bidder}
				bidTx, err := testutils.CreateAuctionTxWithSigners(suite.encodingConfig.TxConfig, bidder, bid, nonce, timeout, signers)
				suite.Require().NoError(err)

				txs = []sdk.Tx{}
				auctionTxs = []sdk.Tx{bidTx}
				winningBidTx = bidTx
				insertBundledTxs = false
				maxTxBytes = 1000000000000000000
			},
			2,
			map[string]int{
				base.LaneName:    0,
				auction.LaneName: 1,
				free.LaneName:    0,
			},
		},
		{
			"single invalid tob transaction in the mempool",
			func() {
				bidder := suite.accounts[0]
				bid := reserveFee.Sub(sdk.NewCoin("stake", math.NewInt(1))) // bid is less than the reserve fee
				nonce := suite.nonces[bidder.Address.String()]
				timeout := uint64(100)
				signers := []testutils.Account{bidder}
				bidTx, err := testutils.CreateAuctionTxWithSigners(suite.encodingConfig.TxConfig, bidder, bid, nonce, timeout, signers)
				suite.Require().NoError(err)

				txs = []sdk.Tx{}
				auctionTxs = []sdk.Tx{bidTx}
				winningBidTx = nil
				insertBundledTxs = false
			},
			0,
			map[string]int{
				base.LaneName:    0,
				auction.LaneName: 0,
				free.LaneName:    0,
			},
		},
		{
			"normal transactions in the mempool",
			func() {
				account := suite.accounts[0]
				nonce := suite.nonces[account.Address.String()]
				timeout := uint64(100)
				numberMsgs := uint64(3)
				normalTx, err := testutils.CreateRandomTx(suite.encodingConfig.TxConfig, account, nonce, numberMsgs, timeout)
				suite.Require().NoError(err)

				txs = []sdk.Tx{normalTx}
				auctionTxs = []sdk.Tx{}
				winningBidTx = nil
				insertBundledTxs = false
			},
			1,
			map[string]int{
				base.LaneName:    1,
				auction.LaneName: 0,
				free.LaneName:    0,
			},
		},
		{
			"normal transactions and tob transactions in the mempool",
			func() {
				// Create a valid tob transaction
				bidder := suite.accounts[0]
				bid := sdk.NewCoin("stake", math.NewInt(1000))
				nonce := suite.nonces[bidder.Address.String()]
				timeout := uint64(100)
				signers := []testutils.Account{bidder}
				bidTx, err := testutils.CreateAuctionTxWithSigners(suite.encodingConfig.TxConfig, bidder, bid, nonce, timeout, signers)
				suite.Require().NoError(err)

				// Create a valid default transaction
				account := suite.accounts[1]
				nonce = suite.nonces[account.Address.String()] + 1
				numberMsgs := uint64(3)
				normalTx, err := testutils.CreateRandomTx(suite.encodingConfig.TxConfig, account, nonce, numberMsgs, timeout)
				suite.Require().NoError(err)

				txs = []sdk.Tx{normalTx}
				auctionTxs = []sdk.Tx{bidTx}
				winningBidTx = bidTx
				insertBundledTxs = false
			},
			3,
			map[string]int{
				base.LaneName:    1,
				auction.LaneName: 1,
				free.LaneName:    0,
			},
		},
		{
			"multiple tob transactions where the first is invalid",
			func() {
				// Create an invalid tob transaction (frontrunning)
				bidder := suite.accounts[0]
				bid := sdk.NewCoin("stake", math.NewInt(1000000000))
				nonce := suite.nonces[bidder.Address.String()]
				timeout := uint64(100)
				signers := []testutils.Account{bidder, bidder, suite.accounts[1]}
				bidTx, err := testutils.CreateAuctionTxWithSigners(suite.encodingConfig.TxConfig, bidder, bid, nonce, timeout, signers)
				suite.Require().NoError(err)

				// Create a valid tob transaction
				bidder = suite.accounts[1]
				bid = sdk.NewCoin("stake", math.NewInt(1000))
				nonce = suite.nonces[bidder.Address.String()]
				timeout = uint64(100)
				signers = []testutils.Account{bidder}
				bidTx2, err := testutils.CreateAuctionTxWithSigners(suite.encodingConfig.TxConfig, bidder, bid, nonce, timeout, signers)
				suite.Require().NoError(err)

				txs = []sdk.Tx{}
				auctionTxs = []sdk.Tx{bidTx, bidTx2}
				winningBidTx = bidTx2
				insertBundledTxs = false
			},
			2,
			map[string]int{
				base.LaneName:    0,
				auction.LaneName: 1,
				free.LaneName:    0,
			},
		},
		{
			"multiple tob transactions where the first is valid",
			func() {
				// Create an valid tob transaction
				bidder := suite.accounts[0]
				bid := sdk.NewCoin("stake", math.NewInt(10000000))
				nonce := suite.nonces[bidder.Address.String()]
				timeout := uint64(100)
				signers := []testutils.Account{suite.accounts[2], bidder}
				bidTx, err := testutils.CreateAuctionTxWithSigners(suite.encodingConfig.TxConfig, bidder, bid, nonce, timeout, signers)
				suite.Require().NoError(err)

				// Create a valid tob transaction
				bidder = suite.accounts[1]
				bid = sdk.NewCoin("stake", math.NewInt(1000))
				nonce = suite.nonces[bidder.Address.String()]
				timeout = uint64(100)
				signers = []testutils.Account{bidder}
				bidTx2, err := testutils.CreateAuctionTxWithSigners(suite.encodingConfig.TxConfig, bidder, bid, nonce, timeout, signers)
				suite.Require().NoError(err)

				txs = []sdk.Tx{}
				auctionTxs = []sdk.Tx{bidTx, bidTx2}
				winningBidTx = bidTx
				insertBundledTxs = false
			},
			3,
			map[string]int{
				base.LaneName:    0,
				auction.LaneName: 2,
				free.LaneName:    0,
			},
		},
		{
			"multiple tob transactions where the first is valid and bundle is inserted into mempool",
			func() {
				frontRunningProtection = false

				// Create an valid tob transaction
				bidder := suite.accounts[0]
				bid := sdk.NewCoin("stake", math.NewInt(10000000))
				nonce := suite.nonces[bidder.Address.String()]
				timeout := uint64(100)
				signers := []testutils.Account{suite.accounts[2], suite.accounts[1], bidder, suite.accounts[3], suite.accounts[4]}
				bidTx, err := testutils.CreateAuctionTxWithSigners(suite.encodingConfig.TxConfig, bidder, bid, nonce, timeout, signers)
				suite.Require().NoError(err)

				txs = []sdk.Tx{}
				auctionTxs = []sdk.Tx{bidTx}
				winningBidTx = bidTx
				insertBundledTxs = true
			},
			6,
			map[string]int{
				base.LaneName:    5,
				auction.LaneName: 1,
				free.LaneName:    0,
			},
		},
		{
			"valid tob, free, and normal tx",
			func() {
				// Create a tob tx
				bidder := suite.accounts[0]
				bid := sdk.NewCoin("stake", math.NewInt(1000))
				nonce := suite.nonces[bidder.Address.String()]
				timeout := uint64(100)
				signers := []testutils.Account{suite.accounts[2], bidder}
				bidTx, err := testutils.CreateAuctionTxWithSigners(suite.encodingConfig.TxConfig, bidder, bid, nonce, timeout, signers)
				suite.Require().NoError(err)

				// Create a free tx
				account := suite.accounts[1]
				nonce = suite.nonces[account.Address.String()]
				freeTx, err := testutils.CreateFreeTx(suite.encodingConfig.TxConfig, account, nonce, timeout, "val1", bid)
				suite.Require().NoError(err)

				// Create a normal tx
				account = suite.accounts[3]
				nonce = suite.nonces[account.Address.String()]
				numberMsgs := uint64(3)
				normalTx, err := testutils.CreateRandomTx(suite.encodingConfig.TxConfig, account, nonce, numberMsgs, timeout)
				suite.Require().NoError(err)

				txs = []sdk.Tx{freeTx, normalTx}
				auctionTxs = []sdk.Tx{bidTx}
				winningBidTx = bidTx
				insertBundledTxs = false
			},
			5,
			map[string]int{
				base.LaneName:    1,
				auction.LaneName: 1,
				free.LaneName:    1,
			},
		},
	}

	for _, tc := range cases {
		suite.Run(tc.name, func() {
			suite.SetupTest() // reset
			tc.malleate()
			suite.resetLanesWithNewConfig()

			// Insert all of the normal transactions into the default lane
			for _, tx := range txs {
				suite.Require().NoError(suite.mempool.Insert(suite.ctx, tx))
			}

			// Insert all of the auction transactions into the TOB lane
			for _, tx := range auctionTxs {
				suite.Require().NoError(suite.mempool.Insert(suite.ctx, tx))
			}

			// Insert all of the bundled transactions into the TOB lane if desired
			if insertBundledTxs {
				for _, tx := range auctionTxs {
					bidInfo, err := suite.tobLane.GetAuctionBidInfo(tx)
					suite.Require().NoError(err)

					for _, txBz := range bidInfo.Transactions {
						tx, err := suite.encodingConfig.TxConfig.TxDecoder()(txBz)
						suite.Require().NoError(err)

						suite.Require().NoError(suite.mempool.Insert(suite.ctx, tx))
					}
				}
			}

			// Create a new auction
			params := buildertypes.Params{
				MaxBundleSize:          maxBundleSize,
				ReserveFee:             reserveFee,
				FrontRunningProtection: frontRunningProtection,
				MinBidIncrement:        minBidIncrement,
			}
			suite.builderKeeper.SetParams(suite.ctx, params)
			suite.builderDecorator = ante.NewBuilderDecorator(suite.builderKeeper, suite.encodingConfig.TxConfig.TxEncoder(), suite.tobLane, suite.mempool)

			for _, lane := range suite.lanes {
				lane.SetAnteHandler(suite.anteHandler)
			}

			// Create a new proposal handler
			suite.proposalHandler = abci.NewProposalHandler(log.NewTestLogger(suite.T()), suite.encodingConfig.TxConfig.TxDecoder(), suite.mempool)
			handler := suite.proposalHandler.PrepareProposalHandler()
			res, err := handler(suite.ctx, &abcitypes.RequestPrepareProposal{
				MaxTxBytes: maxTxBytes,
			})
			suite.Require().NoError(err)

			// -------------------- Check Invariants -------------------- //
			// 1. the number of transactions in the response must be equal to the number of expected transactions
			suite.Require().Equal(tc.expectedNumberProposalTxs, len(res.Txs))

			// 2. total bytes must be less than or equal to maxTxBytes
			totalBytes := int64(0)
			txIndex := 0
			for txIndex < len(res.Txs) {
				totalBytes += int64(len(res.Txs[txIndex]))

				tx, err := suite.encodingConfig.TxConfig.TxDecoder()(res.Txs[txIndex])
				suite.Require().NoError(err)

				suite.Require().Equal(true, suite.mempool.Contains(tx))

				// In the case where we have a tob tx, we skip the other transactions in the bundle
				// in order to not double count
				switch {
				case suite.tobLane.Match(suite.ctx, tx):
					bidInfo, err := suite.tobLane.GetAuctionBidInfo(tx)
					suite.Require().NoError(err)

					txIndex += len(bidInfo.Transactions) + 1
				default:
					txIndex++
				}
			}

			suite.Require().LessOrEqual(totalBytes, maxTxBytes)

			// 3. if there are auction transactions, the first transaction must be the top bid
			// and the rest of the bundle must be in the response
			if winningBidTx != nil {
				auctionTx, err := suite.encodingConfig.TxConfig.TxDecoder()(res.Txs[0])
				suite.Require().NoError(err)

				bidInfo, err := suite.tobLane.GetAuctionBidInfo(auctionTx)
				suite.Require().NoError(err)

				for index, tx := range bidInfo.Transactions {
					suite.Require().Equal(tx, res.Txs[index+1])
				}
			} else if len(res.Txs) > 0 {
				tx, err := suite.encodingConfig.TxConfig.TxDecoder()(res.Txs[0])
				suite.Require().NoError(err)

				bidInfo, err := suite.tobLane.GetAuctionBidInfo(tx)
				suite.Require().NoError(err)
				suite.Require().Nil(bidInfo)
			}

			// 4. All of the transactions must be unique
			uniqueTxs := make(map[string]bool)
			for _, tx := range res.Txs {
				suite.Require().False(uniqueTxs[string(tx)])
				uniqueTxs[string(tx)] = true
			}

			// 5. The number of transactions in the mempool must be correct
			suite.Require().Equal(tc.expectedMempoolDistribution, suite.mempool.GetTxDistribution())

			// 6. The ordering of transactions must respect the ordering of the lanes
			laneIndex := 0
			txIndex = 0
			for txIndex < len(res.Txs) {
				sdkTx, err := suite.encodingConfig.TxConfig.TxDecoder()(res.Txs[txIndex])
				suite.Require().NoError(err)

				if suite.lanes[laneIndex].Match(suite.ctx, sdkTx) {
					switch suite.lanes[laneIndex].Name() {
					case suite.tobLane.Name():
						bidInfo, err := suite.tobLane.GetAuctionBidInfo(sdkTx)
						suite.Require().NoError(err)

						txIndex += len(bidInfo.Transactions) + 1
					default:
						txIndex++
					}
				} else {
					laneIndex++
				}

				suite.Require().Less(laneIndex, len(suite.lanes))
			}
		})
	}
}

func (suite *ABCITestSuite) TestProcessProposal() {
	var (
		// mempool configuration
		txs          []sdk.Tx
		auctionTxs   []sdk.Tx
		insertRefTxs = false

		// auction configuration
		maxBundleSize          uint32 = 10
		reserveFee                    = sdk.NewCoin("stake", math.NewInt(1000))
		frontRunningProtection        = true
	)

	cases := []struct {
		name     string
		malleate func()
		response abcitypes.ResponseProcessProposal_ProposalStatus
	}{
		{
			"no normal tx, no tob tx",
			func() {
			},
			abcitypes.ResponseProcessProposal_ACCEPT,
		},
		{
			"single default tx",
			func() {
				account := suite.accounts[0]
				nonce := suite.nonces[account.Address.String()]
				timeout := uint64(100)
				numberMsgs := uint64(3)
				normalTx, err := testutils.CreateRandomTx(suite.encodingConfig.TxConfig, account, nonce, numberMsgs, timeout)
				suite.Require().NoError(err)

				txs = []sdk.Tx{normalTx}
				auctionTxs = []sdk.Tx{}
				insertRefTxs = false
			},
			abcitypes.ResponseProcessProposal_ACCEPT,
		},
		{
			"single tob tx without bundled txs in proposal",
			func() {
				bidder := suite.accounts[0]
				bid := sdk.NewCoin("stake", math.NewInt(1000))
				nonce := suite.nonces[bidder.Address.String()]
				timeout := uint64(100)
				signers := []testutils.Account{bidder}
				bidTx, err := testutils.CreateAuctionTxWithSigners(suite.encodingConfig.TxConfig, bidder, bid, nonce, timeout, signers)
				suite.Require().NoError(err)

				txs = []sdk.Tx{}
				auctionTxs = []sdk.Tx{bidTx}
				insertRefTxs = false
			},
			abcitypes.ResponseProcessProposal_REJECT,
		},
		{
			"single tob tx with bundled txs in proposal",
			func() {
				bidder := suite.accounts[0]
				bid := sdk.NewCoin("stake", math.NewInt(1000))
				nonce := suite.nonces[bidder.Address.String()]
				timeout := uint64(100)
				signers := []testutils.Account{suite.accounts[1], bidder}
				bidTx, err := testutils.CreateAuctionTxWithSigners(suite.encodingConfig.TxConfig, bidder, bid, nonce, timeout, signers)
				suite.Require().NoError(err)

				txs = []sdk.Tx{}
				auctionTxs = []sdk.Tx{bidTx}
				insertRefTxs = true
			},
			abcitypes.ResponseProcessProposal_ACCEPT,
		},
		{
			"single invalid tob tx (front-running)",
			func() {
				// Create an valid tob transaction
				bidder := suite.accounts[0]
				bid := sdk.NewCoin("stake", math.NewInt(10000000))
				nonce := suite.nonces[bidder.Address.String()]
				timeout := uint64(100)
				signers := []testutils.Account{suite.accounts[2], suite.accounts[1], bidder, suite.accounts[3], suite.accounts[4]}
				bidTx, err := testutils.CreateAuctionTxWithSigners(suite.encodingConfig.TxConfig, bidder, bid, nonce, timeout, signers)
				suite.Require().NoError(err)

				txs = []sdk.Tx{}
				auctionTxs = []sdk.Tx{bidTx}
				insertRefTxs = true
			},
			abcitypes.ResponseProcessProposal_REJECT,
		},
		{
			"multiple tob txs in the proposal",
			func() {
				// Create an valid tob transaction
				bidder := suite.accounts[0]
				bid := sdk.NewCoin("stake", math.NewInt(10000000))
				nonce := suite.nonces[bidder.Address.String()]
				timeout := uint64(100)
				signers := []testutils.Account{suite.accounts[2], bidder}
				bidTx, err := testutils.CreateAuctionTxWithSigners(suite.encodingConfig.TxConfig, bidder, bid, nonce, timeout, signers)
				suite.Require().NoError(err)

				// Create a valid tob transaction
				bidder = suite.accounts[1]
				bid = sdk.NewCoin("stake", math.NewInt(1000))
				nonce = suite.nonces[bidder.Address.String()]
				timeout = uint64(100)
				signers = []testutils.Account{bidder}
				bidTx2, err := testutils.CreateAuctionTxWithSigners(suite.encodingConfig.TxConfig, bidder, bid, nonce, timeout, signers)
				suite.Require().NoError(err)

				txs = []sdk.Tx{}
				auctionTxs = []sdk.Tx{bidTx, bidTx2}
				insertRefTxs = true
			},
			abcitypes.ResponseProcessProposal_REJECT,
		},
		{
			"single tob tx with front-running disabled and multiple other txs",
			func() {
				frontRunningProtection = false

				// Create an valid tob transaction
				bidder := suite.accounts[0]
				bid := sdk.NewCoin("stake", math.NewInt(10000000))
				nonce := suite.nonces[bidder.Address.String()]
				timeout := uint64(100)
				signers := []testutils.Account{suite.accounts[2], bidder}
				bidTx, err := testutils.CreateAuctionTxWithSigners(suite.encodingConfig.TxConfig, bidder, bid, nonce, timeout, signers)
				suite.Require().NoError(err)

				// Create a few other transactions
				account := suite.accounts[1]
				nonce = suite.nonces[account.Address.String()]
				timeout = uint64(100)
				numberMsgs := uint64(3)
				normalTx, err := testutils.CreateRandomTx(suite.encodingConfig.TxConfig, account, nonce, numberMsgs, timeout)
				suite.Require().NoError(err)

				account = suite.accounts[3]
				nonce = suite.nonces[account.Address.String()]
				timeout = uint64(100)
				numberMsgs = uint64(3)
				normalTx2, err := testutils.CreateRandomTx(suite.encodingConfig.TxConfig, account, nonce, numberMsgs, timeout)
				suite.Require().NoError(err)

				txs = []sdk.Tx{normalTx, normalTx2}
				auctionTxs = []sdk.Tx{bidTx}
				insertRefTxs = true
			},
			abcitypes.ResponseProcessProposal_ACCEPT,
		},
		{
			"tob, free, and default tx",
			func() {
				// Create a tob tx
				bidder := suite.accounts[0]
				bid := sdk.NewCoin("stake", math.NewInt(1000))
				nonce := suite.nonces[bidder.Address.String()]
				timeout := uint64(100)
				signers := []testutils.Account{suite.accounts[2], bidder}
				bidTx, err := testutils.CreateAuctionTxWithSigners(suite.encodingConfig.TxConfig, bidder, bid, nonce, timeout, signers)
				suite.Require().NoError(err)

				// Create a free tx
				account := suite.accounts[1]
				nonce = suite.nonces[account.Address.String()]
				freeTx, err := testutils.CreateFreeTx(suite.encodingConfig.TxConfig, account, nonce, timeout, "val1", bid)
				suite.Require().NoError(err)

				// Create a normal tx
				account = suite.accounts[3]
				nonce = suite.nonces[account.Address.String()]
				numberMsgs := uint64(3)
				normalTx, err := testutils.CreateRandomTx(suite.encodingConfig.TxConfig, account, nonce, numberMsgs, timeout)
				suite.Require().NoError(err)

				txs = []sdk.Tx{freeTx, normalTx}
				auctionTxs = []sdk.Tx{bidTx}
				insertRefTxs = true
			},
			abcitypes.ResponseProcessProposal_ACCEPT,
		},
		{
			"tob, free, and default tx with default and free mixed",
			func() {
				// Create a tob tx
				bidder := suite.accounts[0]
				bid := sdk.NewCoin("stake", math.NewInt(1000))
				nonce := suite.nonces[bidder.Address.String()]
				timeout := uint64(100)
				signers := []testutils.Account{suite.accounts[2], bidder}
				bidTx, err := testutils.CreateAuctionTxWithSigners(suite.encodingConfig.TxConfig, bidder, bid, nonce, timeout, signers)
				suite.Require().NoError(err)

				// Create a free tx
				account := suite.accounts[1]
				nonce = suite.nonces[account.Address.String()]
				freeTx, err := testutils.CreateFreeTx(suite.encodingConfig.TxConfig, account, nonce, timeout, "val1", bid)
				suite.Require().NoError(err)

				// Create a normal tx
				account = suite.accounts[3]
				nonce = suite.nonces[account.Address.String()]
				numberMsgs := uint64(3)
				normalTx, err := testutils.CreateRandomTx(suite.encodingConfig.TxConfig, account, nonce, numberMsgs, timeout)
				suite.Require().NoError(err)

				txs = []sdk.Tx{normalTx, freeTx}
				auctionTxs = []sdk.Tx{bidTx}
				insertRefTxs = true
			},
			abcitypes.ResponseProcessProposal_ACCEPT,
		},
	}

	for _, tc := range cases {
		suite.Run(tc.name, func() {
			suite.SetupTest() // reset
			tc.malleate()

			// Insert all of the transactions into the proposal
			proposalTxs := make([][]byte, 0)
			for _, tx := range auctionTxs {
				txBz, err := suite.encodingConfig.TxConfig.TxEncoder()(tx)
				suite.Require().NoError(err)

				proposalTxs = append(proposalTxs, txBz)

				if insertRefTxs {
					bidInfo, err := suite.tobLane.GetAuctionBidInfo(tx)
					suite.Require().NoError(err)

					proposalTxs = append(proposalTxs, bidInfo.Transactions...)
				}
			}

			for _, tx := range txs {
				txBz, err := suite.encodingConfig.TxConfig.TxEncoder()(tx)
				suite.Require().NoError(err)

				proposalTxs = append(proposalTxs, txBz)
			}

			// create a new auction
			params := buildertypes.Params{
				MaxBundleSize:          maxBundleSize,
				ReserveFee:             reserveFee,
				FrontRunningProtection: frontRunningProtection,
			}
			suite.builderKeeper.SetParams(suite.ctx, params)
			suite.builderDecorator = ante.NewBuilderDecorator(suite.builderKeeper, suite.encodingConfig.TxConfig.TxEncoder(), suite.tobLane, suite.mempool)

			handler := suite.proposalHandler.ProcessProposalHandler()
			res, err := handler(suite.ctx, &abcitypes.RequestProcessProposal{
				Txs: proposalTxs,
			})

			// Check if the response is valid
			suite.Require().Equal(tc.response, res.Status)

			if res.Status == abcitypes.ResponseProcessProposal_ACCEPT {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}
