package mev_test

import (
	"crypto/rand"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	testutils "github.com/skip-mev/block-sdk/v2/testutils"
)

func (s *MEVTestSuite) TestIsAuctionTx() {
	testCases := []struct {
		name          string
		createTx      func() sdk.Tx
		isAuctionTx   bool
		expectedError bool
	}{
		{
			"normal sdk tx",
			func() sdk.Tx {
				tx, err := testutils.CreateRandomTx(s.EncCfg.TxConfig, s.Accounts[0], 0, 2, 0, 0)
				s.Require().NoError(err)
				return tx
			},
			false,
			false,
		},
		{
			"malformed auction bid tx",
			func() sdk.Tx {
				msgAuctionBid, err := testutils.CreateMsgAuctionBid(s.EncCfg.TxConfig, s.Accounts[0], sdk.NewInt64Coin("stake", 100), 0, 2)
				s.Require().NoError(err)

				msgs := testutils.CreateRandomMsgs(s.Accounts[0].Address, 2)
				msgs = append(msgs, msgAuctionBid)

				tx, err := testutils.CreateTx(s.EncCfg.TxConfig, s.Accounts[0], 0, 0, msgs)
				s.Require().NoError(err)
				return tx
			},
			false,
			true,
		},
		{
			"valid auction bid tx",
			func() sdk.Tx {
				msgAuctionBid, err := testutils.CreateMsgAuctionBid(s.EncCfg.TxConfig, s.Accounts[0], sdk.NewInt64Coin("stake", 100), 0, 2)
				s.Require().NoError(err)

				msgs := []sdk.Msg{msgAuctionBid}

				tx, err := testutils.CreateTx(s.EncCfg.TxConfig, s.Accounts[0], 0, 0, msgs)
				s.Require().NoError(err)
				return tx
			},
			true,
			false,
		},
		{
			"tx with multiple MsgAuctionBid messages",
			func() sdk.Tx {
				bid1, err := testutils.CreateMsgAuctionBid(s.EncCfg.TxConfig, s.Accounts[0], sdk.NewInt64Coin("stake", 100), 0, 2)
				s.Require().NoError(err)

				bid2, err := testutils.CreateMsgAuctionBid(s.EncCfg.TxConfig, s.Accounts[0], sdk.NewInt64Coin("stake", 100), 1, 2)
				s.Require().NoError(err)

				msgs := []sdk.Msg{bid1, bid2}

				tx, err := testutils.CreateTx(s.EncCfg.TxConfig, s.Accounts[0], 0, 0, msgs)
				s.Require().NoError(err)
				return tx
			},
			false,
			true,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			tx := tc.createTx()

			bidInfo, err := s.Config.GetAuctionBidInfo(tx)

			s.Require().Equal(tc.isAuctionTx, bidInfo != nil)
			if tc.expectedError {
				s.Require().Error(err)
			} else {
				s.Require().NoError(err)
			}
		})
	}
}

func (s *MEVTestSuite) TestGetTransactionSigners() {
	testCases := []struct {
		name            string
		createTx        func() sdk.Tx
		expectedSigners []map[string]struct{}
		expectedError   bool
	}{
		{
			"normal auction tx",
			func() sdk.Tx {
				tx, err := testutils.CreateAuctionTxWithSigners(
					s.EncCfg.TxConfig,
					s.Accounts[0],
					sdk.NewCoin("stake", math.NewInt(100)),
					1,
					0,
					s.Accounts[0:1],
				)
				s.Require().NoError(err)

				return tx
			},
			[]map[string]struct{}{
				{
					s.Accounts[0].Address.String(): {},
				},
			},
			false,
		},
		{
			"normal sdk tx",
			func() sdk.Tx {
				tx, err := testutils.CreateRandomTx(s.EncCfg.TxConfig, s.Accounts[0], 0, 10, 0, 0)
				s.Require().NoError(err)

				return tx
			},
			nil,
			true,
		},
		{
			"multiple signers on auction tx",
			func() sdk.Tx {
				tx, err := testutils.CreateAuctionTxWithSigners(
					s.EncCfg.TxConfig,
					s.Accounts[0],
					sdk.NewCoin("stake", math.NewInt(100)),
					1,
					0,
					s.Accounts[0:3],
				)
				s.Require().NoError(err)

				return tx
			},
			[]map[string]struct{}{
				{
					s.Accounts[0].Address.String(): {},
				},
				{
					s.Accounts[1].Address.String(): {},
				},
				{
					s.Accounts[2].Address.String(): {},
				},
			},
			false,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			tx := tc.createTx()

			bidInfo, _ := s.Config.GetAuctionBidInfo(tx)
			if tc.expectedError {
				s.Require().Nil(bidInfo)
			} else {
				s.Require().Equal(tc.expectedSigners, bidInfo.Signers)
			}
		})
	}
}

func (s *MEVTestSuite) TestWrapBundleTransaction() {
	testCases := []struct {
		name           string
		createBundleTx func() (sdk.Tx, []byte)
		expectedError  bool
	}{
		{
			"normal sdk tx",
			func() (sdk.Tx, []byte) {
				tx, err := testutils.CreateRandomTx(s.EncCfg.TxConfig, s.Accounts[0], 0, 1, 0, 0)
				s.Require().NoError(err)

				bz, err := s.EncCfg.TxConfig.TxEncoder()(tx)
				s.Require().NoError(err)

				return tx, bz
			},
			false,
		},
		{
			"random bytes with expected failure",
			func() (sdk.Tx, []byte) {
				bz := make([]byte, 100)
				rand.Read(bz)

				return nil, bz
			},
			true,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			tx, bz := tc.createBundleTx()

			wrappedTx, err := s.Config.WrapBundleTransaction(bz)
			if tc.expectedError {
				s.Require().Error(err)
			} else {
				s.Require().NoError(err)

				txBytes, err := s.EncCfg.TxConfig.TxEncoder()(tx)
				s.Require().NoError(err)

				wrappedTxBytes, err := s.EncCfg.TxConfig.TxEncoder()(wrappedTx)
				s.Require().NoError(err)

				s.Require().Equal(txBytes, wrappedTxBytes)
			}
		})
	}
}

func (s *MEVTestSuite) TestGetBidder() {
	testCases := []struct {
		name           string
		createTx       func() sdk.Tx
		expectedBidder string
		expectedError  bool
		isAuctionTx    bool
	}{
		{
			"normal sdk tx",
			func() sdk.Tx {
				tx, err := testutils.CreateRandomTx(s.EncCfg.TxConfig, s.Accounts[0], 0, 1, 0, 0)
				s.Require().NoError(err)

				return tx
			},
			"",
			false,
			false,
		},
		{
			"valid auction tx",
			func() sdk.Tx {
				msgAuctionBid, err := testutils.CreateMsgAuctionBid(s.EncCfg.TxConfig, s.Accounts[0], sdk.NewInt64Coin("stake", 100), 0, 2)
				s.Require().NoError(err)

				msgs := []sdk.Msg{msgAuctionBid}

				tx, err := testutils.CreateTx(s.EncCfg.TxConfig, s.Accounts[0], 0, 0, msgs)
				s.Require().NoError(err)
				return tx
			},
			s.Accounts[0].Address.String(),
			false,
			true,
		},
		{
			"invalid auction tx",
			func() sdk.Tx {
				msgAuctionBid, err := testutils.CreateMsgAuctionBid(s.EncCfg.TxConfig, s.Accounts[0], sdk.NewInt64Coin("stake", 100), 0, 2)
				s.Require().NoError(err)

				randomMsg := testutils.CreateRandomMsgs(s.Accounts[0].Address, 1)[0]
				s.Require().NoError(err)

				msgs := []sdk.Msg{msgAuctionBid, randomMsg}

				tx, err := testutils.CreateTx(s.EncCfg.TxConfig, s.Accounts[0], 0, 0, msgs)
				s.Require().NoError(err)
				return tx
			},
			"",
			true,
			false,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			tx := tc.createTx()

			bidInfo, err := s.Config.GetAuctionBidInfo(tx)
			if tc.expectedError {
				s.Require().Error(err)
			} else {
				s.Require().NoError(err)

				if tc.isAuctionTx {
					s.Require().Equal(tc.expectedBidder, bidInfo.Bidder.String())
				}
			}
		})
	}
}

func (s *MEVTestSuite) TestGetBid() {
	testCases := []struct {
		name          string
		createTx      func() sdk.Tx
		expectedBid   sdk.Coin
		expectedError bool
		isAuctionTx   bool
	}{
		{
			"normal sdk tx",
			func() sdk.Tx {
				tx, err := testutils.CreateRandomTx(s.EncCfg.TxConfig, s.Accounts[0], 0, 1, 0, 0)
				s.Require().NoError(err)

				return tx
			},
			sdk.Coin{},
			false,
			false,
		},
		{
			"valid auction tx",
			func() sdk.Tx {
				msgAuctionBid, err := testutils.CreateMsgAuctionBid(s.EncCfg.TxConfig, s.Accounts[0], sdk.NewInt64Coin("stake", 100), 0, 2)
				s.Require().NoError(err)

				msgs := []sdk.Msg{msgAuctionBid}

				tx, err := testutils.CreateTx(s.EncCfg.TxConfig, s.Accounts[0], 0, 0, msgs)
				s.Require().NoError(err)
				return tx
			},
			sdk.NewInt64Coin("stake", 100),
			false,
			true,
		},
		{
			"invalid auction tx",
			func() sdk.Tx {
				msgAuctionBid, err := testutils.CreateMsgAuctionBid(s.EncCfg.TxConfig, s.Accounts[0], sdk.NewInt64Coin("stake", 100), 0, 2)
				s.Require().NoError(err)

				randomMsg := testutils.CreateRandomMsgs(s.Accounts[0].Address, 1)[0]
				s.Require().NoError(err)

				msgs := []sdk.Msg{msgAuctionBid, randomMsg}

				tx, err := testutils.CreateTx(s.EncCfg.TxConfig, s.Accounts[0], 0, 0, msgs)
				s.Require().NoError(err)
				return tx
			},
			sdk.Coin{},
			true,
			false,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			tx := tc.createTx()

			bidInfo, err := s.Config.GetAuctionBidInfo(tx)
			if tc.expectedError {
				s.Require().Error(err)
			} else {
				s.Require().NoError(err)

				if tc.isAuctionTx {
					s.Require().Equal(tc.expectedBid, bidInfo.Bid)
				}
			}
		})
	}
}

func (s *MEVTestSuite) TestGetBundledTransactions() {
	testCases := []struct {
		name          string
		createTx      func() (sdk.Tx, [][]byte)
		expectedError bool
		isAuctionTx   bool
	}{
		{
			"normal sdk tx",
			func() (sdk.Tx, [][]byte) {
				tx, err := testutils.CreateRandomTx(s.EncCfg.TxConfig, s.Accounts[0], 0, 1, 0, 0)
				s.Require().NoError(err)

				return tx, nil
			},
			false,
			false,
		},
		{
			"valid auction tx",
			func() (sdk.Tx, [][]byte) {
				msgAuctionBid, err := testutils.CreateMsgAuctionBid(s.EncCfg.TxConfig, s.Accounts[0], sdk.NewInt64Coin("stake", 100), 0, 2)
				s.Require().NoError(err)

				msgs := []sdk.Msg{msgAuctionBid}

				tx, err := testutils.CreateTx(s.EncCfg.TxConfig, s.Accounts[0], 0, 0, msgs)
				s.Require().NoError(err)
				return tx, msgAuctionBid.Transactions
			},
			false,
			true,
		},
		{
			"invalid auction tx",
			func() (sdk.Tx, [][]byte) {
				msgAuctionBid, err := testutils.CreateMsgAuctionBid(s.EncCfg.TxConfig, s.Accounts[0], sdk.NewInt64Coin("stake", 100), 0, 2)
				s.Require().NoError(err)

				randomMsg := testutils.CreateRandomMsgs(s.Accounts[0].Address, 1)[0]
				s.Require().NoError(err)

				msgs := []sdk.Msg{msgAuctionBid, randomMsg}

				tx, err := testutils.CreateTx(s.EncCfg.TxConfig, s.Accounts[0], 0, 0, msgs)
				s.Require().NoError(err)
				return tx, nil
			},
			true,
			false,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			tx, expectedBundledTxs := tc.createTx()

			bidInfo, err := s.Config.GetAuctionBidInfo(tx)
			if tc.expectedError {
				s.Require().Error(err)
			} else {
				s.Require().NoError(err)

				if tc.isAuctionTx {
					s.Require().Equal(expectedBundledTxs, bidInfo.Transactions)
				}
			}
		})
	}
}

func (s *MEVTestSuite) TestGetTimeout() {
	testCases := []struct {
		name            string
		createTx        func() sdk.Tx
		expectedError   bool
		isAuctionTx     bool
		expectedTimeout uint64
	}{
		{
			"normal sdk tx",
			func() sdk.Tx {
				tx, err := testutils.CreateRandomTx(s.EncCfg.TxConfig, s.Accounts[0], 0, 1, 1, 0)
				s.Require().NoError(err)

				return tx
			},
			false,
			false,
			1,
		},
		{
			"valid auction tx",
			func() sdk.Tx {
				msgAuctionBid, err := testutils.CreateMsgAuctionBid(s.EncCfg.TxConfig, s.Accounts[0], sdk.NewInt64Coin("stake", 100), 0, 2)
				s.Require().NoError(err)

				msgs := []sdk.Msg{msgAuctionBid}

				tx, err := testutils.CreateTx(s.EncCfg.TxConfig, s.Accounts[0], 0, 10, msgs)
				s.Require().NoError(err)
				return tx
			},
			false,
			true,
			10,
		},
		{
			"invalid auction tx",
			func() sdk.Tx {
				msgAuctionBid, err := testutils.CreateMsgAuctionBid(s.EncCfg.TxConfig, s.Accounts[0], sdk.NewInt64Coin("stake", 100), 0, 2)
				s.Require().NoError(err)

				randomMsg := testutils.CreateRandomMsgs(s.Accounts[0].Address, 1)[0]
				s.Require().NoError(err)

				msgs := []sdk.Msg{msgAuctionBid, randomMsg}

				tx, err := testutils.CreateTx(s.EncCfg.TxConfig, s.Accounts[0], 0, 10, msgs)
				s.Require().NoError(err)
				return tx
			},
			true,
			false,
			10,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			tx := tc.createTx()

			bidInfo, err := s.Config.GetAuctionBidInfo(tx)
			if tc.expectedError {
				s.Require().Error(err)
			} else {
				s.Require().NoError(err)

				if tc.isAuctionTx {
					s.Require().Equal(tc.expectedTimeout, bidInfo.Timeout)
				}
			}
		})
	}
}
