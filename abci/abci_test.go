package abci_test

import (
	"bytes"
	"math/rand"
	"testing"
	"time"

	abcitypes "github.com/cometbft/cometbft/abci/types"
	"github.com/cometbft/cometbft/libs/log"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/golang/mock/gomock"
	"github.com/skip-mev/pob/abci"
	"github.com/skip-mev/pob/mempool"
	testutils "github.com/skip-mev/pob/testutils"
	"github.com/skip-mev/pob/x/builder/ante"
	"github.com/skip-mev/pob/x/builder/keeper"
	buildertypes "github.com/skip-mev/pob/x/builder/types"
	"github.com/stretchr/testify/suite"
)

type ABCITestSuite struct {
	suite.Suite
	ctx sdk.Context

	// mempool setup
	mempool         *mempool.AuctionMempool
	logger          log.Logger
	encodingConfig  testutils.EncodingConfig
	proposalHandler *abci.ProposalHandler

	// auction bid setup
	auctionBidAmount sdk.Coin
	minBidIncrement  sdk.Coin

	// builder setup
	builderKeeper    keeper.Keeper
	bankKeeper       *testutils.MockBankKeeper
	accountKeeper    *testutils.MockAccountKeeper
	distrKeeper      *testutils.MockDistributionKeeper
	stakingKeeper    *testutils.MockStakingKeeper
	builderDecorator ante.BuilderDecorator
	key              *storetypes.KVStoreKey
	authorityAccount sdk.AccAddress

	// account set up
	accounts []testutils.Account
	balances sdk.Coins
	random   *rand.Rand
	nonces   map[string]uint64
}

func TestABCISuite(t *testing.T) {
	suite.Run(t, new(ABCITestSuite))
}

func (suite *ABCITestSuite) SetupTest() {
	// General config
	suite.encodingConfig = testutils.CreateTestEncodingConfig()
	suite.random = rand.New(rand.NewSource(time.Now().Unix()))
	suite.key = storetypes.NewKVStoreKey(buildertypes.StoreKey)
	testCtx := testutil.DefaultContextWithDB(suite.T(), suite.key, storetypes.NewTransientStoreKey("transient_test"))
	suite.ctx = testCtx.Ctx

	// Mempool set up
	suite.mempool = mempool.NewAuctionMempool(suite.encodingConfig.TxConfig.TxDecoder(), 0)
	suite.auctionBidAmount = sdk.NewCoin("foo", sdk.NewInt(1000000000))
	suite.minBidIncrement = sdk.NewCoin("foo", sdk.NewInt(1000))

	// Mock keepers set up
	ctrl := gomock.NewController(suite.T())
	suite.accountKeeper = testutils.NewMockAccountKeeper(ctrl)
	suite.accountKeeper.EXPECT().GetModuleAddress(buildertypes.ModuleName).Return(sdk.AccAddress{}).AnyTimes()
	suite.bankKeeper = testutils.NewMockBankKeeper(ctrl)
	suite.distrKeeper = testutils.NewMockDistributionKeeper(ctrl)
	suite.stakingKeeper = testutils.NewMockStakingKeeper(ctrl)
	suite.authorityAccount = sdk.AccAddress([]byte("authority"))

	// Builder keeper / decorator set up
	suite.builderKeeper = keeper.NewKeeper(
		suite.encodingConfig.Codec,
		suite.key,
		suite.accountKeeper,
		suite.bankKeeper,
		suite.distrKeeper,
		suite.stakingKeeper,
		suite.authorityAccount.String(),
	)
	err := suite.builderKeeper.SetParams(suite.ctx, buildertypes.DefaultParams())
	suite.Require().NoError(err)
	suite.builderDecorator = ante.NewBuilderDecorator(suite.builderKeeper, suite.encodingConfig.TxConfig.TxDecoder(), suite.encodingConfig.TxConfig.TxEncoder(), suite.mempool)

	// Accounts set up
	suite.accounts = testutils.RandomAccounts(suite.random, 1)
	suite.balances = sdk.NewCoins(sdk.NewCoin("foo", sdk.NewInt(1000000000000000000)))
	suite.nonces = make(map[string]uint64)
	for _, acc := range suite.accounts {
		suite.nonces[acc.Address.String()] = 0
	}

	// Proposal handler set up
	suite.logger = log.NewNopLogger()
	suite.proposalHandler = abci.NewProposalHandler(suite.mempool, suite.logger, suite, suite.encodingConfig.TxConfig.TxEncoder(), suite.encodingConfig.TxConfig.TxDecoder())
}

func (suite *ABCITestSuite) PrepareProposalVerifyTx(tx sdk.Tx) ([]byte, error) {
	_, err := suite.executeAnteHandler(tx)
	if err != nil {
		return nil, err
	}

	txBz, err := suite.encodingConfig.TxConfig.TxEncoder()(tx)
	if err != nil {
		return nil, err
	}

	return txBz, nil
}

func (suite *ABCITestSuite) ProcessProposalVerifyTx(txBz []byte) (sdk.Tx, error) {
	tx, err := suite.encodingConfig.TxConfig.TxDecoder()(txBz)
	if err != nil {
		return nil, err
	}

	_, err = suite.executeAnteHandler(tx)
	if err != nil {
		return tx, err
	}

	return tx, nil
}

func (suite *ABCITestSuite) executeAnteHandler(tx sdk.Tx) (sdk.Context, error) {
	signer := tx.GetMsgs()[0].GetSigners()[0]
	suite.bankKeeper.EXPECT().GetAllBalances(suite.ctx, signer).AnyTimes().Return(suite.balances)

	next := func(ctx sdk.Context, tx sdk.Tx, simulate bool) (sdk.Context, error) {
		return ctx, nil
	}

	return suite.builderDecorator.AnteHandle(suite.ctx, tx, false, next)
}

func (suite *ABCITestSuite) createFilledMempool(numNormalTxs, numAuctionTxs, numBundledTxs int, insertRefTxs bool) int {
	// Insert a bunch of normal transactions into the global mempool
	for i := 0; i < numNormalTxs; i++ {
		// randomly select an account to create the tx
		randomIndex := suite.random.Intn(len(suite.accounts))
		acc := suite.accounts[randomIndex]

		// create a few random msgs
		randomMsgs := testutils.CreateRandomMsgs(acc.Address, 3)

		nonce := suite.nonces[acc.Address.String()]
		randomTx, err := testutils.CreateTx(suite.encodingConfig.TxConfig, acc, nonce, randomMsgs)
		suite.Require().NoError(err)

		suite.nonces[acc.Address.String()]++
		priority := suite.random.Int63n(100) + 1
		suite.Require().NoError(suite.mempool.Insert(suite.ctx.WithPriority(priority), randomTx))
	}

	suite.Require().Equal(numNormalTxs, suite.mempool.CountTx())
	suite.Require().Equal(0, suite.mempool.CountAuctionTx())

	// Insert a bunch of auction transactions into the global mempool and auction mempool
	for i := 0; i < numAuctionTxs; i++ {
		// randomly select a bidder to create the tx
		randomIndex := suite.random.Intn(len(suite.accounts))
		acc := suite.accounts[randomIndex]

		// create a new auction bid msg with numBundledTxs bundled transactions
		nonce := suite.nonces[acc.Address.String()]
		bidMsg, err := testutils.CreateMsgAuctionBid(suite.encodingConfig.TxConfig, acc, suite.auctionBidAmount, nonce, numBundledTxs)
		suite.nonces[acc.Address.String()] += uint64(numBundledTxs)
		suite.Require().NoError(err)

		// create the auction tx
		nonce = suite.nonces[acc.Address.String()]
		auctionTx, err := testutils.CreateTx(suite.encodingConfig.TxConfig, acc, nonce, []sdk.Msg{bidMsg})
		suite.Require().NoError(err)

		// insert the auction tx into the global mempool
		priority := suite.random.Int63n(100) + 1
		suite.Require().NoError(suite.mempool.Insert(suite.ctx.WithPriority(priority), auctionTx))
		suite.nonces[acc.Address.String()]++

		if insertRefTxs {
			for _, refRawTx := range bidMsg.GetTransactions() {
				refTx, err := suite.encodingConfig.TxConfig.TxDecoder()(refRawTx)
				suite.Require().NoError(err)
				priority := suite.random.Int63n(100) + 1
				suite.Require().NoError(suite.mempool.Insert(suite.ctx.WithPriority(priority), refTx))
			}
		}

		// decrement the bid amount for the next auction tx
		suite.auctionBidAmount = suite.auctionBidAmount.Sub(suite.minBidIncrement)
	}

	numSeenGlobalTxs := 0
	for iterator := suite.mempool.Select(suite.ctx, nil); iterator != nil; iterator = iterator.Next() {
		numSeenGlobalTxs++
	}

	numSeenAuctionTxs := 0
	for iterator := suite.mempool.AuctionBidSelect(suite.ctx); iterator != nil; iterator = iterator.Next() {
		numSeenAuctionTxs++
	}

	var totalNumTxs int
	suite.Require().Equal(numAuctionTxs, suite.mempool.CountAuctionTx())
	if insertRefTxs {
		totalNumTxs = numNormalTxs + numAuctionTxs*(numBundledTxs+1)
		suite.Require().Equal(totalNumTxs, suite.mempool.CountTx())
		suite.Require().Equal(totalNumTxs, numSeenGlobalTxs)
	} else {
		totalNumTxs = numNormalTxs + numAuctionTxs
		suite.Require().Equal(totalNumTxs, suite.mempool.CountTx())
		suite.Require().Equal(totalNumTxs, numSeenGlobalTxs)
	}

	suite.Require().Equal(numAuctionTxs, numSeenAuctionTxs)

	return totalNumTxs
}

func (suite *ABCITestSuite) exportMempool(exportRefTxs bool) [][]byte {
	txs := make([][]byte, 0)
	seenTxs := make(map[string]bool)

	auctionIterator := suite.mempool.AuctionBidSelect(suite.ctx)
	for ; auctionIterator != nil; auctionIterator = auctionIterator.Next() {
		auctionTx := auctionIterator.Tx()
		txBz, err := suite.encodingConfig.TxConfig.TxEncoder()(auctionTx)
		suite.Require().NoError(err)

		txs = append(txs, txBz)

		if exportRefTxs {
			for _, refRawTx := range auctionTx.GetMsgs()[0].(*buildertypes.MsgAuctionBid).GetTransactions() {
				txs = append(txs, refRawTx)
				seenTxs[string(refRawTx)] = true
			}
		}

		seenTxs[string(txBz)] = true
	}

	iterator := suite.mempool.Select(suite.ctx, nil)
	for ; iterator != nil; iterator = iterator.Next() {
		txBz, err := suite.encodingConfig.TxConfig.TxEncoder()(iterator.Tx())
		suite.Require().NoError(err)

		if !seenTxs[string(txBz)] {
			txs = append(txs, txBz)
		}
	}

	return txs
}

func (suite *ABCITestSuite) TestPrepareProposal() {
	var (
		// the modified transactions cannot exceed this size
		maxTxBytes int64 = 1000000000000000000

		// mempool configuration
		numNormalTxs  = 100
		numAuctionTxs = 100
		numBundledTxs = 3
		insertRefTxs  = false

		// auction configuration
		maxBundleSize          uint32 = 10
		reserveFee                    = sdk.NewCoin("foo", sdk.NewInt(1000))
		minBuyInFee                   = sdk.NewCoin("foo", sdk.NewInt(1000))
		frontRunningProtection        = true
	)

	cases := []struct {
		name                       string
		malleate                   func()
		expectedNumberProposalTxs  int
		expectedNumberTxsInMempool int
		isTopBidValid              bool
	}{
		{
			"single bundle in the mempool",
			func() {
				numNormalTxs = 0
				numAuctionTxs = 1
				numBundledTxs = 3
				insertRefTxs = true
			},
			4,
			4,
			true,
		},
		{
			"single bundle in the mempool, no ref txs in mempool",
			func() {
				numNormalTxs = 0
				numAuctionTxs = 1
				numBundledTxs = 3
				insertRefTxs = false
			},
			4,
			1,
			true,
		},
		{
			"single bundle in the mempool, not valid",
			func() {
				reserveFee = sdk.NewCoin("foo", sdk.NewInt(100000))
				suite.auctionBidAmount = sdk.NewCoin("foo", sdk.NewInt(10000)) // this will fail the ante handler
				numNormalTxs = 0
				numAuctionTxs = 1
				numBundledTxs = 3
			},
			0,
			0,
			false,
		},
		{
			"single bundle in the mempool, not valid with ref txs in mempool",
			func() {
				reserveFee = sdk.NewCoin("foo", sdk.NewInt(100000))
				suite.auctionBidAmount = sdk.NewCoin("foo", sdk.NewInt(10000)) // this will fail the ante handler
				numNormalTxs = 0
				numAuctionTxs = 1
				numBundledTxs = 3
				insertRefTxs = true
			},
			3,
			3,
			false,
		},
		{
			"multiple bundles in the mempool, no normal txs + no ref txs in mempool",
			func() {
				reserveFee = sdk.NewCoin("foo", sdk.NewInt(1000))
				suite.auctionBidAmount = sdk.NewCoin("foo", sdk.NewInt(10000000))
				numNormalTxs = 0
				numAuctionTxs = 10
				numBundledTxs = 3
				insertRefTxs = false
			},
			4,
			1,
			true,
		},
		{
			"multiple bundles in the mempool, no normal txs + ref txs in mempool",
			func() {
				numNormalTxs = 0
				numAuctionTxs = 10
				numBundledTxs = 3
				insertRefTxs = true
			},
			31,
			31,
			true,
		},
		{
			"normal txs only",
			func() {
				numNormalTxs = 1
				numAuctionTxs = 0
				numBundledTxs = 0
			},
			1,
			1,
			false,
		},
		{
			"many normal txs only",
			func() {
				numNormalTxs = 100
				numAuctionTxs = 0
				numBundledTxs = 0
			},
			100,
			100,
			false,
		},
		{
			"single normal tx, single auction tx",
			func() {
				numNormalTxs = 1
				numAuctionTxs = 1
				numBundledTxs = 0
			},
			2,
			2,
			true,
		},
		{
			"single normal tx, single auction tx with ref txs",
			func() {
				numNormalTxs = 1
				numAuctionTxs = 1
				numBundledTxs = 3
				insertRefTxs = false
			},
			5,
			2,
			true,
		},
		{
			"single normal tx, single failing auction tx with ref txs",
			func() {
				numNormalTxs = 1
				numAuctionTxs = 1
				numBundledTxs = 3
				insertRefTxs = true
				suite.auctionBidAmount = sdk.NewCoin("foo", sdk.NewInt(2000)) // this will fail the ante handler
				reserveFee = sdk.NewCoin("foo", sdk.NewInt(1000000000))
			},
			4,
			4,
			false,
		},
		{
			"many normal tx, single auction tx with no ref txs",
			func() {
				reserveFee = sdk.NewCoin("foo", sdk.NewInt(1000))
				suite.auctionBidAmount = sdk.NewCoin("foo", sdk.NewInt(2000000))
				numNormalTxs = 100
				numAuctionTxs = 1
				numBundledTxs = 0
			},
			101,
			101,
			true,
		},
		{
			"many normal tx, single auction tx with ref txs",
			func() {
				numNormalTxs = 100
				numAuctionTxs = 1
				numBundledTxs = 3
				insertRefTxs = true
			},
			104,
			104,
			true,
		},
		{
			"many normal tx, single auction tx with ref txs",
			func() {
				numNormalTxs = 100
				numAuctionTxs = 1
				numBundledTxs = 3
				insertRefTxs = false
			},
			104,
			101,
			true,
		},
		{
			"many normal tx, many auction tx with ref txs",
			func() {
				numNormalTxs = 100
				numAuctionTxs = 100
				numBundledTxs = 1
				insertRefTxs = true
			},
			201,
			201,
			true,
		},
	}

	for _, tc := range cases {
		suite.Run(tc.name, func() {
			suite.SetupTest() // reset
			tc.malleate()

			suite.createFilledMempool(numNormalTxs, numAuctionTxs, numBundledTxs, insertRefTxs)

			// create a new auction
			params := buildertypes.Params{
				MaxBundleSize:          maxBundleSize,
				ReserveFee:             reserveFee,
				MinBuyInFee:            minBuyInFee,
				FrontRunningProtection: frontRunningProtection,
				MinBidIncrement:        suite.minBidIncrement,
			}
			suite.builderKeeper.SetParams(suite.ctx, params)
			suite.builderDecorator = ante.NewBuilderDecorator(suite.builderKeeper, suite.encodingConfig.TxConfig.TxDecoder(), suite.encodingConfig.TxConfig.TxEncoder(), suite.mempool)

			handler := suite.proposalHandler.PrepareProposalHandler()
			res := handler(suite.ctx, abcitypes.RequestPrepareProposal{
				MaxTxBytes: maxTxBytes,
			})

			// -------------------- Check Invariants -------------------- //
			// 1. The auction tx must fail if we know it is invalid
			suite.Require().Equal(tc.isTopBidValid, suite.isTopBidValid())

			// 2. total bytes must be less than or equal to maxTxBytes
			totalBytes := int64(0)
			if suite.isTopBidValid() {
				totalBytes += int64(len(res.Txs[0]))

				for _, tx := range res.Txs[1+numBundledTxs:] {
					totalBytes += int64(len(tx))
				}
			} else {
				for _, tx := range res.Txs {
					totalBytes += int64(len(tx))
				}
			}
			suite.Require().LessOrEqual(totalBytes, maxTxBytes)

			// 3. the number of transactions in the response must be equal to the number of expected transactions
			suite.Require().Equal(tc.expectedNumberProposalTxs, len(res.Txs))

			// 4. if there are auction transactions, the first transaction must be the top bid
			// and the rest of the bundle must be in the response
			if suite.isTopBidValid() {
				auctionTx, err := suite.encodingConfig.TxConfig.TxDecoder()(res.Txs[0])
				suite.Require().NoError(err)

				msgAuctionBid, err := mempool.GetMsgAuctionBidFromTx(auctionTx)
				suite.Require().NoError(err)

				for index, tx := range msgAuctionBid.GetTransactions() {
					suite.Require().Equal(tx, res.Txs[index+1])
				}
			}

			// 5. All of the transactions must be unique
			uniqueTxs := make(map[string]bool)
			for _, tx := range res.Txs {
				suite.Require().False(uniqueTxs[string(tx)])
				uniqueTxs[string(tx)] = true
			}

			// 6. The number of transactions in the mempool must be correct
			suite.Require().Equal(tc.expectedNumberTxsInMempool, suite.mempool.CountTx())
		})
	}
}

func (suite *ABCITestSuite) TestProcessProposal() {
	var (
		// mempool set up
		numNormalTxs   = 100
		numAuctionTxs  = 1
		numBundledTxs  = 3
		insertRefTxs   = true
		exportRefTxs   = true
		frontRunningTx sdk.Tx

		// auction set up
		maxBundleSize          uint32 = 10
		reserveFee                    = sdk.NewCoin("foo", sdk.NewInt(1000))
		minBuyInFee                   = sdk.NewCoin("foo", sdk.NewInt(1000))
		frontRunningProtection        = true
	)

	cases := []struct {
		name          string
		malleate      func()
		isTopBidValid bool
		response      abcitypes.ResponseProcessProposal_ProposalStatus
	}{
		{
			"single normal tx, no auction tx",
			func() {
				numNormalTxs = 1
				numAuctionTxs = 0
				numBundledTxs = 0
			},
			false,
			abcitypes.ResponseProcessProposal_ACCEPT,
		},
		{
			"single auction tx, no normal txs",
			func() {
				numNormalTxs = 0
				numAuctionTxs = 1
				numBundledTxs = 0
			},
			true,
			abcitypes.ResponseProcessProposal_ACCEPT,
		},
		{
			"single auction tx, single auction tx",
			func() {
				numNormalTxs = 1
				numAuctionTxs = 1
				numBundledTxs = 0
			},
			true,
			abcitypes.ResponseProcessProposal_ACCEPT,
		},
		{
			"single auction tx, single auction tx with ref txs",
			func() {
				numNormalTxs = 1
				numAuctionTxs = 1
				numBundledTxs = 4
			},
			true,
			abcitypes.ResponseProcessProposal_ACCEPT,
		},
		{
			"single auction tx, single auction tx with no ref txs",
			func() {
				numNormalTxs = 1
				numAuctionTxs = 1
				numBundledTxs = 4
				insertRefTxs = false
				exportRefTxs = false
			},
			true,
			abcitypes.ResponseProcessProposal_REJECT,
		},
		{
			"multiple auction txs, single normal tx",
			func() {
				numNormalTxs = 1
				numAuctionTxs = 2
				numBundledTxs = 4
				insertRefTxs = true
				exportRefTxs = true
			},
			true,
			abcitypes.ResponseProcessProposal_REJECT,
		},
		{
			"single auction txs, multiple normal tx",
			func() {
				numNormalTxs = 100
				numAuctionTxs = 1
				numBundledTxs = 4
			},
			true,
			abcitypes.ResponseProcessProposal_ACCEPT,
		},
		{
			"single invalid auction tx, multiple normal tx",
			func() {
				numNormalTxs = 100
				numAuctionTxs = 1
				numBundledTxs = 4
				reserveFee = sdk.NewCoin("foo", sdk.NewInt(100000000000000000))
				insertRefTxs = true
			},
			false,
			abcitypes.ResponseProcessProposal_REJECT,
		},
		{
			"single valid auction txs but missing ref txs",
			func() {
				numNormalTxs = 0
				numAuctionTxs = 1
				numBundledTxs = 4
				reserveFee = sdk.NewCoin("foo", sdk.NewInt(1000))
				insertRefTxs = false
				exportRefTxs = false
			},
			true,
			abcitypes.ResponseProcessProposal_REJECT,
		},
		{
			"single valid auction txs but missing ref txs, with many normal txs",
			func() {
				numNormalTxs = 100
				numAuctionTxs = 1
				numBundledTxs = 4
				reserveFee = sdk.NewCoin("foo", sdk.NewInt(1000))
				insertRefTxs = false
				exportRefTxs = false
			},
			true,
			abcitypes.ResponseProcessProposal_REJECT,
		},
		{
			"auction tx with frontrunning",
			func() {
				randomAccount := testutils.RandomAccounts(suite.random, 1)[0]
				bidder := suite.accounts[0]
				bid := sdk.NewCoin("foo", sdk.NewInt(696969696969))
				nonce := suite.nonces[bidder.Address.String()]
				frontRunningTx, _ = testutils.CreateAuctionTxWithSigners(suite.encodingConfig.TxConfig, suite.accounts[0], bid, nonce+1, []testutils.Account{bidder, randomAccount})
				suite.Require().NotNil(frontRunningTx)

				numNormalTxs = 100
				numAuctionTxs = 1
				numBundledTxs = 4
				insertRefTxs = true
				exportRefTxs = true
			},
			false,
			abcitypes.ResponseProcessProposal_REJECT,
		},
		{
			"auction tx with frontrunning, but frontrunning protection disabled",
			func() {
				randomAccount := testutils.RandomAccounts(suite.random, 1)[0]
				bidder := suite.accounts[0]
				bid := sdk.NewCoin("foo", sdk.NewInt(696969696969))
				nonce := suite.nonces[bidder.Address.String()]
				frontRunningTx, _ = testutils.CreateAuctionTxWithSigners(suite.encodingConfig.TxConfig, suite.accounts[0], bid, nonce+1, []testutils.Account{bidder, randomAccount})
				suite.Require().NotNil(frontRunningTx)

				numAuctionTxs = 0
				frontRunningProtection = false
			},
			true,
			abcitypes.ResponseProcessProposal_ACCEPT,
		},
	}

	for _, tc := range cases {
		suite.Run(tc.name, func() {
			suite.SetupTest() // reset
			tc.malleate()

			suite.createFilledMempool(numNormalTxs, numAuctionTxs, numBundledTxs, insertRefTxs)

			if frontRunningTx != nil {
				suite.Require().NoError(suite.mempool.Insert(suite.ctx, frontRunningTx))
			}

			// create a new auction
			params := buildertypes.Params{
				MaxBundleSize:          maxBundleSize,
				ReserveFee:             reserveFee,
				MinBuyInFee:            minBuyInFee,
				FrontRunningProtection: frontRunningProtection,
				MinBidIncrement:        suite.minBidIncrement,
			}
			suite.builderKeeper.SetParams(suite.ctx, params)
			suite.builderDecorator = ante.NewBuilderDecorator(suite.builderKeeper, suite.encodingConfig.TxConfig.TxDecoder(), suite.encodingConfig.TxConfig.TxEncoder(), suite.mempool)
			suite.Require().Equal(tc.isTopBidValid, suite.isTopBidValid())

			txs := suite.exportMempool(exportRefTxs)

			if frontRunningTx != nil {
				txBz, err := suite.encodingConfig.TxConfig.TxEncoder()(frontRunningTx)
				suite.Require().NoError(err)

				suite.Require().True(bytes.Equal(txs[0], txBz))
			}

			handler := suite.proposalHandler.ProcessProposalHandler()
			res := handler(suite.ctx, abcitypes.RequestProcessProposal{
				Txs: txs,
			})

			// Check if the response is valid
			suite.Require().Equal(tc.response, res.Status)
		})
	}
}

// isTopBidValid returns true if the top bid is valid. We purposefully insert invalid
// auction transactions into the mempool to test the handlers.
func (suite *ABCITestSuite) isTopBidValid() bool {
	iterator := suite.mempool.AuctionBidSelect(suite.ctx)
	if iterator == nil {
		return false
	}

	// check if the top bid is valid
	_, err := suite.executeAnteHandler(iterator.Tx())
	return err == nil
}
