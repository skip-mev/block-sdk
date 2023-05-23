package v2_test

import (
	comettypes "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/skip-mev/pob/abci"
	v2 "github.com/skip-mev/pob/abci/v2"
	testutils "github.com/skip-mev/pob/testutils"
	"github.com/skip-mev/pob/x/builder/ante"
	buildertypes "github.com/skip-mev/pob/x/builder/types"
)

func (suite *ABCITestSuite) TestPrepareProposal() {
	var (
		// the modified transactions cannot exceed this size
		maxTxBytes int64 = 1000000000000000000

		// mempool configuration
		numNormalTxs         = 100
		numAuctionTxs        = 100
		numBundledTxs        = 3
		insertRefTxs         = false
		expectedTopAuctionTx sdk.Tx

		// auction configuration
		maxBundleSize          uint32 = 10
		reserveFee                    = sdk.NewCoin("foo", sdk.NewInt(1000))
		frontRunningProtection        = true
	)

	cases := []struct {
		name                              string
		malleate                          func()
		expectedNumberProposalTxs         int
		expectedNumberTxsInMempool        int
		expectedNumberTxsInAuctionMempool int
	}{
		{
			"single bundle in the mempool",
			func() {
				numNormalTxs = 0
				numAuctionTxs = 1
				numBundledTxs = 3
				insertRefTxs = true

				suite.createFilledMempool(numNormalTxs, numAuctionTxs, numBundledTxs, insertRefTxs)

				expectedTopAuctionTx = suite.mempool.GetTopAuctionTx(suite.ctx)
			},
			5,
			3,
			1,
		},
		{
			"single bundle in the mempool, no ref txs in mempool",
			func() {
				numNormalTxs = 0
				numAuctionTxs = 1
				numBundledTxs = 3
				insertRefTxs = false

				suite.createFilledMempool(numNormalTxs, numAuctionTxs, numBundledTxs, insertRefTxs)

				expectedTopAuctionTx = suite.mempool.GetTopAuctionTx(suite.ctx)
			},
			5,
			0,
			1,
		},
		{
			"single bundle in the mempool, not valid",
			func() {
				reserveFee = sdk.NewCoin("foo", sdk.NewInt(100000))
				suite.auctionBidAmount = sdk.NewCoin("foo", sdk.NewInt(10000)) // this will fail the ante handler
				numNormalTxs = 0
				numAuctionTxs = 1
				numBundledTxs = 3

				suite.createFilledMempool(numNormalTxs, numAuctionTxs, numBundledTxs, insertRefTxs)

				expectedTopAuctionTx = nil
			},
			1,
			0,
			0,
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

				suite.createFilledMempool(numNormalTxs, numAuctionTxs, numBundledTxs, insertRefTxs)

				expectedTopAuctionTx = nil
			},
			4,
			3,
			0,
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

				suite.createFilledMempool(numNormalTxs, numAuctionTxs, numBundledTxs, insertRefTxs)

				expectedTopAuctionTx = suite.mempool.GetTopAuctionTx(suite.ctx)
			},
			5,
			0,
			10,
		},
		{
			"multiple bundles in the mempool, normal txs + ref txs in mempool",
			func() {
				numNormalTxs = 0
				numAuctionTxs = 10
				numBundledTxs = 3
				insertRefTxs = true

				suite.createFilledMempool(numNormalTxs, numAuctionTxs, numBundledTxs, insertRefTxs)

				expectedTopAuctionTx = suite.mempool.GetTopAuctionTx(suite.ctx)
			},
			32,
			30,
			10,
		},
		{
			"normal txs only",
			func() {
				numNormalTxs = 1
				numAuctionTxs = 0
				numBundledTxs = 0

				suite.createFilledMempool(numNormalTxs, numAuctionTxs, numBundledTxs, insertRefTxs)

				expectedTopAuctionTx = suite.mempool.GetTopAuctionTx(suite.ctx)
			},
			2,
			1,
			0,
		},
		{
			"many normal txs only",
			func() {
				numNormalTxs = 100
				numAuctionTxs = 0
				numBundledTxs = 0

				suite.createFilledMempool(numNormalTxs, numAuctionTxs, numBundledTxs, insertRefTxs)

				expectedTopAuctionTx = suite.mempool.GetTopAuctionTx(suite.ctx)
			},
			101,
			100,
			0,
		},
		{
			"single normal tx, single auction tx",
			func() {
				numNormalTxs = 1
				numAuctionTxs = 1
				numBundledTxs = 0

				suite.createFilledMempool(numNormalTxs, numAuctionTxs, numBundledTxs, insertRefTxs)

				expectedTopAuctionTx = suite.mempool.GetTopAuctionTx(suite.ctx)
			},
			3,
			1,
			1,
		},
		{
			"single normal tx, single auction tx with ref txs",
			func() {
				numNormalTxs = 1
				numAuctionTxs = 1
				numBundledTxs = 3
				insertRefTxs = true

				suite.createFilledMempool(numNormalTxs, numAuctionTxs, numBundledTxs, insertRefTxs)

				expectedTopAuctionTx = suite.mempool.GetTopAuctionTx(suite.ctx)
			},
			6,
			4,
			1,
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

				suite.createFilledMempool(numNormalTxs, numAuctionTxs, numBundledTxs, insertRefTxs)

				expectedTopAuctionTx = nil
			},
			5,
			4,
			0,
		},
		{
			"many normal tx, single auction tx with no ref txs",
			func() {
				reserveFee = sdk.NewCoin("foo", sdk.NewInt(1000))
				suite.auctionBidAmount = sdk.NewCoin("foo", sdk.NewInt(2000000))
				numNormalTxs = 100
				numAuctionTxs = 1
				numBundledTxs = 0

				suite.createFilledMempool(numNormalTxs, numAuctionTxs, numBundledTxs, insertRefTxs)

				expectedTopAuctionTx = nil
			},
			102,
			100,
			1,
		},
		{
			"many normal tx, single auction tx with ref txs",
			func() {
				numNormalTxs = 100
				numAuctionTxs = 100
				numBundledTxs = 3
				insertRefTxs = true

				suite.createFilledMempool(numNormalTxs, numAuctionTxs, numBundledTxs, insertRefTxs)

				expectedTopAuctionTx = suite.mempool.GetTopAuctionTx(suite.ctx)
			},
			402,
			400,
			100,
		},
		{
			"many normal tx, many auction tx with ref txs but top bid is invalid",
			func() {
				numNormalTxs = 100
				numAuctionTxs = 100
				numBundledTxs = 1
				insertRefTxs = true

				suite.createFilledMempool(numNormalTxs, numAuctionTxs, numBundledTxs, insertRefTxs)

				expectedTopAuctionTx = suite.mempool.GetTopAuctionTx(suite.ctx)

				// create a new bid that is greater than the current top bid
				bid := sdk.NewCoin("foo", sdk.NewInt(200000000000000000))
				bidTx, err := testutils.CreateAuctionTxWithSigners(
					suite.encodingConfig.TxConfig,
					suite.accounts[0],
					bid,
					0,
					0,
					[]testutils.Account{suite.accounts[0], suite.accounts[1]},
				)
				suite.Require().NoError(err)

				// add the new bid to the mempool
				err = suite.mempool.Insert(suite.ctx, bidTx)
				suite.Require().NoError(err)

				suite.Require().Equal(suite.mempool.CountAuctionTx(), 101)
			},
			202,
			200,
			100,
		},
	}

	for _, tc := range cases {
		suite.Run(tc.name, func() {
			tc.malleate()

			// Create a new auction.
			params := buildertypes.Params{
				MaxBundleSize:          maxBundleSize,
				ReserveFee:             reserveFee,
				FrontRunningProtection: frontRunningProtection,
				MinBidIncrement:        suite.minBidIncrement,
			}
			suite.builderKeeper.SetParams(suite.ctx, params)
			suite.builderDecorator = ante.NewBuilderDecorator(suite.builderKeeper, suite.encodingConfig.TxConfig.TxEncoder(), suite.mempool)

			// Reset the proposal handler with the new mempool.
			suite.proposalHandler = v2.NewProposalHandler(suite.mempool, suite.logger, suite.anteHandler, suite.encodingConfig.TxConfig.TxEncoder(), suite.encodingConfig.TxConfig.TxDecoder())

			// Create a prepare proposal request based on the current state of the mempool.
			handler := suite.proposalHandler.PrepareProposalHandler()
			req := suite.createPrepareProposalRequest(maxTxBytes)
			res := handler(suite.ctx, req)

			// -------------------- Check Invariants -------------------- //
			// The first slot in the proposal must be the auction info
			auctionInfo := abci.AuctionInfo{}
			err := auctionInfo.Unmarshal(res.Txs[v2.AuctionInfoIndex])
			suite.Require().NoError(err)

			// Total bytes must be less than or equal to maxTxBytes
			totalBytes := int64(0)
			for _, tx := range res.Txs[v2.NumInjectedTxs:] {
				totalBytes += int64(len(tx))
			}
			suite.Require().LessOrEqual(totalBytes, maxTxBytes)

			// The number of transactions in the response must be equal to the number of expected transactions
			suite.Require().Equal(tc.expectedNumberProposalTxs, len(res.Txs))

			// If there are auction transactions, the first transaction must be the top bid
			// and the rest of the bundle must be in the response
			if expectedTopAuctionTx != nil {
				auctionTx, err := suite.encodingConfig.TxConfig.TxDecoder()(res.Txs[1])
				suite.Require().NoError(err)

				bidInfo, err := suite.mempool.GetAuctionBidInfo(auctionTx)
				suite.Require().NoError(err)

				for index, tx := range bidInfo.Transactions {
					suite.Require().Equal(tx, res.Txs[v2.NumInjectedTxs+index+1])
				}
			}

			// 5. All of the transactions must be unique
			uniqueTxs := make(map[string]bool)
			for _, tx := range res.Txs[v2.NumInjectedTxs:] {
				suite.Require().False(uniqueTxs[string(tx)])
				uniqueTxs[string(tx)] = true
			}

			// 6. The number of transactions in the mempool must be correct
			suite.Require().Equal(tc.expectedNumberTxsInMempool, suite.mempool.CountTx())
			suite.Require().Equal(tc.expectedNumberTxsInAuctionMempool, suite.mempool.CountAuctionTx())
		})
	}
}

func (suite *ABCITestSuite) TestProcessProposal() {
	var (
		// mempool set up
		numNormalTxs  = 100
		numAuctionTxs = 1
		numBundledTxs = 3
		insertRefTxs  = false

		// auction set up
		maxBundleSize uint32 = 10
		reserveFee           = sdk.NewCoin("foo", sdk.NewInt(1000))
	)

	cases := []struct {
		name      string
		createTxs func() [][]byte
		response  comettypes.ResponseProcessProposal_ProposalStatus
	}{
		{
			"single normal tx, no vote extension info",
			func() [][]byte {
				numNormalTxs = 1
				numAuctionTxs = 0
				numBundledTxs = 0

				suite.createFilledMempool(numNormalTxs, numAuctionTxs, numBundledTxs, insertRefTxs)

				txs := suite.exportMempool()

				return txs
			},
			comettypes.ResponseProcessProposal_REJECT,
		},
		{
			"single auction tx, no vote extension info",
			func() [][]byte {
				numNormalTxs = 0
				numAuctionTxs = 1
				numBundledTxs = 0

				suite.createFilledMempool(numNormalTxs, numAuctionTxs, numBundledTxs, insertRefTxs)

				return suite.exportMempool()
			},
			comettypes.ResponseProcessProposal_REJECT,
		},
		{
			"single auction tx, single auction tx, no vote extension info",
			func() [][]byte {
				numNormalTxs = 1
				numAuctionTxs = 1
				numBundledTxs = 0

				suite.createFilledMempool(numNormalTxs, numAuctionTxs, numBundledTxs, insertRefTxs)

				return suite.exportMempool()
			},
			comettypes.ResponseProcessProposal_REJECT,
		},
		{
			"single auction tx with ref txs (no unwrapping)",
			func() [][]byte {
				numNormalTxs = 1
				numAuctionTxs = 1
				numBundledTxs = 4

				suite.createFilledMempool(numNormalTxs, numAuctionTxs, numBundledTxs, insertRefTxs)

				topAuctionTx := suite.mempool.GetTopAuctionTx(suite.ctx)
				suite.Require().NotNil(topAuctionTx)

				txBz, err := suite.encodingConfig.TxConfig.TxEncoder()(topAuctionTx)
				suite.Require().NoError(err)

				auctionInfo := suite.createAuctionInfoFromTxBzs([][]byte{txBz}, 5)

				proposal := append([][]byte{
					auctionInfo,
					txBz,
				}, suite.exportMempool()...)

				return proposal
			},
			comettypes.ResponseProcessProposal_REJECT,
		},
		{
			"single auction tx with ref txs (with unwrapping)",
			func() [][]byte {
				numNormalTxs = 0
				numAuctionTxs = 1
				numBundledTxs = 4
				insertRefTxs = false

				suite.createFilledMempool(numNormalTxs, numAuctionTxs, numBundledTxs, insertRefTxs)

				topAuctionTx := suite.mempool.GetTopAuctionTx(suite.ctx)
				suite.Require().NotNil(topAuctionTx)

				bidInfo, err := suite.mempool.GetAuctionBidInfo(topAuctionTx)
				suite.Require().NoError(err)

				txBz, err := suite.encodingConfig.TxConfig.TxEncoder()(topAuctionTx)
				suite.Require().NoError(err)

				auctionInfo := suite.createAuctionInfoFromTxBzs([][]byte{txBz}, 5)

				proposal := append([][]byte{
					auctionInfo,
					txBz,
				}, bidInfo.Transactions...)

				return proposal
			},
			comettypes.ResponseProcessProposal_ACCEPT,
		},
		{
			"single auction tx but no inclusion of ref txs",
			func() [][]byte {
				numNormalTxs = 0
				numAuctionTxs = 1
				numBundledTxs = 4
				insertRefTxs = false

				suite.createFilledMempool(numNormalTxs, numAuctionTxs, numBundledTxs, insertRefTxs)

				topAuctionTx := suite.mempool.GetTopAuctionTx(suite.ctx)
				suite.Require().NotNil(topAuctionTx)

				txBz, err := suite.encodingConfig.TxConfig.TxEncoder()(topAuctionTx)
				suite.Require().NoError(err)

				auctionInfo := suite.createAuctionInfoFromTxBzs([][]byte{txBz}, 5)

				return [][]byte{
					auctionInfo,
					txBz,
				}
			},
			comettypes.ResponseProcessProposal_REJECT,
		},
		{
			"single auction tx, but auction tx is not valid",
			func() [][]byte {
				tx, err := testutils.CreateAuctionTxWithSigners(
					suite.encodingConfig.TxConfig,
					suite.accounts[0],
					sdk.NewCoin("foo", sdk.NewInt(100)),
					1,
					0, // invalid timeout
					[]testutils.Account{},
				)
				suite.Require().NoError(err)

				txBz, err := suite.encodingConfig.TxConfig.TxEncoder()(tx)
				suite.Require().NoError(err)

				auctionInfoBz := suite.createAuctionInfoFromTxBzs([][]byte{txBz}, 1)

				return [][]byte{
					auctionInfoBz,
					txBz,
				}
			},
			comettypes.ResponseProcessProposal_REJECT,
		},
		{
			"single auction tx with ref txs, but auction tx is not valid",
			func() [][]byte {
				tx, err := testutils.CreateAuctionTxWithSigners(
					suite.encodingConfig.TxConfig,
					suite.accounts[0],
					sdk.NewCoin("foo", sdk.NewInt(100)),
					1,
					1,
					[]testutils.Account{suite.accounts[1], suite.accounts[1], suite.accounts[0]},
				)
				suite.Require().NoError(err)

				txBz, err := suite.encodingConfig.TxConfig.TxEncoder()(tx)
				suite.Require().NoError(err)

				auctionInfoBz := suite.createAuctionInfoFromTxBzs([][]byte{txBz}, 4)

				bidInfo, err := suite.mempool.GetAuctionBidInfo(tx)
				suite.Require().NoError(err)

				return append([][]byte{
					auctionInfoBz,
					txBz,
				}, bidInfo.Transactions...)
			},
			comettypes.ResponseProcessProposal_REJECT,
		},
		{
			"multiple auction txs but wrong auction tx is at top of block",
			func() [][]byte {
				numNormalTxs = 0
				numAuctionTxs = 2
				numBundledTxs = 0
				insertRefTxs = false

				suite.createFilledMempool(numNormalTxs, numAuctionTxs, numBundledTxs, insertRefTxs)

				_, auctionTxBzs := suite.getAllAuctionTxs()

				auctionInfo := suite.createAuctionInfoFromTxBzs(auctionTxBzs, 1)

				proposal := [][]byte{
					auctionInfo,
					auctionTxBzs[1],
				}

				return proposal
			},
			comettypes.ResponseProcessProposal_REJECT,
		},
		{
			"multiple auction txs included in block",
			func() [][]byte {
				numNormalTxs = 0
				numAuctionTxs = 2
				numBundledTxs = 0
				insertRefTxs = false

				suite.createFilledMempool(numNormalTxs, numAuctionTxs, numBundledTxs, insertRefTxs)

				_, auctionTxBzs := suite.getAllAuctionTxs()

				auctionInfo := suite.createAuctionInfoFromTxBzs(auctionTxBzs, 1)

				proposal := [][]byte{
					auctionInfo,
					auctionTxBzs[0],
					auctionTxBzs[1],
				}

				return proposal
			},
			comettypes.ResponseProcessProposal_REJECT,
		},
		{
			"single auction tx, but rest of the mempool is invalid",
			func() [][]byte {
				numNormalTxs = 0
				numAuctionTxs = 1
				numBundledTxs = 0
				insertRefTxs = false

				suite.createFilledMempool(numNormalTxs, numAuctionTxs, numBundledTxs, insertRefTxs)

				topAuctionTx := suite.mempool.GetTopAuctionTx(suite.ctx)
				suite.Require().NotNil(topAuctionTx)

				txBz, err := suite.encodingConfig.TxConfig.TxEncoder()(topAuctionTx)
				suite.Require().NoError(err)

				auctionInfo := suite.createAuctionInfoFromTxBzs([][]byte{txBz}, 1)

				proposal := [][]byte{
					auctionInfo,
					txBz,
					[]byte("invalid tx"),
				}

				return proposal
			},
			comettypes.ResponseProcessProposal_REJECT,
		},
		{
			"single auction tx with filled mempool, but rest of the mempool is invalid",
			func() [][]byte {
				numNormalTxs = 100
				numAuctionTxs = 1
				numBundledTxs = 0
				insertRefTxs = false

				suite.createFilledMempool(numNormalTxs, numAuctionTxs, numBundledTxs, insertRefTxs)

				topAuctionTx := suite.mempool.GetTopAuctionTx(suite.ctx)
				suite.Require().NotNil(topAuctionTx)

				txBz, err := suite.encodingConfig.TxConfig.TxEncoder()(topAuctionTx)
				suite.Require().NoError(err)

				auctionInfo := suite.createAuctionInfoFromTxBzs([][]byte{txBz}, 1)

				proposal := append([][]byte{
					auctionInfo,
					txBz,
				}, suite.exportMempool()...)

				proposal = append(proposal, []byte("invalid tx"))

				return proposal
			},
			comettypes.ResponseProcessProposal_REJECT,
		},
		{
			"multiple auction txs with filled mempool",
			func() [][]byte {
				numNormalTxs = 100
				numAuctionTxs = 10
				numBundledTxs = 0
				insertRefTxs = false

				suite.createFilledMempool(numNormalTxs, numAuctionTxs, numBundledTxs, insertRefTxs)

				_, auctionTxBzs := suite.getAllAuctionTxs()

				auctionInfo := suite.createAuctionInfoFromTxBzs(auctionTxBzs, 1)

				proposal := append([][]byte{
					auctionInfo,
					auctionTxBzs[0],
				}, suite.exportMempool()...)

				return proposal
			},
			comettypes.ResponseProcessProposal_ACCEPT,
		},
		{
			"multiple auction txs with ref txs + filled mempool",
			func() [][]byte {
				numNormalTxs = 100
				numAuctionTxs = 10
				numBundledTxs = 10
				insertRefTxs = false

				suite.createFilledMempool(numNormalTxs, numAuctionTxs, numBundledTxs, insertRefTxs)

				auctionTxs, auctionTxBzs := suite.getAllAuctionTxs()

				auctionInfo := suite.createAuctionInfoFromTxBzs(auctionTxBzs, 11)

				bidInfo, err := suite.mempool.GetAuctionBidInfo(auctionTxs[0])
				suite.Require().NoError(err)

				proposal := append([][]byte{
					auctionInfo,
					auctionTxBzs[0],
				}, bidInfo.Transactions...)

				proposal = append(proposal, suite.exportMempool()...)

				return proposal
			},
			comettypes.ResponseProcessProposal_ACCEPT,
		},
		{
			"auction tx with front-running",
			func() [][]byte {
				numNormalTxs = 100
				numAuctionTxs = 0
				numBundledTxs = 0
				insertRefTxs = false

				suite.createFilledMempool(numNormalTxs, numAuctionTxs, numBundledTxs, insertRefTxs)

				topAuctionTx, err := testutils.CreateAuctionTxWithSigners(
					suite.encodingConfig.TxConfig,
					suite.accounts[0],
					sdk.NewCoin("foo", sdk.NewInt(1000000)),
					0,
					1,
					[]testutils.Account{suite.accounts[0], suite.accounts[1]}, // front-running
				)
				suite.Require().NoError(err)

				txBz, err := suite.encodingConfig.TxConfig.TxEncoder()(topAuctionTx)
				suite.Require().NoError(err)

				bidInfo, err := suite.mempool.GetAuctionBidInfo(topAuctionTx)
				suite.Require().NoError(err)

				auctionInfo := suite.createAuctionInfoFromTxBzs([][]byte{txBz}, 3)

				proposal := append([][]byte{
					auctionInfo,
					txBz,
				}, bidInfo.Transactions...)

				proposal = append(proposal, suite.exportMempool()...)

				return proposal
			},
			comettypes.ResponseProcessProposal_REJECT,
		},
	}

	for _, tc := range cases {
		suite.Run(tc.name, func() {
			// create a new auction
			params := buildertypes.Params{
				MaxBundleSize:          maxBundleSize,
				ReserveFee:             reserveFee,
				FrontRunningProtection: true,
				MinBidIncrement:        suite.minBidIncrement,
			}
			suite.builderKeeper.SetParams(suite.ctx, params)
			suite.builderDecorator = ante.NewBuilderDecorator(suite.builderKeeper, suite.encodingConfig.TxConfig.TxEncoder(), suite.mempool)

			// reset the proposal handler with the new mempool
			suite.proposalHandler = v2.NewProposalHandler(suite.mempool, suite.logger, suite.anteHandler, suite.encodingConfig.TxConfig.TxEncoder(), suite.encodingConfig.TxConfig.TxDecoder())

			handler := suite.proposalHandler.ProcessProposalHandler()
			res := handler(suite.ctx, comettypes.RequestProcessProposal{
				Txs: tc.createTxs(),
			})

			// Check if the response is valid
			suite.Require().Equal(tc.response, res.Status)
		})
	}
}
