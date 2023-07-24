package abci_test

import (
	comettypes "github.com/cometbft/cometbft/abci/types"
	"github.com/cometbft/cometbft/libs/log"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/skip-mev/pob/abci"
	"github.com/skip-mev/pob/blockbuster"
	"github.com/skip-mev/pob/blockbuster/lanes/auction"
	"github.com/skip-mev/pob/blockbuster/lanes/base"
	testutils "github.com/skip-mev/pob/testutils"
	"github.com/skip-mev/pob/x/builder/ante"
	buildertypes "github.com/skip-mev/pob/x/builder/types"
)

// TODO:
// - Add tests that can that trigger a panic for the tob of block lane
func (suite *ABCITestSuite) TestPrepareProposal() {
	var (
		// the modified transactions cannot exceed this size
		maxTxBytes int64 = 1000000000000000000

		// mempool configuration
		normalTxs        []sdk.Tx
		auctionTxs       []sdk.Tx
		winningBidTx     sdk.Tx
		insertBundledTxs = false

		// auction configuration
		maxBundleSize          uint32 = 10
		reserveFee                    = sdk.NewCoin("foo", sdk.NewInt(1000))
		frontRunningProtection        = true
	)

	cases := []struct {
		name                        string
		malleate                    func()
		expectedNumberProposalTxs   int
		expectedMempoolDistribution map[string]int
	}{
		{
			"single valid tob transaction in the mempool",
			func() {
				bidder := suite.accounts[0]
				bid := sdk.NewCoin("foo", sdk.NewInt(1000))
				nonce := suite.nonces[bidder.Address.String()]
				timeout := uint64(100)
				signers := []testutils.Account{bidder}
				bidTx, err := testutils.CreateAuctionTxWithSigners(suite.encodingConfig.TxConfig, bidder, bid, nonce, timeout, signers)
				suite.Require().NoError(err)

				normalTxs = []sdk.Tx{}
				auctionTxs = []sdk.Tx{bidTx}
				winningBidTx = bidTx
				insertBundledTxs = false
			},
			2,
			map[string]int{
				base.LaneName:    0,
				auction.LaneName: 1,
			},
		},
		{
			"single invalid tob transaction in the mempool",
			func() {
				bidder := suite.accounts[0]
				bid := reserveFee.Sub(sdk.NewCoin("foo", sdk.NewInt(1))) // bid is less than the reserve fee
				nonce := suite.nonces[bidder.Address.String()]
				timeout := uint64(100)
				signers := []testutils.Account{bidder}
				bidTx, err := testutils.CreateAuctionTxWithSigners(suite.encodingConfig.TxConfig, bidder, bid, nonce, timeout, signers)
				suite.Require().NoError(err)

				normalTxs = []sdk.Tx{}
				auctionTxs = []sdk.Tx{bidTx}
				winningBidTx = nil
				insertBundledTxs = false
			},
			0,
			map[string]int{
				base.LaneName:    0,
				auction.LaneName: 0,
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

				normalTxs = []sdk.Tx{normalTx}
				auctionTxs = []sdk.Tx{}
				winningBidTx = nil
				insertBundledTxs = false
			},
			1,
			map[string]int{
				base.LaneName:    1,
				auction.LaneName: 0,
			},
		},
		{
			"normal transactions and tob transactions in the mempool",
			func() {
				// Create a valid tob transaction
				bidder := suite.accounts[0]
				bid := sdk.NewCoin("foo", sdk.NewInt(1000))
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

				normalTxs = []sdk.Tx{normalTx}
				auctionTxs = []sdk.Tx{bidTx}
				winningBidTx = bidTx
				insertBundledTxs = false
			},
			3,
			map[string]int{
				base.LaneName:    1,
				auction.LaneName: 1,
			},
		},
		{
			"multiple tob transactions where the first is invalid",
			func() {
				// Create an invalid tob transaction (frontrunning)
				bidder := suite.accounts[0]
				bid := sdk.NewCoin("foo", sdk.NewInt(1000000000))
				nonce := suite.nonces[bidder.Address.String()]
				timeout := uint64(100)
				signers := []testutils.Account{bidder, bidder, suite.accounts[1]}
				bidTx, err := testutils.CreateAuctionTxWithSigners(suite.encodingConfig.TxConfig, bidder, bid, nonce, timeout, signers)
				suite.Require().NoError(err)

				// Create a valid tob transaction
				bidder = suite.accounts[1]
				bid = sdk.NewCoin("foo", sdk.NewInt(1000))
				nonce = suite.nonces[bidder.Address.String()]
				timeout = uint64(100)
				signers = []testutils.Account{bidder}
				bidTx2, err := testutils.CreateAuctionTxWithSigners(suite.encodingConfig.TxConfig, bidder, bid, nonce, timeout, signers)
				suite.Require().NoError(err)

				normalTxs = []sdk.Tx{}
				auctionTxs = []sdk.Tx{bidTx, bidTx2}
				winningBidTx = bidTx2
				insertBundledTxs = false
			},
			2,
			map[string]int{
				base.LaneName:    0,
				auction.LaneName: 1,
			},
		},
		{
			"multiple tob transactions where the first is valid",
			func() {
				// Create an valid tob transaction
				bidder := suite.accounts[0]
				bid := sdk.NewCoin("foo", sdk.NewInt(10000000))
				nonce := suite.nonces[bidder.Address.String()]
				timeout := uint64(100)
				signers := []testutils.Account{suite.accounts[2], bidder}
				bidTx, err := testutils.CreateAuctionTxWithSigners(suite.encodingConfig.TxConfig, bidder, bid, nonce, timeout, signers)
				suite.Require().NoError(err)

				// Create a valid tob transaction
				bidder = suite.accounts[1]
				bid = sdk.NewCoin("foo", sdk.NewInt(1000))
				nonce = suite.nonces[bidder.Address.String()]
				timeout = uint64(100)
				signers = []testutils.Account{bidder}
				bidTx2, err := testutils.CreateAuctionTxWithSigners(suite.encodingConfig.TxConfig, bidder, bid, nonce, timeout, signers)
				suite.Require().NoError(err)

				normalTxs = []sdk.Tx{}
				auctionTxs = []sdk.Tx{bidTx, bidTx2}
				winningBidTx = bidTx
				insertBundledTxs = false
			},
			3,
			map[string]int{
				base.LaneName:    0,
				auction.LaneName: 2,
			},
		},
		{
			"multiple tob transactions where the first is valid and bundle is inserted into mempool",
			func() {
				frontRunningProtection = false

				// Create an valid tob transaction
				bidder := suite.accounts[0]
				bid := sdk.NewCoin("foo", sdk.NewInt(10000000))
				nonce := suite.nonces[bidder.Address.String()]
				timeout := uint64(100)
				signers := []testutils.Account{suite.accounts[2], suite.accounts[1], bidder, suite.accounts[3], suite.accounts[4]}
				bidTx, err := testutils.CreateAuctionTxWithSigners(suite.encodingConfig.TxConfig, bidder, bid, nonce, timeout, signers)
				suite.Require().NoError(err)

				normalTxs = []sdk.Tx{}
				auctionTxs = []sdk.Tx{bidTx}
				winningBidTx = bidTx
				insertBundledTxs = true
			},
			6,
			map[string]int{
				base.LaneName:    5,
				auction.LaneName: 1,
			},
		},
		{
			"single tob transaction with other normal transactions in the mempool",
			func() {
				// Create an valid tob transaction
				bidder := suite.accounts[0]
				bid := sdk.NewCoin("foo", sdk.NewInt(10000000))
				nonce := suite.nonces[bidder.Address.String()]
				timeout := uint64(100)
				signers := []testutils.Account{suite.accounts[2], suite.accounts[1], bidder, suite.accounts[3], suite.accounts[4]}
				bidTx, err := testutils.CreateAuctionTxWithSigners(suite.encodingConfig.TxConfig, bidder, bid, nonce, timeout, signers)
				suite.Require().NoError(err)

				account := suite.accounts[5]
				nonce = suite.nonces[account.Address.String()]
				timeout = uint64(100)
				numberMsgs := uint64(3)
				normalTx, err := testutils.CreateRandomTx(suite.encodingConfig.TxConfig, account, nonce, numberMsgs, timeout)
				suite.Require().NoError(err)

				normalTxs = []sdk.Tx{normalTx}
				auctionTxs = []sdk.Tx{bidTx}
				winningBidTx = bidTx
				insertBundledTxs = true
			},
			7,
			map[string]int{
				base.LaneName:    6,
				auction.LaneName: 1,
			},
		},
	}

	for _, tc := range cases {
		suite.Run(tc.name, func() {
			suite.SetupTest() // reset
			tc.malleate()

			// Insert all of the normal transactions into the default lane
			for _, tx := range normalTxs {
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

			// create a new auction
			params := buildertypes.Params{
				MaxBundleSize:          maxBundleSize,
				ReserveFee:             reserveFee,
				FrontRunningProtection: frontRunningProtection,
			}
			suite.builderKeeper.SetParams(suite.ctx, params)
			suite.builderDecorator = ante.NewBuilderDecorator(suite.builderKeeper, suite.encodingConfig.TxConfig.TxEncoder(), suite.tobLane, suite.mempool)

			suite.proposalHandler = abci.NewProposalHandler(
				[]blockbuster.Lane{suite.baseLane},
				suite.tobLane,
				suite.logger,
				suite.encodingConfig.TxConfig.TxEncoder(),
				suite.encodingConfig.TxConfig.TxDecoder(),
				abci.NoOpValidateVoteExtensionsFn(),
			)
			handler := suite.proposalHandler.PrepareProposalHandler()
			req := suite.createPrepareProposalRequest(maxTxBytes)
			res := handler(suite.ctx, req)

			// -------------------- Check Invariants -------------------- //
			// The first slot in the proposal must be the auction info
			auctionInfo := abci.AuctionInfo{}
			err := auctionInfo.Unmarshal(res.Txs[abci.AuctionInfoIndex])
			suite.Require().NoError(err)

			// Total bytes must be less than or equal to maxTxBytes
			totalBytes := int64(0)
			for _, tx := range res.Txs[abci.NumInjectedTxs:] {
				totalBytes += int64(len(tx))
			}
			suite.Require().LessOrEqual(totalBytes, maxTxBytes)

			// 2. the number of transactions in the response must be equal to the number of expected transactions
			// NOTE: We add 1 to the expected number of transactions because the first transaction in the response
			// is the auction transaction
			suite.Require().Equal(tc.expectedNumberProposalTxs+1, len(res.Txs))

			// 3. if there are auction transactions, the first transaction must be the top bid
			// and the rest of the bundle must be in the response
			if winningBidTx != nil {
				auctionTx, err := suite.encodingConfig.TxConfig.TxDecoder()(res.Txs[1])
				suite.Require().NoError(err)

				bidInfo, err := suite.tobLane.GetAuctionBidInfo(auctionTx)
				suite.Require().NoError(err)

				for index, tx := range bidInfo.Transactions {
					suite.Require().Equal(tx, res.Txs[index+1+abci.NumInjectedTxs])
				}
			} else if len(res.Txs) > 1 {
				tx, err := suite.encodingConfig.TxConfig.TxDecoder()(res.Txs[1])
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
		})
	}
}

// TODO:
// - Add tests that ensure that the top of block lane does not propose more transactions than it is allowed to
func (suite *ABCITestSuite) TestProcessProposal() {
	var (
		// auction configuration
		maxBundleSize          uint32 = 10
		reserveFee                    = sdk.NewCoin("foo", sdk.NewInt(1000))
		frontRunningProtection        = true
		maxTxBytes             int64  = 1000000000000000000

		// mempool configuration
		proposal [][]byte
	)

	params := buildertypes.Params{
		MaxBundleSize:          maxBundleSize,
		ReserveFee:             reserveFee,
		FrontRunningProtection: frontRunningProtection,
	}
	suite.builderKeeper.SetParams(suite.ctx, params)

	cases := []struct {
		name      string
		createTxs func()
		response  comettypes.ResponseProcessProposal_ProposalStatus
	}{
		{
			"no transactions in mempool with no vote extension info",
			func() {
				proposal = nil
			},
			comettypes.ResponseProcessProposal_REJECT,
		},
		{
			"no transactions in mempool with empty vote extension info",
			func() {
				proposal = [][]byte{}
			},
			comettypes.ResponseProcessProposal_REJECT,
		},
		{
			"single normal tx, no vote extension info",
			func() {
				account := suite.accounts[0]
				nonce := suite.nonces[account.Address.String()]
				timeout := uint64(100)
				numberMsgs := uint64(3)
				normalTxBz, err := testutils.CreateRandomTxBz(suite.encodingConfig.TxConfig, account, nonce, numberMsgs, timeout)
				suite.Require().NoError(err)

				proposal = [][]byte{normalTxBz}
			},
			comettypes.ResponseProcessProposal_REJECT,
		},
		{
			"single auction tx, single auction tx, no vote extension info",
			func() {
				// Create a valid tob transaction
				bidder := suite.accounts[0]
				bid := sdk.NewCoin("foo", sdk.NewInt(1000))
				nonce := suite.nonces[bidder.Address.String()]
				timeout := uint64(100)
				signers := []testutils.Account{bidder}
				bidTx, err := testutils.CreateAuctionTxWithSignerBz(suite.encodingConfig.TxConfig, bidder, bid, nonce, timeout, signers)
				suite.Require().NoError(err)

				// Create a valid default transaction
				account := suite.accounts[1]
				nonce = suite.nonces[account.Address.String()] + 1
				numberMsgs := uint64(3)
				normalTx, err := testutils.CreateRandomTxBz(suite.encodingConfig.TxConfig, account, nonce, numberMsgs, timeout)
				suite.Require().NoError(err)

				proposal = [][]byte{bidTx, normalTx}
			},
			comettypes.ResponseProcessProposal_REJECT,
		},
		{
			"single auction tx with ref txs (no unwrapping)",
			func() {
				// Create a valid tob transaction
				bidder := suite.accounts[0]
				bid := sdk.NewCoin("foo", sdk.NewInt(1000))
				nonce := suite.nonces[bidder.Address.String()]
				timeout := uint64(100)
				signers := []testutils.Account{bidder}
				bidTx, err := testutils.CreateAuctionTxWithSignerBz(suite.encodingConfig.TxConfig, bidder, bid, nonce, timeout, signers)
				suite.Require().NoError(err)

				// Create a valid default transaction
				account := suite.accounts[1]
				nonce = suite.nonces[account.Address.String()] + 1
				numberMsgs := uint64(3)
				normalTx, err := testutils.CreateRandomTxBz(suite.encodingConfig.TxConfig, account, nonce, numberMsgs, timeout)
				suite.Require().NoError(err)

				auctionInfo := suite.createAuctionInfoFromTxBzs([][]byte{bidTx}, 2, maxTxBytes)

				proposal = [][]byte{
					auctionInfo,
					bidTx,
					normalTx,
				}
			},
			comettypes.ResponseProcessProposal_REJECT,
		},
		{
			"single auction tx with ref txs (with unwrapping)",
			func() {
				// Create a valid tob transaction
				bidder := suite.accounts[0]
				bid := sdk.NewCoin("foo", sdk.NewInt(1000))
				nonce := suite.nonces[bidder.Address.String()]
				timeout := uint64(100)
				signers := []testutils.Account{bidder}
				bidTxBz, err := testutils.CreateAuctionTxWithSignerBz(suite.encodingConfig.TxConfig, bidder, bid, nonce, timeout, signers)
				suite.Require().NoError(err)

				auctionInfo := suite.createAuctionInfoFromTxBzs([][]byte{bidTxBz}, 2, maxTxBytes)

				bidInfo := suite.getAuctionBidInfoFromTxBz(bidTxBz)

				proposal = append(
					[][]byte{
						auctionInfo,
						bidTxBz,
					},
					bidInfo.Transactions...,
				)
			},
			comettypes.ResponseProcessProposal_ACCEPT,
		},
		{
			"single auction tx with ref txs but misplaced in proposal",
			func() {
				// Create a valid tob transaction
				bidder := suite.accounts[0]
				bid := sdk.NewCoin("foo", sdk.NewInt(1000))
				nonce := suite.nonces[bidder.Address.String()]
				timeout := uint64(100)
				signers := []testutils.Account{suite.accounts[1], bidder}
				bidTxBz, err := testutils.CreateAuctionTxWithSignerBz(suite.encodingConfig.TxConfig, bidder, bid, nonce, timeout, signers)
				suite.Require().NoError(err)

				auctionInfo := suite.createAuctionInfoFromTxBzs([][]byte{bidTxBz}, 3, maxTxBytes)

				bidInfo := suite.getAuctionBidInfoFromTxBz(bidTxBz)

				proposal = [][]byte{
					auctionInfo,
					bidTxBz,
					bidInfo.Transactions[1],
					bidInfo.Transactions[0],
				}
			},
			comettypes.ResponseProcessProposal_REJECT,
		},
		{
			"single auction tx, but auction tx is not valid",
			func() {
				// Create a valid tob transaction
				bidder := suite.accounts[0]
				bid := sdk.NewCoin("foo", sdk.NewInt(1000))
				nonce := suite.nonces[bidder.Address.String()]
				timeout := uint64(100)
				signers := []testutils.Account{bidder, suite.accounts[1]} // front-running
				bidTxBz, err := testutils.CreateAuctionTxWithSignerBz(suite.encodingConfig.TxConfig, bidder, bid, nonce, timeout, signers)
				suite.Require().NoError(err)

				auctionInfo := suite.createAuctionInfoFromTxBzs([][]byte{bidTxBz}, 3, maxTxBytes)

				bidInfo := suite.getAuctionBidInfoFromTxBz(bidTxBz)
				proposal = append(
					[][]byte{
						auctionInfo,
						bidTxBz,
					},
					bidInfo.Transactions...,
				)
			},
			comettypes.ResponseProcessProposal_REJECT,
		},
		{
			"multiple auction txs but wrong auction tx is at top of block",
			func() {
				// Create a valid tob transaction
				bidder := suite.accounts[0]
				bid := sdk.NewCoin("foo", sdk.NewInt(1000))
				nonce := suite.nonces[bidder.Address.String()]
				timeout := uint64(100)
				signers := []testutils.Account{bidder, bidder}
				bidTxBz, err := testutils.CreateAuctionTxWithSignerBz(suite.encodingConfig.TxConfig, bidder, bid, nonce, timeout, signers)
				suite.Require().NoError(err)

				// Create another valid tob transaction
				bidder = suite.accounts[1]
				bid = sdk.NewCoin("foo", sdk.NewInt(1000000))
				nonce = suite.nonces[bidder.Address.String()]
				timeout = uint64(100)
				signers = []testutils.Account{bidder}
				bidTxBz2, err := testutils.CreateAuctionTxWithSignerBz(suite.encodingConfig.TxConfig, bidder, bid, nonce, timeout, signers)
				suite.Require().NoError(err)

				auctionInfo := suite.createAuctionInfoFromTxBzs([][]byte{bidTxBz, bidTxBz2}, 3, maxTxBytes)

				bidInfo := suite.getAuctionBidInfoFromTxBz(bidTxBz)

				proposal = append(
					[][]byte{
						auctionInfo,
						bidTxBz,
					},
					bidInfo.Transactions...,
				)
			},
			comettypes.ResponseProcessProposal_REJECT,
		},
		{
			"multiple auction txs and correct auction tx is selected",
			func() {
				// Create a valid tob transaction
				bidder := suite.accounts[0]
				bid := sdk.NewCoin("foo", sdk.NewInt(1000))
				nonce := suite.nonces[bidder.Address.String()]
				timeout := uint64(100)
				signers := []testutils.Account{bidder, bidder}
				bidTxBz, err := testutils.CreateAuctionTxWithSignerBz(suite.encodingConfig.TxConfig, bidder, bid, nonce, timeout, signers)
				suite.Require().NoError(err)

				// Create another valid tob transaction
				bidder = suite.accounts[1]
				bid = sdk.NewCoin("foo", sdk.NewInt(1000000))
				nonce = suite.nonces[bidder.Address.String()]
				timeout = uint64(100)
				signers = []testutils.Account{bidder}
				bidTxBz2, err := testutils.CreateAuctionTxWithSignerBz(suite.encodingConfig.TxConfig, bidder, bid, nonce, timeout, signers)
				suite.Require().NoError(err)

				auctionInfo := suite.createAuctionInfoFromTxBzs([][]byte{bidTxBz, bidTxBz2}, 2, maxTxBytes)

				bidInfo := suite.getAuctionBidInfoFromTxBz(bidTxBz2)

				proposal = append(
					[][]byte{
						auctionInfo,
						bidTxBz2,
					},
					bidInfo.Transactions...,
				)
			},
			comettypes.ResponseProcessProposal_ACCEPT,
		},
		{
			"multiple auction txs included in block",
			func() {
				// Create a valid tob transaction
				bidder := suite.accounts[0]
				bid := sdk.NewCoin("foo", sdk.NewInt(1000))
				nonce := suite.nonces[bidder.Address.String()]
				timeout := uint64(100)
				signers := []testutils.Account{bidder, bidder}
				bidTxBz, err := testutils.CreateAuctionTxWithSignerBz(suite.encodingConfig.TxConfig, bidder, bid, nonce, timeout, signers)
				suite.Require().NoError(err)

				// Create another valid tob transaction
				bidder = suite.accounts[1]
				bid = sdk.NewCoin("foo", sdk.NewInt(1000000))
				nonce = suite.nonces[bidder.Address.String()]
				timeout = uint64(100)
				signers = []testutils.Account{bidder}
				bidTxBz2, err := testutils.CreateAuctionTxWithSignerBz(suite.encodingConfig.TxConfig, bidder, bid, nonce, timeout, signers)
				suite.Require().NoError(err)

				auctionInfo := suite.createAuctionInfoFromTxBzs([][]byte{bidTxBz, bidTxBz2}, 2, maxTxBytes)

				bidInfo := suite.getAuctionBidInfoFromTxBz(bidTxBz2)
				bidInfo2 := suite.getAuctionBidInfoFromTxBz(bidTxBz)

				proposal = append(
					[][]byte{
						auctionInfo,
						bidTxBz2,
					},
					bidInfo.Transactions...,
				)

				proposal = append(proposal, bidTxBz)
				proposal = append(proposal, bidInfo2.Transactions...)
			},
			comettypes.ResponseProcessProposal_REJECT,
		},
		{
			"single auction tx, but rest of the mempool is invalid",
			func() {
				// Create a valid tob transaction
				bidder := suite.accounts[0]
				bid := sdk.NewCoin("foo", sdk.NewInt(1000))
				nonce := suite.nonces[bidder.Address.String()]
				timeout := uint64(100)
				signers := []testutils.Account{bidder}
				bidTxBz, err := testutils.CreateAuctionTxWithSignerBz(suite.encodingConfig.TxConfig, bidder, bid, nonce, timeout, signers)
				suite.Require().NoError(err)

				auctionInfo := suite.createAuctionInfoFromTxBzs([][]byte{bidTxBz}, 2, maxTxBytes)

				bidInfo := suite.getAuctionBidInfoFromTxBz(bidTxBz)

				proposal = append(
					[][]byte{
						auctionInfo,
						bidTxBz,
					},
					bidInfo.Transactions...,
				)

				proposal = append(proposal, []byte("invalid tx"))
			},
			comettypes.ResponseProcessProposal_REJECT,
		},
		{
			"multiple auction txs with ref txs + normal transactions",
			func() {
				// Create a valid tob transaction
				bidder := suite.accounts[0]
				bid := sdk.NewCoin("foo", sdk.NewInt(1000))
				nonce := suite.nonces[bidder.Address.String()]
				timeout := uint64(100)
				signers := []testutils.Account{bidder}
				bidTxBz, err := testutils.CreateAuctionTxWithSignerBz(suite.encodingConfig.TxConfig, bidder, bid, nonce, timeout, signers)
				suite.Require().NoError(err)

				auctionInfo := suite.createAuctionInfoFromTxBzs([][]byte{bidTxBz}, 2, maxTxBytes)

				bidInfo := suite.getAuctionBidInfoFromTxBz(bidTxBz)

				proposal = append(
					[][]byte{
						auctionInfo,
						bidTxBz,
					},
					bidInfo.Transactions...,
				)

				normalTxBz, err := testutils.CreateRandomTxBz(suite.encodingConfig.TxConfig, suite.accounts[1], nonce, 3, timeout)
				suite.Require().NoError(err)
				proposal = append(proposal, normalTxBz)

				normalTxBz, err = testutils.CreateRandomTxBz(suite.encodingConfig.TxConfig, suite.accounts[2], nonce, 3, timeout)
				suite.Require().NoError(err)
				proposal = append(proposal, normalTxBz)
			},
			comettypes.ResponseProcessProposal_ACCEPT,
		},
		{
			"front-running protection disabled",
			func() {
				// Create a valid tob transaction
				bidder := suite.accounts[0]
				bid := sdk.NewCoin("foo", sdk.NewInt(10000000))
				nonce := suite.nonces[bidder.Address.String()]
				timeout := uint64(100)
				signers := []testutils.Account{suite.accounts[2], suite.accounts[1], bidder, suite.accounts[3], suite.accounts[4]}
				bidTxBz, err := testutils.CreateAuctionTxWithSignerBz(suite.encodingConfig.TxConfig, bidder, bid, nonce, timeout, signers)
				suite.Require().NoError(err)

				auctionInfo := suite.createAuctionInfoFromTxBzs([][]byte{bidTxBz}, uint64(len(signers)+1), maxTxBytes)

				bidInfo := suite.getAuctionBidInfoFromTxBz(bidTxBz)

				proposal = append(
					[][]byte{
						auctionInfo,
						bidTxBz,
					},
					bidInfo.Transactions...,
				)

				normalTxBz, err := testutils.CreateRandomTxBz(suite.encodingConfig.TxConfig, suite.accounts[5], nonce, 3, timeout)
				suite.Require().NoError(err)
				proposal = append(proposal, normalTxBz)

				normalTxBz, err = testutils.CreateRandomTxBz(suite.encodingConfig.TxConfig, suite.accounts[6], nonce, 3, timeout)
				suite.Require().NoError(err)
				proposal = append(proposal, normalTxBz)

				// disable frontrunning protection
				params := buildertypes.Params{
					MaxBundleSize:          maxBundleSize,
					ReserveFee:             reserveFee,
					FrontRunningProtection: false,
				}
				suite.builderKeeper.SetParams(suite.ctx, params)
			},
			comettypes.ResponseProcessProposal_ACCEPT,
		},
	}

	for _, tc := range cases {
		suite.Run(tc.name, func() {
			// suite.SetupTest() // reset
			suite.builderDecorator = ante.NewBuilderDecorator(suite.builderKeeper, suite.encodingConfig.TxConfig.TxEncoder(), suite.tobLane, suite.mempool)

			// reset the proposal handler with the new mempool
			suite.proposalHandler = abci.NewProposalHandler(
				[]blockbuster.Lane{suite.baseLane},
				suite.tobLane, log.NewNopLogger(),
				suite.encodingConfig.TxConfig.TxEncoder(),
				suite.encodingConfig.TxConfig.TxDecoder(),
				abci.NoOpValidateVoteExtensionsFn(),
			)

			tc.createTxs()

			handler := suite.proposalHandler.ProcessProposalHandler()
			res := handler(suite.ctx, comettypes.RequestProcessProposal{
				Txs: proposal,
			})

			// Check if the response is valid
			suite.Require().Equal(tc.response, res.Status)
		})
	}
}
