package mempool_test

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	testutils "github.com/skip-mev/pob/testutils"
)

func (suite *IntegrationTestSuite) TestIsAuctionTx() {
	testCases := []struct {
		name          string
		createTx      func() sdk.Tx
		isAuctionTx   bool
		expectedError bool
	}{
		{
			"normal sdk tx",
			func() sdk.Tx {
				tx, err := testutils.CreateRandomTx(suite.encCfg.TxConfig, suite.accounts[0], 0, 2, 0)
				suite.Require().NoError(err)
				return tx
			},
			false,
			false,
		},
		{
			"malformed auction bid tx",
			func() sdk.Tx {
				msgAuctionBid, err := testutils.CreateMsgAuctionBid(suite.encCfg.TxConfig, suite.accounts[0], sdk.NewInt64Coin("foo", 100), 0, 2)
				suite.Require().NoError(err)

				msgs := testutils.CreateRandomMsgs(suite.accounts[0].Address, 2)
				msgs = append(msgs, msgAuctionBid)

				tx, err := testutils.CreateTx(suite.encCfg.TxConfig, suite.accounts[0], 0, 0, msgs)
				suite.Require().NoError(err)
				return tx
			},
			false,
			true,
		},
		{
			"valid auction bid tx",
			func() sdk.Tx {
				msgAuctionBid, err := testutils.CreateMsgAuctionBid(suite.encCfg.TxConfig, suite.accounts[0], sdk.NewInt64Coin("foo", 100), 0, 2)
				suite.Require().NoError(err)

				msgs := []sdk.Msg{msgAuctionBid}

				tx, err := testutils.CreateTx(suite.encCfg.TxConfig, suite.accounts[0], 0, 0, msgs)
				suite.Require().NoError(err)
				return tx
			},
			true,
			false,
		},
		{
			"tx with multiple MsgAuctionBid messages",
			func() sdk.Tx {
				bid1, err := testutils.CreateMsgAuctionBid(suite.encCfg.TxConfig, suite.accounts[0], sdk.NewInt64Coin("foo", 100), 0, 2)
				suite.Require().NoError(err)

				bid2, err := testutils.CreateMsgAuctionBid(suite.encCfg.TxConfig, suite.accounts[0], sdk.NewInt64Coin("foo", 100), 1, 2)
				suite.Require().NoError(err)

				msgs := []sdk.Msg{bid1, bid2}

				tx, err := testutils.CreateTx(suite.encCfg.TxConfig, suite.accounts[0], 0, 0, msgs)
				suite.Require().NoError(err)
				return tx
			},
			false,
			true,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			tx := tc.createTx()

			isAuctionTx, err := suite.config.IsAuctionTx(tx)

			suite.Require().Equal(tc.isAuctionTx, isAuctionTx)
			if tc.expectedError {
				suite.Require().Error(err)
			} else {
				suite.Require().NoError(err)
			}
		})
	}
}

func (suite *IntegrationTestSuite) TestGetTransactionSigners() {
	testCases := []struct {
		name            string
		createTx        func() []byte
		expectedSigners []string
	}{
		{
			"normal sdk tx",
			func() []byte {
				tx, err := testutils.CreateRandomTx(suite.encCfg.TxConfig, suite.accounts[0], 0, 1, 0)
				suite.Require().NoError(err)

				bz, err := suite.encCfg.TxConfig.TxEncoder()(tx)
				suite.Require().NoError(err)

				return bz
			},
			[]string{suite.accounts[0].Address.String()},
		},
		{
			"normal sdk tx with several messages",
			func() []byte {
				tx, err := testutils.CreateRandomTx(suite.encCfg.TxConfig, suite.accounts[0], 0, 10, 0)
				suite.Require().NoError(err)

				bz, err := suite.encCfg.TxConfig.TxEncoder()(tx)
				suite.Require().NoError(err)

				return bz
			},
			[]string{suite.accounts[0].Address.String()},
		},
		{
			"multiple signers on tx",
			func() []byte {
				tx, err := testutils.CreateTxWithSigners(suite.encCfg.TxConfig, 0, 0, suite.accounts[0:3])
				suite.Require().NoError(err)

				bz, err := suite.encCfg.TxConfig.TxEncoder()(tx)
				suite.Require().NoError(err)

				return bz
			},
			[]string{suite.accounts[0].Address.String(), suite.accounts[1].Address.String(), suite.accounts[2].Address.String()},
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			tx := tc.createTx()

			signers, err := suite.config.GetTransactionSigners(tx)
			suite.Require().NoError(err)
			suite.Require().Equal(len(tc.expectedSigners), len(signers))

			for _, signer := range tc.expectedSigners {
				suite.Require().Contains(signers, signer)
			}
		})
	}
}

func (suite *IntegrationTestSuite) TestGetBundleSigners() {
	testCases := []struct {
		name            string
		createBundle    func() [][]byte
		expectedSigners [][]string
	}{
		{
			"single bundle with one signer",
			func() [][]byte {
				tx, err := testutils.CreateRandomTx(suite.encCfg.TxConfig, suite.accounts[0], 0, 1, 0)
				suite.Require().NoError(err)

				bz, err := suite.encCfg.TxConfig.TxEncoder()(tx)
				suite.Require().NoError(err)

				return [][]byte{bz}
			},
			[][]string{{suite.accounts[0].Address.String()}},
		},
		{
			"single bundle with multiple signers",
			func() [][]byte {
				tx, err := testutils.CreateTxWithSigners(suite.encCfg.TxConfig, 0, 0, suite.accounts[0:3])
				suite.Require().NoError(err)

				bz, err := suite.encCfg.TxConfig.TxEncoder()(tx)
				suite.Require().NoError(err)

				return [][]byte{bz}
			},
			[][]string{{suite.accounts[0].Address.String(), suite.accounts[1].Address.String(), suite.accounts[2].Address.String()}},
		},
		{
			"multiple bundles with one signer",
			func() [][]byte {
				tx1, err := testutils.CreateRandomTx(suite.encCfg.TxConfig, suite.accounts[0], 0, 1, 0)
				suite.Require().NoError(err)

				bz1, err := suite.encCfg.TxConfig.TxEncoder()(tx1)
				suite.Require().NoError(err)

				tx2, err := testutils.CreateRandomTx(suite.encCfg.TxConfig, suite.accounts[1], 0, 1, 0)
				suite.Require().NoError(err)

				bz2, err := suite.encCfg.TxConfig.TxEncoder()(tx2)
				suite.Require().NoError(err)

				return [][]byte{bz1, bz2}
			},
			[][]string{{suite.accounts[0].Address.String()}, {suite.accounts[1].Address.String()}},
		},
		{
			"multiple bundles with multiple signers",
			func() [][]byte {
				tx1, err := testutils.CreateTxWithSigners(suite.encCfg.TxConfig, 0, 0, suite.accounts[0:3])
				suite.Require().NoError(err)

				bz1, err := suite.encCfg.TxConfig.TxEncoder()(tx1)
				suite.Require().NoError(err)

				tx2, err := testutils.CreateTxWithSigners(suite.encCfg.TxConfig, 0, 0, suite.accounts[3:6])
				suite.Require().NoError(err)

				bz2, err := suite.encCfg.TxConfig.TxEncoder()(tx2)
				suite.Require().NoError(err)

				return [][]byte{bz1, bz2}
			},
			[][]string{{suite.accounts[0].Address.String(), suite.accounts[1].Address.String(), suite.accounts[2].Address.String()}, {suite.accounts[3].Address.String(), suite.accounts[4].Address.String(), suite.accounts[5].Address.String()}},
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			bundle := tc.createBundle()

			signers, err := suite.config.GetBundleSigners(bundle)
			suite.Require().NoError(err)
			suite.Require().Equal(len(tc.expectedSigners), len(signers))

			for i, bundleSigners := range tc.expectedSigners {
				suite.Require().Equal(len(bundleSigners), len(signers[i]))

				for _, signer := range bundleSigners {
					suite.Require().Contains(signers[i], signer)
				}
			}
		})
	}
}

func (suite *IntegrationTestSuite) TestWrapBundleTransaction() {
	testCases := []struct {
		name           string
		createBundleTx func() (sdk.Tx, []byte)
	}{
		{
			"normal sdk tx",
			func() (sdk.Tx, []byte) {
				tx, err := testutils.CreateRandomTx(suite.encCfg.TxConfig, suite.accounts[0], 0, 1, 0)
				suite.Require().NoError(err)

				bz, err := suite.encCfg.TxConfig.TxEncoder()(tx)
				suite.Require().NoError(err)

				return tx, bz
			},
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			tx, bz := tc.createBundleTx()

			wrappedTx, err := suite.config.WrapBundleTransaction(bz)
			suite.Require().NoError(err)

			txBytes, err := suite.encCfg.TxConfig.TxEncoder()(tx)
			suite.Require().NoError(err)

			wrappedTxBytes, err := suite.encCfg.TxConfig.TxEncoder()(wrappedTx)
			suite.Require().NoError(err)

			suite.Require().Equal(txBytes, wrappedTxBytes)
		})
	}
}

func (suite *IntegrationTestSuite) TestGetBidder() {
	testCases := []struct {
		name           string
		createTx       func() sdk.Tx
		expectedBidder string
		expectedError  bool
	}{
		{
			"normal sdk tx",
			func() sdk.Tx {
				tx, err := testutils.CreateRandomTx(suite.encCfg.TxConfig, suite.accounts[0], 0, 1, 0)
				suite.Require().NoError(err)

				return tx
			},
			"",
			true,
		},
		{
			"valid auction tx",
			func() sdk.Tx {
				msgAuctionBid, err := testutils.CreateMsgAuctionBid(suite.encCfg.TxConfig, suite.accounts[0], sdk.NewInt64Coin("foo", 100), 0, 2)
				suite.Require().NoError(err)

				msgs := []sdk.Msg{msgAuctionBid}

				tx, err := testutils.CreateTx(suite.encCfg.TxConfig, suite.accounts[0], 0, 0, msgs)
				suite.Require().NoError(err)
				return tx
			},
			suite.accounts[0].Address.String(),
			false,
		},
		{
			"invalid auction tx",
			func() sdk.Tx {
				msgAuctionBid, err := testutils.CreateMsgAuctionBid(suite.encCfg.TxConfig, suite.accounts[0], sdk.NewInt64Coin("foo", 100), 0, 2)
				suite.Require().NoError(err)

				randomMsg := testutils.CreateRandomMsgs(suite.accounts[0].Address, 1)[0]
				suite.Require().NoError(err)

				msgs := []sdk.Msg{msgAuctionBid, randomMsg}

				tx, err := testutils.CreateTx(suite.encCfg.TxConfig, suite.accounts[0], 0, 0, msgs)
				suite.Require().NoError(err)
				return tx
			},
			"",
			true,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			tx := tc.createTx()

			bidder, err := suite.config.GetBidder(tx)
			if tc.expectedError {
				suite.Require().Error(err)
			} else {
				suite.Require().NoError(err)
				suite.Require().Equal(tc.expectedBidder, bidder.String())
			}
		})
	}
}

func (suite *IntegrationTestSuite) TestGetBid() {
	testCases := []struct {
		name          string
		createTx      func() sdk.Tx
		expectedBid   sdk.Coin
		expectedError bool
	}{
		{
			"normal sdk tx",
			func() sdk.Tx {
				tx, err := testutils.CreateRandomTx(suite.encCfg.TxConfig, suite.accounts[0], 0, 1, 0)
				suite.Require().NoError(err)

				return tx
			},
			sdk.Coin{},
			true,
		},
		{
			"valid auction tx",
			func() sdk.Tx {
				msgAuctionBid, err := testutils.CreateMsgAuctionBid(suite.encCfg.TxConfig, suite.accounts[0], sdk.NewInt64Coin("foo", 100), 0, 2)
				suite.Require().NoError(err)

				msgs := []sdk.Msg{msgAuctionBid}

				tx, err := testutils.CreateTx(suite.encCfg.TxConfig, suite.accounts[0], 0, 0, msgs)
				suite.Require().NoError(err)
				return tx
			},
			sdk.NewInt64Coin("foo", 100),
			false,
		},
		{
			"invalid auction tx",
			func() sdk.Tx {
				msgAuctionBid, err := testutils.CreateMsgAuctionBid(suite.encCfg.TxConfig, suite.accounts[0], sdk.NewInt64Coin("foo", 100), 0, 2)
				suite.Require().NoError(err)

				randomMsg := testutils.CreateRandomMsgs(suite.accounts[0].Address, 1)[0]
				suite.Require().NoError(err)

				msgs := []sdk.Msg{msgAuctionBid, randomMsg}

				tx, err := testutils.CreateTx(suite.encCfg.TxConfig, suite.accounts[0], 0, 0, msgs)
				suite.Require().NoError(err)
				return tx
			},
			sdk.Coin{},
			true,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			tx := tc.createTx()

			bid, err := suite.config.GetBid(tx)
			if tc.expectedError {
				suite.Require().Error(err)
			} else {
				suite.Require().NoError(err)
				suite.Require().Equal(tc.expectedBid, bid)
			}
		})
	}
}

func (suite *IntegrationTestSuite) TestGetBundledTransactions() {
	testCases := []struct {
		name          string
		createTx      func() (sdk.Tx, [][]byte)
		expectedError bool
	}{
		{
			"normal sdk tx",
			func() (sdk.Tx, [][]byte) {
				tx, err := testutils.CreateRandomTx(suite.encCfg.TxConfig, suite.accounts[0], 0, 1, 0)
				suite.Require().NoError(err)

				return tx, nil
			},
			true,
		},
		{
			"valid auction tx",
			func() (sdk.Tx, [][]byte) {
				msgAuctionBid, err := testutils.CreateMsgAuctionBid(suite.encCfg.TxConfig, suite.accounts[0], sdk.NewInt64Coin("foo", 100), 0, 2)
				suite.Require().NoError(err)

				msgs := []sdk.Msg{msgAuctionBid}

				tx, err := testutils.CreateTx(suite.encCfg.TxConfig, suite.accounts[0], 0, 0, msgs)
				suite.Require().NoError(err)
				return tx, msgAuctionBid.Transactions
			},
			false,
		},
		{
			"invalid auction tx",
			func() (sdk.Tx, [][]byte) {
				msgAuctionBid, err := testutils.CreateMsgAuctionBid(suite.encCfg.TxConfig, suite.accounts[0], sdk.NewInt64Coin("foo", 100), 0, 2)
				suite.Require().NoError(err)

				randomMsg := testutils.CreateRandomMsgs(suite.accounts[0].Address, 1)[0]
				suite.Require().NoError(err)

				msgs := []sdk.Msg{msgAuctionBid, randomMsg}

				tx, err := testutils.CreateTx(suite.encCfg.TxConfig, suite.accounts[0], 0, 0, msgs)
				suite.Require().NoError(err)
				return tx, nil
			},
			true,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			tx, expectedBundledTxs := tc.createTx()

			bundledTxs, err := suite.config.GetBundledTransactions(tx)
			if tc.expectedError {
				suite.Require().Error(err)
			} else {
				suite.Require().NoError(err)
				suite.Require().Equal(expectedBundledTxs, bundledTxs)
			}
		})
	}
}

func (suite *IntegrationTestSuite) TestGetTimeout() {
	testCases := []struct {
		name            string
		createTx        func() sdk.Tx
		expectedError   bool
		expectedTimeout uint64
	}{
		{
			"normal sdk tx",
			func() sdk.Tx {
				tx, err := testutils.CreateRandomTx(suite.encCfg.TxConfig, suite.accounts[0], 0, 1, 1)
				suite.Require().NoError(err)

				return tx
			},
			false,
			1,
		},
		{
			"valid auction tx",
			func() sdk.Tx {
				msgAuctionBid, err := testutils.CreateMsgAuctionBid(suite.encCfg.TxConfig, suite.accounts[0], sdk.NewInt64Coin("foo", 100), 0, 2)
				suite.Require().NoError(err)

				msgs := []sdk.Msg{msgAuctionBid}

				tx, err := testutils.CreateTx(suite.encCfg.TxConfig, suite.accounts[0], 0, 10, msgs)
				suite.Require().NoError(err)
				return tx
			},
			false,
			10,
		},
		{
			"invalid auction tx",
			func() sdk.Tx {
				msgAuctionBid, err := testutils.CreateMsgAuctionBid(suite.encCfg.TxConfig, suite.accounts[0], sdk.NewInt64Coin("foo", 100), 0, 2)
				suite.Require().NoError(err)

				randomMsg := testutils.CreateRandomMsgs(suite.accounts[0].Address, 1)[0]
				suite.Require().NoError(err)

				msgs := []sdk.Msg{msgAuctionBid, randomMsg}

				tx, err := testutils.CreateTx(suite.encCfg.TxConfig, suite.accounts[0], 0, 10, msgs)
				suite.Require().NoError(err)
				return tx
			},
			false,
			10,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			tx := tc.createTx()

			timeout, err := suite.config.GetTimeout(tx)
			if tc.expectedError {
				suite.Require().Error(err)
			} else {
				suite.Require().NoError(err)
				suite.Require().Equal(tc.expectedTimeout, timeout)
			}
		})
	}
}
