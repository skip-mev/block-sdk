package integration_test

import (
	"context"

	"cosmossdk.io/math"
	cmttypes "github.com/cometbft/cometbft/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/skip-mev/chaintestutil/account"
	"github.com/skip-mev/chaintestutil/network"
	"github.com/stretchr/testify/require"

	auctiontypes "github.com/skip-mev/block-sdk/x/auction/types"
)

func (s *NetworkTestSuite) TestAuctionWithValidBids() {
	cc, closeFn, err := s.NetworkSuite.GetGRPC()
	require.NoError(s.T(), err)
	defer closeFn()

	cmtClient, err := s.NetworkSuite.GetCometClient()
	require.NoError(s.T(), err)

	params, err := s.QueryAuctionParams()
	require.NoError(s.T(), err)

	fee := sdk.NewCoins(sdk.NewInt64Coin(sdk.DefaultBondDenom, 1000000))

	// Get the escrow account's initial balance
	beginEscrowBalances, err := s.NetworkSuite.Balances(*s.AuctionEscrow)
	require.NoError(s.T(), err)
	beginEscrowBalance := beginEscrowBalances.AmountOf(params.Params.ReserveFee.Denom)

	// Create and fund the bidders
	bidder1 := account.NewAccount()
	bidder2 := account.NewAccount()
	receiver := account.NewAccount()

	// Fund bidder1
	bz, err := s.NetworkSuite.CreateTxBytes(context.Background(),
		network.TxGenInfo{
			Account:       *s.Accounts[0],
			GasLimit:      10000000,
			TimeoutHeight: 100000000,
			Fee:           fee,
		},
		banktypes.NewMsgSend(
			s.Accounts[0].Address(),
			bidder1.Address(),
			sdk.NewCoins(sdk.NewInt64Coin(sdk.DefaultBondDenom, 10000000000))),
	)
	require.NoError(s.T(), err)
	res, err := s.NetworkSuite.BroadcastTxCommit(
		context.Background(),
		bz,
	)
	require.NoError(s.T(), err)
	require.Equal(s.T(), uint32(0), res.CheckTx.Code)
	require.Equal(s.T(), uint32(0), res.TxResult.Code)

	// Fund bidder2
	bz, err = s.NetworkSuite.CreateTxBytes(context.Background(),
		network.TxGenInfo{
			Account:       *s.Accounts[0],
			GasLimit:      10000000,
			TimeoutHeight: 100000000,
			Fee:           fee,
		},
		banktypes.NewMsgSend(
			s.Accounts[0].Address(),
			bidder2.Address(),
			sdk.NewCoins(sdk.NewInt64Coin(sdk.DefaultBondDenom, 10000000000))),
	)
	require.NoError(s.T(), err)
	res, err = s.NetworkSuite.BroadcastTxCommit(
		context.Background(),
		bz,
	)
	require.NoError(s.T(), err)
	require.Equal(s.T(), uint32(0), res.CheckTx.Code)
	require.Equal(s.T(), uint32(0), res.TxResult.Code)

	s.Run("two valid bids--balance/fee verification", func() {
		// Store the receiver's initial balance
		beginReceiverBalances, err := s.NetworkSuite.Balances(*receiver)
		require.NoError(s.T(), err)
		beginReceiverBalance := beginReceiverBalances.AmountOf(params.Params.ReserveFee.Denom)

		bid1Seq, _, err := getAccount(context.Background(), authtypes.NewQueryClient(cc), *bidder1)
		s.Require().NoError(err)
		bid2Seq, _, err := getAccount(context.Background(), authtypes.NewQueryClient(cc), *bidder2)
		s.Require().NoError(err)

		// Get current height
		resp, err := cmtClient.Status(context.Background())
		s.Require().NoError(err)
		bidHeight := uint64(resp.SyncInfo.LatestBlockHeight + 1)

		// Bidder1's send tx they want included
		send1Tx, err := s.NetworkSuite.CreateTxBytes(
			context.Background(),
			network.TxGenInfo{
				Account:          *bidder1,
				GasLimit:         1000000,
				TimeoutHeight:    bidHeight,
				Fee:              fee,
				Sequence:         bid1Seq + 1,
				OverrideSequence: true,
			},
			banktypes.NewMsgSend(bidder1.Address(), receiver.Address(), sdk.NewCoins(sdk.NewCoin(params.Params.ReserveFee.Denom, math.NewInt(1)))),
		)
		require.NoError(s.T(), err)

		// Bidder1's Bid Tx
		bid1Tx, err := s.NetworkSuite.CreateTxBytes(
			context.Background(),
			network.TxGenInfo{
				Account:          *bidder1,
				GasLimit:         1000009,
				TimeoutHeight:    bidHeight,
				Fee:              fee,
				Sequence:         bid1Seq,
				OverrideSequence: true,
			},
			auctiontypes.NewMsgAuctionBid(
				bidder1.Address(),
				params.Params.ReserveFee,
				[][]byte{send1Tx},
			),
		)
		require.NoError(s.T(), err)

		// Bidder2's send tx they want included
		send2Tx, err := s.NetworkSuite.CreateTxBytes(
			context.Background(),
			network.TxGenInfo{
				Account:          *bidder2,
				GasLimit:         1000000,
				TimeoutHeight:    bidHeight,
				Fee:              fee,
				Sequence:         bid2Seq + 1,
				OverrideSequence: true,
			},
			banktypes.NewMsgSend(bidder2.Address(), receiver.Address(), sdk.NewCoins(sdk.NewCoin(params.Params.ReserveFee.Denom, math.NewInt(2)))),
		)
		require.NoError(s.T(), err)

		// Bidder2's Bid Tx
		bid2Tx, err := s.NetworkSuite.CreateTxBytes(
			context.Background(),
			network.TxGenInfo{
				Account:          *bidder2,
				GasLimit:         1000000,
				TimeoutHeight:    bidHeight,
				Fee:              fee,
				Sequence:         bid2Seq,
				OverrideSequence: true,
			},
			auctiontypes.NewMsgAuctionBid(
				bidder2.Address(),
				params.Params.ReserveFee.Add(params.Params.MinBidIncrement),
				[][]byte{send2Tx},
			),
		)
		require.NoError(s.T(), err)

		// Broadcast the bids
		for _, tx := range [][]byte{bid1Tx, bid2Tx} {
			result, err := s.NetworkSuite.BroadcastTx(context.Background(), tx, network.BroadcastModeSync)
			require.NoError(s.T(), err)
			require.Equal(s.T(), uint32(0), result.Code)
		}
		require.NoError(s.T(), waitForTxCommit(context.Background(), cmtClient, cmttypes.Tx(bid2Tx).Hash()))

		// Validate that the receiver got the funds
		endReceiverBalances, err := s.NetworkSuite.Balances(*receiver)
		require.NoError(s.T(), err)
		endReceiverBalance := endReceiverBalances.AmountOf(params.Params.ReserveFee.Denom)
		require.Equal(s.T(), beginReceiverBalance.Add(math.NewInt(2)), endReceiverBalance)

		// Validate that the escrow got the funds
		endEscrowBalances, err := s.NetworkSuite.Balances(*s.AuctionEscrow)
		require.NoError(s.T(), err)
		endEscrowBalance := endEscrowBalances.AmountOf(params.Params.ReserveFee.Denom)
		require.Equal(s.T(), beginEscrowBalance.Add(math.NewInt(2)), endEscrowBalance)
	})
	s.Run("bid w/ too many txs", func() {
		bid1Seq, _, err := getAccount(context.Background(), authtypes.NewQueryClient(cc), *bidder1)
		s.Require().NoError(err)

		// Get current height
		resp, err := cmtClient.Status(context.Background())
		s.Require().NoError(err)
		bidHeight := uint64(resp.SyncInfo.LatestBlockHeight + 1)

		bundle := make([][]byte, 0, s.AuctionState.Params.MaxBundleSize+1)
		for i := 0; i <= int(s.AuctionState.Params.MaxBundleSize); i++ {
			// Bidder1's send tx they want included
			sendTx, err := s.NetworkSuite.CreateTxBytes(
				context.Background(),
				network.TxGenInfo{
					Account:          *bidder1,
					GasLimit:         1000000,
					TimeoutHeight:    bidHeight,
					Fee:              fee,
					Sequence:         bid1Seq + 1,
					OverrideSequence: true,
				},
				banktypes.NewMsgSend(bidder1.Address(), receiver.Address(), sdk.NewCoins(sdk.NewCoin(params.Params.ReserveFee.Denom, math.NewInt(1)))),
			)
			require.NoError(s.T(), err)
			bundle = append(bundle, sendTx)
		}

		// Bidder1's Bid Tx
		bid1Tx, err := s.NetworkSuite.CreateTxBytes(
			context.Background(),
			network.TxGenInfo{
				Account:          *bidder1,
				GasLimit:         1000009,
				TimeoutHeight:    bidHeight,
				Fee:              fee,
				Sequence:         bid1Seq,
				OverrideSequence: true,
			},
			auctiontypes.NewMsgAuctionBid(
				bidder1.Address(),
				params.Params.ReserveFee,
				bundle,
			),
		)
		require.NoError(s.T(), err)

		result, err := s.NetworkSuite.BroadcastTx(context.Background(), bid1Tx, network.BroadcastModeSync)
		require.NoError(s.T(), err)
		require.Equal(s.T(), uint32(1), result.Code)
	})
}
