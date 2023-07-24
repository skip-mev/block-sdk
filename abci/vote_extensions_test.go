package abci_test

import (
	"cosmossdk.io/log"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/skip-mev/pob/abci"
	testutils "github.com/skip-mev/pob/testutils"
	"github.com/skip-mev/pob/x/builder/types"
)

func (suite *ABCITestSuite) TestExtendVoteExtensionHandler() {
	params := types.Params{
		MaxBundleSize:          5,
		ReserveFee:             sdk.NewCoin("foo", sdk.NewInt(10)),
		FrontRunningProtection: true,
	}

	testCases := []struct {
		name          string
		getExpectedVE func() []byte
	}{
		{
			"empty mempool",
			func() []byte {
				return []byte{}
			},
		},
		{
			"filled mempool with no auction transactions",
			func() []byte {
				suite.fillBaseLane(10)
				return []byte{}
			},
		},
		{
			"mempool with invalid auction transaction (too many bundled transactions)",
			func() []byte {
				suite.fillTOBLane(3, int(params.MaxBundleSize)+1)
				return []byte{}
			},
		},
		{
			"mempool with invalid auction transaction (invalid bid)",
			func() []byte {
				bidder := suite.accounts[0]
				bid := params.ReserveFee.Sub(sdk.NewCoin("foo", sdk.NewInt(1)))
				signers := []testutils.Account{bidder}
				timeout := 1

				bidTx, err := testutils.CreateAuctionTxWithSigners(suite.encodingConfig.TxConfig, bidder, bid, 0, uint64(timeout), signers)
				suite.Require().NoError(err)

				suite.Require().NoError(suite.mempool.Insert(suite.ctx, bidTx))

				// this should return nothing since the top bid is not valid
				return []byte{}
			},
		},
		{
			"mempool contains only invalid auction bids (bid is too low)",
			func() []byte {
				params.ReserveFee = sdk.NewCoin("foo", sdk.NewInt(10000000000000000))
				err := suite.builderKeeper.SetParams(suite.ctx, params)
				suite.Require().NoError(err)

				// this way all of the bids will be too small
				suite.fillTOBLane(4, 1)

				return []byte{}
			},
		},
		{
			"mempool contains bid that has an invalid timeout",
			func() []byte {
				bidder := suite.accounts[0]
				bid := params.ReserveFee
				signers := []testutils.Account{bidder}
				timeout := 0

				bidTx, err := testutils.CreateAuctionTxWithSigners(suite.encodingConfig.TxConfig, bidder, bid, 0, uint64(timeout), signers)
				suite.Require().NoError(err)
				suite.Require().NoError(suite.tobLane.Insert(suite.ctx, bidTx))

				// this should return nothing since the top bid is not valid
				return []byte{}
			},
		},
		{
			"top bid is invalid but next best is valid",
			func() []byte {
				params.ReserveFee = sdk.NewCoin("foo", sdk.NewInt(10))

				bidder := suite.accounts[0]
				bid := params.ReserveFee.Add(params.ReserveFee)
				signers := []testutils.Account{bidder}
				timeout := 0

				bidTx, err := testutils.CreateAuctionTxWithSigners(suite.encodingConfig.TxConfig, bidder, bid, 0, uint64(timeout), signers)
				suite.Require().NoError(err)
				suite.Require().NoError(suite.mempool.Insert(suite.ctx, bidTx))

				bidder = suite.accounts[1]
				bid = params.ReserveFee
				signers = []testutils.Account{bidder}
				timeout = 100
				bidTx2, err := testutils.CreateAuctionTxWithSigners(suite.encodingConfig.TxConfig, bidder, bid, 0, uint64(timeout), signers)
				suite.Require().NoError(err)
				suite.Require().NoError(suite.mempool.Insert(suite.ctx, bidTx2))

				bz, err := suite.encodingConfig.TxConfig.TxEncoder()(bidTx2)
				suite.Require().NoError(err)

				return bz
			},
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest() // reset
			expectedVE := tc.getExpectedVE()

			err := suite.builderKeeper.SetParams(suite.ctx, params)
			suite.Require().NoError(err)

			// Reset the handler with the new mempool
			suite.voteExtensionHandler = abci.NewVoteExtensionHandler(
				log.NewTestLogger(suite.T()),
				suite.tobLane,
				suite.encodingConfig.TxConfig.TxDecoder(),
				suite.encodingConfig.TxConfig.TxEncoder(),
			)

			handler := suite.voteExtensionHandler.ExtendVoteHandler()
			resp, err := handler(suite.ctx, nil)

			suite.Require().NoError(err)
			suite.Require().Equal(expectedVE, resp.VoteExtension)
		})
	}
}

func (suite *ABCITestSuite) TestVerifyVoteExtensionHandler() {
	params := types.Params{
		MaxBundleSize:          5,
		ReserveFee:             sdk.NewCoin("foo", sdk.NewInt(100)),
		FrontRunningProtection: true,
	}

	err := suite.builderKeeper.SetParams(suite.ctx, params)
	suite.Require().NoError(err)

	testCases := []struct {
		name        string
		req         func() *abci.RequestVerifyVoteExtension
		expectedErr bool
	}{
		{
			"invalid vote extension bytes",
			func() *abci.RequestVerifyVoteExtension {
				return &abci.RequestVerifyVoteExtension{
					VoteExtension: []byte("invalid vote extension"),
				}
			},
			true,
		},
		{
			"empty vote extension bytes",
			func() *abci.RequestVerifyVoteExtension {
				return &abci.RequestVerifyVoteExtension{
					VoteExtension: []byte{},
				}
			},
			false,
		},
		{
			"nil vote extension bytes",
			func() *abci.RequestVerifyVoteExtension {
				return &abci.RequestVerifyVoteExtension{
					VoteExtension: nil,
				}
			},
			false,
		},
		{
			"invalid extension with bid tx with bad timeout",
			func() *abci.RequestVerifyVoteExtension {
				bidder := suite.accounts[0]
				bid := sdk.NewCoin("foo", sdk.NewInt(10))
				signers := []testutils.Account{bidder}
				timeout := 0

				bz := suite.createAuctionTxBz(bidder, bid, signers, timeout)
				return &abci.RequestVerifyVoteExtension{
					VoteExtension: bz,
				}
			},
			true,
		},
		{
			"invalid vote extension with bid tx with bad bid",
			func() *abci.RequestVerifyVoteExtension {
				bidder := suite.accounts[0]
				bid := sdk.NewCoin("foo", sdk.NewInt(0))
				signers := []testutils.Account{bidder}
				timeout := 10

				bz := suite.createAuctionTxBz(bidder, bid, signers, timeout)
				return &abci.RequestVerifyVoteExtension{
					VoteExtension: bz,
				}
			},
			true,
		},
		{
			"valid vote extension",
			func() *abci.RequestVerifyVoteExtension {
				bidder := suite.accounts[0]
				bid := params.ReserveFee
				signers := []testutils.Account{bidder}
				timeout := 10

				bz := suite.createAuctionTxBz(bidder, bid, signers, timeout)
				return &abci.RequestVerifyVoteExtension{
					VoteExtension: bz,
				}
			},
			false,
		},
		{
			"invalid vote extension with front running bid tx",
			func() *abci.RequestVerifyVoteExtension {
				bidder := suite.accounts[0]
				bid := params.ReserveFee
				timeout := 10

				bundlee := testutils.RandomAccounts(suite.random, 1)[0]
				signers := []testutils.Account{bidder, bundlee}

				bz := suite.createAuctionTxBz(bidder, bid, signers, timeout)
				return &abci.RequestVerifyVoteExtension{
					VoteExtension: bz,
				}
			},
			true,
		},
		{
			"invalid vote extension with too many bundle txs",
			func() *abci.RequestVerifyVoteExtension {
				// disable front running protection
				params.FrontRunningProtection = false
				err := suite.builderKeeper.SetParams(suite.ctx, params)
				suite.Require().NoError(err)

				bidder := suite.accounts[0]
				bid := params.ReserveFee
				signers := testutils.RandomAccounts(suite.random, int(params.MaxBundleSize)+1)
				timeout := 10

				bz := suite.createAuctionTxBz(bidder, bid, signers, timeout)
				return &abci.RequestVerifyVoteExtension{
					VoteExtension: bz,
				}
			},
			true,
		},
		{
			"invalid vote extension with a failing bundle tx",
			func() *abci.RequestVerifyVoteExtension {
				bidder := suite.accounts[0]
				bid := params.ReserveFee

				msgAuctionBid, err := testutils.CreateMsgAuctionBid(suite.encodingConfig.TxConfig, bidder, bid, 0, 0)
				suite.Require().NoError(err)

				// Create a failing tx
				msgAuctionBid.Transactions = [][]byte{{0x01}}

				bidTx, err := testutils.CreateTx(suite.encodingConfig.TxConfig, suite.accounts[0], 0, 1, []sdk.Msg{msgAuctionBid})
				suite.Require().NoError(err)

				bz, err := suite.encodingConfig.TxConfig.TxEncoder()(bidTx)
				suite.Require().NoError(err)

				return &abci.RequestVerifyVoteExtension{
					VoteExtension: bz,
				}
			},
			true,
		},
		{
			"valid vote extension + no comparison to local mempool",
			func() *abci.RequestVerifyVoteExtension {
				bidder := suite.accounts[0]
				bid := params.ReserveFee
				signers := []testutils.Account{bidder}
				timeout := 10

				bz := suite.createAuctionTxBz(bidder, bid, signers, timeout)

				// Add a bid to the mempool that is greater than the one in the vote extension
				bid = bid.Add(params.ReserveFee)
				bidTx, err := testutils.CreateAuctionTxWithSigners(suite.encodingConfig.TxConfig, bidder, bid, 10, 1, signers)
				suite.Require().NoError(err)

				err = suite.mempool.Insert(suite.ctx, bidTx)
				suite.Require().NoError(err)

				tx := suite.tobLane.GetTopAuctionTx(suite.ctx)
				suite.Require().NotNil(tx)

				return &abci.RequestVerifyVoteExtension{
					VoteExtension: bz,
				}
			},
			false,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			req := tc.req()

			handler := suite.voteExtensionHandler.VerifyVoteExtensionHandler()
			_, err := handler(suite.ctx, req)

			if tc.expectedErr {
				suite.Require().Error(err)
			} else {
				suite.Require().NoError(err)
			}
		})
	}
}

func (suite *ABCITestSuite) createAuctionTxBz(bidder testutils.Account, bid sdk.Coin, signers []testutils.Account, timeout int) []byte {
	auctionTx, err := testutils.CreateAuctionTxWithSigners(suite.encodingConfig.TxConfig, bidder, bid, 0, uint64(timeout), signers)
	suite.Require().NoError(err)

	txBz, err := suite.encodingConfig.TxConfig.TxEncoder()(auctionTx)
	suite.Require().NoError(err)

	return txBz
}
