package abci_test

import (
	"bytes"

	abcitypes "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/skip-mev/pob/abci"
	testutils "github.com/skip-mev/pob/testutils"
	"github.com/skip-mev/pob/x/builder/ante"
	buildertypes "github.com/skip-mev/pob/x/builder/types"
)

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
			3,
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
			0,
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
			0,
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
			30,
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
			1,
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
			1,
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
			100,
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
			103,
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
			100,
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
			200,
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

			// reset the proposal handler with the new mempool
			suite.proposalHandler = abci.NewProposalHandler(suite.mempool, suite.logger, suite.anteHandler, suite.encodingConfig.TxConfig.TxEncoder(), suite.encodingConfig.TxConfig.TxDecoder())

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

				bidInfo, err := suite.mempool.GetAuctionBidInfo(auctionTx)
				suite.Require().NoError(err)

				for index, tx := range bidInfo.Transactions {
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
				frontRunningTx, _ = testutils.CreateAuctionTxWithSigners(suite.encodingConfig.TxConfig, suite.accounts[0], bid, nonce+1, 1000, []testutils.Account{bidder, randomAccount})
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
				frontRunningTx, _ = testutils.CreateAuctionTxWithSigners(suite.encodingConfig.TxConfig, suite.accounts[0], bid, nonce+1, 1000, []testutils.Account{bidder, randomAccount})
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

			// reset the proposal handler with the new mempool
			suite.proposalHandler = abci.NewProposalHandler(suite.mempool, suite.logger, suite.anteHandler, suite.encodingConfig.TxConfig.TxEncoder(), suite.encodingConfig.TxConfig.TxDecoder())

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
	_, err := suite.anteHandler(suite.ctx, iterator.Tx(), true)
	return err == nil
}
