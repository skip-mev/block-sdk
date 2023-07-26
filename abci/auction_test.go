package abci_test

import (
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	testutils "github.com/skip-mev/pob/testutils"
	buildertypes "github.com/skip-mev/pob/x/builder/types"
)

func (suite *ABCITestSuite) TestGetBidsFromVoteExtensions() {
	testCases := []struct {
		name                 string
		createVoteExtensions func() ([][]byte, [][]byte) // returns (vote extensions, expected bids)
	}{
		{
			"no vote extensions",
			func() ([][]byte, [][]byte) {
				return nil, [][]byte{}
			},
		},
		{
			"no vote extensions",
			func() ([][]byte, [][]byte) {
				return [][]byte{}, [][]byte{}
			},
		},
		{
			"single vote extension",
			func() ([][]byte, [][]byte) {
				bidTxBz, err := testutils.CreateAuctionTxWithSignerBz(
					suite.encodingConfig.TxConfig,
					suite.accounts[0],
					sdk.NewCoin("stake", math.NewInt(100)),
					0,
					1,
					[]testutils.Account{suite.accounts[0]},
				)
				suite.Require().NoError(err)

				voteExtensions := [][]byte{
					bidTxBz,
				}

				expectedBids := [][]byte{
					bidTxBz,
				}

				return voteExtensions, expectedBids
			},
		},
		{
			"multiple vote extensions",
			func() ([][]byte, [][]byte) {
				bidTxBz1, err := testutils.CreateAuctionTxWithSignerBz(
					suite.encodingConfig.TxConfig,
					suite.accounts[0],
					sdk.NewCoin("stake", math.NewInt(101)),
					0,
					1,
					[]testutils.Account{suite.accounts[0]},
				)
				suite.Require().NoError(err)

				bidTxBz2, err := testutils.CreateAuctionTxWithSignerBz(
					suite.encodingConfig.TxConfig,
					suite.accounts[0],
					sdk.NewCoin("stake", math.NewInt(100)),
					0,
					1,
					[]testutils.Account{suite.accounts[0]},
				)
				suite.Require().NoError(err)

				voteExtensions := [][]byte{
					bidTxBz1,
					bidTxBz2,
				}

				expectedBids := [][]byte{
					bidTxBz1,
					bidTxBz2,
				}

				return voteExtensions, expectedBids
			},
		},
		{
			"multiple vote extensions with some noise",
			func() ([][]byte, [][]byte) {
				bidTxBz1, err := testutils.CreateAuctionTxWithSignerBz(
					suite.encodingConfig.TxConfig,
					suite.accounts[0],
					sdk.NewCoin("stake", math.NewInt(101)),
					0,
					1,
					[]testutils.Account{suite.accounts[0]},
				)
				suite.Require().NoError(err)

				bidTxBz2, err := testutils.CreateAuctionTxWithSignerBz(
					suite.encodingConfig.TxConfig,
					suite.accounts[0],
					sdk.NewCoin("stake", math.NewInt(100)),
					0,
					1,
					[]testutils.Account{suite.accounts[0]},
				)
				suite.Require().NoError(err)

				voteExtensions := [][]byte{
					bidTxBz1,
					nil,
					bidTxBz2,
					[]byte("noise"),
					[]byte("noise p2"),
				}

				expectedBids := [][]byte{
					bidTxBz1,
					bidTxBz2,
				}

				return voteExtensions, expectedBids
			},
		},
		{
			"multiple vote extensions with some normal txs",
			func() ([][]byte, [][]byte) {
				bidTxBz1, err := testutils.CreateAuctionTxWithSignerBz(
					suite.encodingConfig.TxConfig,
					suite.accounts[0],
					sdk.NewCoin("stake", math.NewInt(101)),
					0,
					1,
					[]testutils.Account{suite.accounts[0]},
				)
				suite.Require().NoError(err)

				bidTxBz2, err := testutils.CreateAuctionTxWithSignerBz(
					suite.encodingConfig.TxConfig,
					suite.accounts[0],
					sdk.NewCoin("stake", math.NewInt(100)),
					0,
					1,
					[]testutils.Account{suite.accounts[0]},
				)
				suite.Require().NoError(err)

				randomBz, err := testutils.CreateRandomTxBz(
					suite.encodingConfig.TxConfig,
					suite.accounts[0],
					0,
					1,
					0,
				)
				suite.Require().NoError(err)

				voteExtensions := [][]byte{
					bidTxBz1,
					bidTxBz2,
					nil,
					randomBz,
					[]byte("noise p2"),
				}

				expectedBids := [][]byte{
					bidTxBz1,
					bidTxBz2,
				}

				return voteExtensions, expectedBids
			},
		},
		{
			"multiple vote extensions with some normal txs in unsorted order",
			func() ([][]byte, [][]byte) {
				bidTxBz1, err := testutils.CreateAuctionTxWithSignerBz(
					suite.encodingConfig.TxConfig,
					suite.accounts[0],
					sdk.NewCoin("stake", math.NewInt(1001)),
					0,
					1,
					[]testutils.Account{suite.accounts[0]},
				)
				suite.Require().NoError(err)

				bidTxBz2, err := testutils.CreateAuctionTxWithSignerBz(
					suite.encodingConfig.TxConfig,
					suite.accounts[0],
					sdk.NewCoin("stake", math.NewInt(100)),
					0,
					1,
					[]testutils.Account{suite.accounts[0]},
				)
				suite.Require().NoError(err)

				randomBz, err := testutils.CreateRandomTxBz(
					suite.encodingConfig.TxConfig,
					suite.accounts[0],
					0,
					1,
					0,
				)
				suite.Require().NoError(err)

				voteExtensions := [][]byte{
					bidTxBz2,
					bidTxBz1,
					nil,
					randomBz,
					[]byte("noise p2"),
				}

				expectedBids := [][]byte{
					bidTxBz1,
					bidTxBz2,
				}

				return voteExtensions, expectedBids
			},
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			voteExtensions, expectedBids := tc.createVoteExtensions()

			commitInfo := suite.createExtendedVoteInfo(voteExtensions)

			// get the bids from the vote extensions
			bids := suite.proposalHandler.GetBidsFromVoteExtensions(commitInfo)

			// Check invarients
			suite.Require().Equal(len(expectedBids), len(bids))
			for i, bid := range expectedBids {
				actualBz, err := suite.encodingConfig.TxConfig.TxEncoder()(bids[i])
				suite.Require().NoError(err)

				suite.Require().Equal(bid, actualBz)
			}
		})
	}
}

func (suite *ABCITestSuite) TestBuildTOB() {
	params := buildertypes.Params{
		MaxBundleSize:          4,
		ReserveFee:             sdk.NewCoin("stake", math.NewInt(100)),
		MinBidIncrement:        sdk.NewCoin("stake", math.NewInt(100)),
		FrontRunningProtection: true,
	}
	suite.builderKeeper.SetParams(suite.ctx, params)

	testCases := []struct {
		name      string
		getBidTxs func() ([]sdk.Tx, sdk.Tx) // returns the bids and the winning bid
		maxBytes  int64
	}{
		{
			"no bids",
			func() ([]sdk.Tx, sdk.Tx) {
				return []sdk.Tx{}, nil
			},
			1000000000,
		},
		{
			"single bid",
			func() ([]sdk.Tx, sdk.Tx) {
				bidTx, err := testutils.CreateAuctionTxWithSigners(
					suite.encodingConfig.TxConfig,
					suite.accounts[0],
					sdk.NewCoin("stake", math.NewInt(101)),
					0,
					uint64(suite.ctx.BlockHeight())+2,
					[]testutils.Account{suite.accounts[0]},
				)
				suite.Require().NoError(err)

				return []sdk.Tx{bidTx}, bidTx
			},
			1000000000,
		},
		{
			"single invalid bid (bid is too small)",
			func() ([]sdk.Tx, sdk.Tx) {
				bidTx, err := testutils.CreateAuctionTxWithSigners(
					suite.encodingConfig.TxConfig,
					suite.accounts[0],
					sdk.NewCoin("stake", math.NewInt(1)),
					0,
					uint64(suite.ctx.BlockHeight())+2,
					[]testutils.Account{suite.accounts[0]},
				)
				suite.Require().NoError(err)

				return []sdk.Tx{bidTx}, nil
			},
			1000000000,
		},
		{
			"single invalid bid with front-running",
			func() ([]sdk.Tx, sdk.Tx) {
				bidTx, err := testutils.CreateAuctionTxWithSigners(
					suite.encodingConfig.TxConfig,
					suite.accounts[0],
					sdk.NewCoin("stake", math.NewInt(1000)),
					0,
					uint64(suite.ctx.BlockHeight())+2,
					[]testutils.Account{suite.accounts[0], suite.accounts[1]},
				)
				suite.Require().NoError(err)

				return []sdk.Tx{bidTx}, nil
			},
			1000000000,
		},
		{
			"single invalid bid with too many transactions in the bundle",
			func() ([]sdk.Tx, sdk.Tx) {
				bidTx, err := testutils.CreateAuctionTxWithSigners(
					suite.encodingConfig.TxConfig,
					suite.accounts[0],
					sdk.NewCoin("stake", math.NewInt(101)),
					0,
					uint64(suite.ctx.BlockHeight())+2,
					[]testutils.Account{
						suite.accounts[0],
						suite.accounts[0],
						suite.accounts[0],
						suite.accounts[0],
						suite.accounts[0],
						suite.accounts[0],
					},
				)
				suite.Require().NoError(err)

				return []sdk.Tx{bidTx}, nil
			},
			1000000000,
		},
		{
			"single bid but max bytes is too small",
			func() ([]sdk.Tx, sdk.Tx) {
				bidTx, err := testutils.CreateAuctionTxWithSigners(
					suite.encodingConfig.TxConfig,
					suite.accounts[0],
					sdk.NewCoin("stake", math.NewInt(101)),
					0,
					uint64(suite.ctx.BlockHeight())+2,
					[]testutils.Account{suite.accounts[0]},
				)
				suite.Require().NoError(err)

				return []sdk.Tx{bidTx}, nil
			},
			1,
		},
		{
			"multiple bids",
			func() ([]sdk.Tx, sdk.Tx) {
				bidTx1, err := testutils.CreateAuctionTxWithSigners(
					suite.encodingConfig.TxConfig,
					suite.accounts[0],
					sdk.NewCoin("stake", math.NewInt(101)),
					0,
					uint64(suite.ctx.BlockHeight())+2,
					[]testutils.Account{suite.accounts[0]},
				)
				suite.Require().NoError(err)

				bidTx2, err := testutils.CreateAuctionTxWithSigners(
					suite.encodingConfig.TxConfig,
					suite.accounts[1],
					sdk.NewCoin("stake", math.NewInt(102)),
					0,
					uint64(suite.ctx.BlockHeight())+2,
					[]testutils.Account{suite.accounts[1]},
				)
				suite.Require().NoError(err)

				return []sdk.Tx{bidTx2, bidTx1}, bidTx2
			},
			1000000000,
		},
		{
			"multiple bids with front-running",
			func() ([]sdk.Tx, sdk.Tx) {
				bidTx1, err := testutils.CreateAuctionTxWithSigners(
					suite.encodingConfig.TxConfig,
					suite.accounts[0],
					sdk.NewCoin("stake", math.NewInt(1000)),
					0,
					uint64(suite.ctx.BlockHeight())+2,
					[]testutils.Account{suite.accounts[0], suite.accounts[1]},
				)
				suite.Require().NoError(err)

				bidTx2, err := testutils.CreateAuctionTxWithSigners(
					suite.encodingConfig.TxConfig,
					suite.accounts[1],
					sdk.NewCoin("stake", math.NewInt(200)),
					0,
					uint64(suite.ctx.BlockHeight())+2,
					[]testutils.Account{suite.accounts[1]},
				)
				suite.Require().NoError(err)

				return []sdk.Tx{bidTx1, bidTx2}, bidTx2
			},
			1000000000,
		},
		{
			"multiple bids with too many transactions in the bundle",
			func() ([]sdk.Tx, sdk.Tx) {
				bidTx1, err := testutils.CreateAuctionTxWithSigners(
					suite.encodingConfig.TxConfig,
					suite.accounts[0],
					sdk.NewCoin("stake", math.NewInt(101)),
					0,
					uint64(suite.ctx.BlockHeight())+2,
					[]testutils.Account{
						suite.accounts[0],
						suite.accounts[0],
						suite.accounts[0],
						suite.accounts[0],
						suite.accounts[0],
						suite.accounts[0],
					},
				)
				suite.Require().NoError(err)

				bidTx2, err := testutils.CreateAuctionTxWithSigners(
					suite.encodingConfig.TxConfig,
					suite.accounts[1],
					sdk.NewCoin("stake", math.NewInt(102)),
					0,
					uint64(suite.ctx.BlockHeight())+2,
					[]testutils.Account{suite.accounts[1]},
				)
				suite.Require().NoError(err)

				return []sdk.Tx{bidTx1, bidTx2}, bidTx2
			},
			1000000000,
		},
		{
			"multiple bids unsorted",
			func() ([]sdk.Tx, sdk.Tx) {
				bidTx1, err := testutils.CreateAuctionTxWithSigners(
					suite.encodingConfig.TxConfig,
					suite.accounts[0],
					sdk.NewCoin("stake", math.NewInt(101)),
					0,
					uint64(suite.ctx.BlockHeight())+2,
					[]testutils.Account{suite.accounts[0]},
				)
				suite.Require().NoError(err)

				bidTx2, err := testutils.CreateAuctionTxWithSigners(
					suite.encodingConfig.TxConfig,
					suite.accounts[1],
					sdk.NewCoin("stake", math.NewInt(102)),
					0,
					uint64(suite.ctx.BlockHeight())+2,
					[]testutils.Account{suite.accounts[1]},
				)
				suite.Require().NoError(err)

				return []sdk.Tx{bidTx1, bidTx2}, bidTx2
			},
			1000000000,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			bidTxs, winningBid := tc.getBidTxs()

			commitInfo := suite.createExtendedCommitInfoFromTxs(bidTxs)

			// Host the auction
			proposal := suite.proposalHandler.BuildTOB(suite.ctx, commitInfo, tc.maxBytes)

			// Size of the proposal should be less than or equal to the max bytes
			suite.Require().LessOrEqual(proposal.GetTotalTxBytes(), tc.maxBytes)

			if winningBid == nil {
				suite.Require().Len(proposal.GetTxs(), 0)
				suite.Require().Equal(proposal.GetTotalTxBytes(), int64(0))
			} else {
				// Get info about the winning bid
				winningBidBz, err := suite.encodingConfig.TxConfig.TxEncoder()(winningBid)
				suite.Require().NoError(err)

				auctionBidInfo, err := suite.tobLane.GetAuctionBidInfo(winningBid)
				suite.Require().NoError(err)

				// Verify that the size of the proposal is the size of the winning bid
				// plus the size of the bundle
				suite.Require().Equal(len(proposal.GetTxs()), len(auctionBidInfo.Transactions)+1)

				// Verify that the winning bid is the first transaction in the proposal
				suite.Require().Equal(proposal.GetTxs()[0], winningBidBz)

				// Verify the ordering of transactions in the proposal
				for index, tx := range proposal.GetTxs()[1:] {
					suite.Equal(tx, auctionBidInfo.Transactions[index])
				}
			}
		})
	}
}
