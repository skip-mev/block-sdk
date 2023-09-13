package integration

import (
	"context"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	interchaintest "github.com/strangelove-ventures/interchaintest/v7"
	"github.com/strangelove-ventures/interchaintest/v7/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v7/ibc"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

const (
	initBalance = 1000000000000
)

// IntegrationTestSuite runs the Block SDK integration test-suite against a given interchaintest specification
type IntegrationTestSuite struct {
	suite.Suite
	// spec
	spec *interchaintest.ChainSpec
	// chain
	chain ibc.Chain
	// interchain
	ic *interchaintest.Interchain
	// users
	user1, user2, user3 ibc.Wallet
	// denom
	denom string
}

func NewIntegrationTestSuiteFromSpec(spec *interchaintest.ChainSpec) *IntegrationTestSuite {
	return &IntegrationTestSuite{
		spec:  spec,
		denom: "stake",
	}
}

func (s *IntegrationTestSuite) WithDenom(denom string) *IntegrationTestSuite {
	s.denom = denom
	return s
}

func (s *IntegrationTestSuite) SetupSuite() {
	// build the chain
	s.T().Log("building chain with spec", s.spec)
	s.chain = ChainBuilderFromChainSpec(s.T(), s.spec)

	// build the interchain
	s.T().Log("building interchain")
	ctx := context.Background()
	s.ic = BuildInterchain(s.T(), ctx, s.chain)

	// get the users
	s.user1 = interchaintest.GetAndFundTestUsers(s.T(), ctx, s.T().Name(), initBalance, s.chain)[0]
	s.user2 = interchaintest.GetAndFundTestUsers(s.T(), ctx, s.T().Name(), initBalance, s.chain)[0]
	s.user3 = interchaintest.GetAndFundTestUsers(s.T(), ctx, s.T().Name(), initBalance, s.chain)[0]
}

func (s *IntegrationTestSuite) TearDownSuite() {
	// close the interchain
	s.ic.Close()
}

func (s *IntegrationTestSuite) SetupSubTest() {
	// wait for 1 block height
	// query height
	height, err := s.chain.(*cosmos.CosmosChain).Height(context.Background())
	require.NoError(s.T(), err)
	WaitForHeight(s.T(), s.chain.(*cosmos.CosmosChain), height+1)
}

func (s *IntegrationTestSuite) TestQueryParams() {
	// query params
	params := QueryAuctionParams(s.T(), s.chain)

	// expect validate to pass
	require.NoError(s.T(), params.Validate())
}

// TestValidBids tests the execution of various valid auction bids. There are a few
// invariants that are tested:
//
//  1. The order of transactions in a bundle is preserved when bids are valid.
//  2. All transactions execute as expected.
//  3. The balance of the escrow account should be updated correctly.
//  4. Top of block bids will be included in block proposals before other transactions
func (s *IntegrationTestSuite) TestValidBids() {
	params := QueryAuctionParams(s.T(), s.chain)
	escrowAddr := sdk.AccAddress(params.EscrowAccountAddress).String()

	s.Run("Valid Auction Bid", func() {
		// get escrow account balance before bid
		escrowAcctBalanceBeforeBid := QueryAccountBalance(s.T(), s.chain, escrowAddr, params.ReserveFee.Denom)

		// create bundle w/ a single tx
		// create message send tx
		tx := banktypes.NewMsgSend(s.user1.Address(), s.user2.Address(), sdk.NewCoins(sdk.NewCoin(s.denom, sdk.NewInt(100))))

		// create the MsgAuctioBid
		bidAmt := params.ReserveFee
		bid, bundledTxs := CreateAuctionBidMsg(s.T(), context.Background(), s.user1, s.chain.(*cosmos.CosmosChain), bidAmt, []Tx{
			{
				User: s.user1,
				Msgs: []sdk.Msg{
					tx,
				},
				SequenceIncrement: 1,
			},
		})

		height, err := s.chain.(*cosmos.CosmosChain).Height(context.Background())
		require.NoError(s.T(), err)

		// broadcast + wait for the tx to be included in a block
		res := BroadcastTxs(s.T(), context.Background(), s.chain.(*cosmos.CosmosChain), []Tx{
			{
				User:   s.user1,
				Msgs:   []sdk.Msg{bid},
				Height: height + 1,
			},
		})
		height = height + 1

		// wait for next height
		WaitForHeight(s.T(), s.chain.(*cosmos.CosmosChain), height)

		// query + verify the block
		block := Block(s.T(), s.chain.(*cosmos.CosmosChain), int64(height))
		VerifyBlock(s.T(), block, 0, TxHash(res[0]), bundledTxs)

		// ensure that the escrow account has the correct balance
		escrowAcctBalanceAfterBid := QueryAccountBalance(s.T(), s.chain, escrowAddr, params.ReserveFee.Denom)
		expectedIncrement := escrowAddressIncrement(bidAmt.Amount, params.ProposerFee)
		require.Equal(s.T(), escrowAcctBalanceBeforeBid+expectedIncrement, escrowAcctBalanceAfterBid)
	})

	s.Run("Valid bid with multiple other transactions", func() {
		// get escrow account balance before bid
		escrowAcctBalanceBeforeBid := QueryAccountBalance(s.T(), s.chain, escrowAddr, params.ReserveFee.Denom)

		// create the bundle w/ a single tx
		// bank-send msg
		msgs := make([]sdk.Msg, 2)
		msgs[0] = banktypes.NewMsgSend(s.user1.Address(), s.user2.Address(), sdk.NewCoins(sdk.NewCoin(s.denom, sdk.NewInt(100))))
		msgs[1] = banktypes.NewMsgSend(s.user2.Address(), s.user3.Address(), sdk.NewCoins(sdk.NewCoin(s.denom, sdk.NewInt(100))))

		// create the MsgAuctionBid
		bidAmt := params.ReserveFee
		bid, bundledTxs := CreateAuctionBidMsg(s.T(), context.Background(), s.user1, s.chain.(*cosmos.CosmosChain), bidAmt, []Tx{
			{
				User:              s.user1,
				Msgs:              msgs[0:1],
				SequenceIncrement: 1,
			},
		})

		height, err := s.chain.(*cosmos.CosmosChain).Height(context.Background())
		require.NoError(s.T(), err)

		// create the messages to be broadcast
		msgsToBcast := make([]Tx, 0)
		msgsToBcast = append(msgsToBcast, Tx{
			User:   s.user1,
			Msgs:   []sdk.Msg{bid},
			Height: height + 1,
		})

		msgsToBcast = append(msgsToBcast, Tx{
			User:   s.user2,
			Msgs:   msgs[1:2],
			Height: height + 1,
		})

		regular_txs := BroadcastTxs(s.T(), context.Background(), s.chain.(*cosmos.CosmosChain), msgsToBcast)

		// get the block at the next height
		WaitForHeight(s.T(), s.chain.(*cosmos.CosmosChain), height+1)
		block := Block(s.T(), s.chain.(*cosmos.CosmosChain), int64(height+1))

		// verify the block
		bidTxHash := TxHash(regular_txs[0])
		VerifyBlock(s.T(), block, 0, bidTxHash, append(bundledTxs, regular_txs[1:]...))

		// ensure that escrow account has the correct balance
		escrowAcctBalanceAfterBid := QueryAccountBalance(s.T(), s.chain, escrowAddr, params.ReserveFee.Denom)
		expectedIncrement := escrowAddressIncrement(bidAmt.Amount, params.ProposerFee)
		require.Equal(s.T(), escrowAcctBalanceBeforeBid+expectedIncrement, escrowAcctBalanceAfterBid)
	})

	s.Run("iterative bidding from the same account", func() {
		// get escrow account balance before bid
		escrowAcctBalanceBeforeBid := QueryAccountBalance(s.T(), s.chain, escrowAddr, params.ReserveFee.Denom)

		// create multi-tx valid bundle
		// bank-send msg
		txs := make([]Tx, 2)
		txs[0] = Tx{
			User:              s.user1,
			Msgs:              []sdk.Msg{banktypes.NewMsgSend(s.user1.Address(), s.user2.Address(), sdk.NewCoins(sdk.NewCoin(s.denom, sdk.NewInt(100))))},
			SequenceIncrement: 1,
		}
		txs[1] = Tx{
			User:              s.user1,
			Msgs:              []sdk.Msg{banktypes.NewMsgSend(s.user1.Address(), s.user3.Address(), sdk.NewCoins(sdk.NewCoin(s.denom, sdk.NewInt(100))))},
			SequenceIncrement: 2,
		}
		// create bundle
		bidAmt := params.ReserveFee
		bid, bundledTxs := CreateAuctionBidMsg(s.T(), context.Background(), s.user1, s.chain.(*cosmos.CosmosChain), bidAmt, txs)
		// create 2 more bundle w same txs from same user
		bid2, _ := CreateAuctionBidMsg(s.T(), context.Background(), s.user1, s.chain.(*cosmos.CosmosChain), bidAmt.Add(params.MinBidIncrement), txs)
		bid3, _ := CreateAuctionBidMsg(s.T(), context.Background(), s.user1, s.chain.(*cosmos.CosmosChain), bidAmt.Add(params.MinBidIncrement).Add(params.MinBidIncrement), txs)

		// query height
		height, err := s.chain.(*cosmos.CosmosChain).Height(context.Background())
		require.NoError(s.T(), err)

		// wait for the next height to broadcast
		WaitForHeight(s.T(), s.chain.(*cosmos.CosmosChain), height+1)
		height++

		// broadcast all bids
		broadcastedTxs := BroadcastTxs(s.T(), context.Background(), s.chain.(*cosmos.CosmosChain), []Tx{
			{
				User:               s.user1,
				Msgs:               []sdk.Msg{bid},
				Height:             height + 1,
				SkipInclusionCheck: true,
			},
			{
				User:               s.user1,
				Msgs:               []sdk.Msg{bid2},
				Height:             height + 1,
				SkipInclusionCheck: true,
			},
			{
				User:   s.user1,
				Msgs:   []sdk.Msg{bid3},
				Height: height + 1,
			},
		})

		// Verify the block
		WaitForHeight(s.T(), s.chain.(*cosmos.CosmosChain), height+1)
		VerifyBlock(s.T(), Block(s.T(), s.chain.(*cosmos.CosmosChain), int64(height+1)), 0, TxHash(broadcastedTxs[2]), bundledTxs)

		//  check escrow account balance
		escrowAcctBalanceAfterBid := QueryAccountBalance(s.T(), s.chain, escrowAddr, params.ReserveFee.Denom)
		expectedIncrement := escrowAddressIncrement(bidAmt.Add(params.MinBidIncrement.Add(params.MinBidIncrement)).Amount, params.ProposerFee)
		require.Equal(s.T(), escrowAcctBalanceBeforeBid+expectedIncrement, escrowAcctBalanceAfterBid)
	})

	s.Run("bid with a bundle with transactions that are already in the mempool", func() {
		// reset
		height, err := s.chain.(*cosmos.CosmosChain).Height(context.Background())
		require.NoError(s.T(), err)

		// wait for the next height
		WaitForHeight(s.T(), s.chain.(*cosmos.CosmosChain), height+1)

		// escrow account balance
		escrowAcctBalanceBeforeBid := QueryAccountBalance(s.T(), s.chain, escrowAddr, params.ReserveFee.Denom)

		// create valid bundle
		// bank-send msg
		txs := make([]Tx, 2)
		txs[0] = Tx{
			User: s.user1,
			Msgs: []sdk.Msg{banktypes.NewMsgSend(s.user1.Address(), s.user2.Address(), sdk.NewCoins(sdk.NewCoin(s.denom, sdk.NewInt(100))))},
		}
		txs[1] = Tx{
			User:              s.user1,
			Msgs:              []sdk.Msg{banktypes.NewMsgSend(s.user1.Address(), s.user3.Address(), sdk.NewCoins(sdk.NewCoin(s.denom, sdk.NewInt(100))))},
			SequenceIncrement: 1,
		}

		// create bundle
		bidAmt := params.ReserveFee
		bid, bundledTxs := CreateAuctionBidMsg(s.T(), context.Background(), s.user2, s.chain.(*cosmos.CosmosChain), bidAmt, txs)

		// get chain height
		height, err = s.chain.(*cosmos.CosmosChain).Height(context.Background())
		require.NoError(s.T(), err)

		// broadcast txs in the bundle to network + bundle + extra
		broadcastedTxs := BroadcastTxs(s.T(), context.Background(), s.chain.(*cosmos.CosmosChain), []Tx{txs[0], txs[1], {
			User:   s.user2,
			Msgs:   []sdk.Msg{bid},
			Height: height + 1,
		}, {
			User: s.user3,
			Msgs: []sdk.Msg{banktypes.NewMsgSend(s.user3.Address(), s.user1.Address(), sdk.NewCoins(sdk.NewCoin(s.denom, sdk.NewInt(100))))},
		}})

		// query next block
		WaitForHeight(s.T(), s.chain.(*cosmos.CosmosChain), height+1)
		block := Block(s.T(), s.chain.(*cosmos.CosmosChain), int64(height+1))

		// check block
		VerifyBlock(s.T(), block, 0, TxHash(broadcastedTxs[2]), append(bundledTxs, broadcastedTxs[3]))

		// check escrow account balance
		escrowAcctBalanceAfterBid := QueryAccountBalance(s.T(), s.chain, escrowAddr, params.ReserveFee.Denom)
		expectedIncrement := escrowAddressIncrement(bidAmt.Amount, params.ProposerFee)
		require.Equal(s.T(), escrowAcctBalanceBeforeBid+expectedIncrement, escrowAcctBalanceAfterBid)
	})
}

// TestMultipleBids tests the execution of various valid auction bids in the same block. There are a few
// invariants that are tested:
//
//  1. The order of transactions in a bundle is preserved when bids are valid.
//  2. All transactions execute as expected.
//  3. The balance of the escrow account should be updated correctly.
//  4. Top of block bids will be included in block proposals before other transactions
//     that are included in the same block.
//  5. If there is a block that has multiple valid bids with timeouts that are sufficiently far apart,
//     the bids should be executed respecting the highest bids until the timeout is reached.
func (s *IntegrationTestSuite) TestMultipleBids() {
	params := QueryAuctionParams(s.T(), s.chain)
	escrowAddr := sdk.AccAddress(params.EscrowAccountAddress).String()

	s.Run("broadcasting bids to two different validators (both should execute over several blocks) with same bid", func() {
		// escrow account balance
		escrowAcctBalanceBeforeBid := QueryAccountBalance(s.T(), s.chain, escrowAddr, params.ReserveFee.Denom)

		// create bid 1
		// bank-send msg
		msg := Tx{
			User:              s.user1,
			Msgs:              []sdk.Msg{banktypes.NewMsgSend(s.user1.Address(), s.user2.Address(), sdk.NewCoins(sdk.NewCoin(s.denom, sdk.NewInt(100))))},
			SequenceIncrement: 1,
		}
		// create bid1
		bidAmt := params.ReserveFee
		bid1, bundledTxs := CreateAuctionBidMsg(s.T(), context.Background(), s.user1, s.chain.(*cosmos.CosmosChain), bidAmt, []Tx{msg})

		// create bid 2
		msg2 := Tx{
			User:              s.user2,
			Msgs:              []sdk.Msg{banktypes.NewMsgSend(s.user2.Address(), s.user3.Address(), sdk.NewCoins(sdk.NewCoin(s.denom, sdk.NewInt(100))))},
			SequenceIncrement: 1,
		}
		// create bid2 w/ higher bid than bid1
		bid2, bundledTxs2 := CreateAuctionBidMsg(s.T(), context.Background(), s.user2, s.chain.(*cosmos.CosmosChain), bidAmt.Add(params.MinBidIncrement), []Tx{msg2})
		// get chain height
		height, err := s.chain.(*cosmos.CosmosChain).Height(context.Background())
		require.NoError(s.T(), err)

		// broadcast both bids
		txs := BroadcastTxs(s.T(), context.Background(), s.chain.(*cosmos.CosmosChain), []Tx{
			{
				User:               s.user1,
				Msgs:               []sdk.Msg{bid1},
				Height:             height + 2,
				SkipInclusionCheck: true,
			},
			{
				User:   s.user2,
				Msgs:   []sdk.Msg{bid2},
				Height: height + 1,
			},
		})

		// query next block
		WaitForHeight(s.T(), s.chain.(*cosmos.CosmosChain), height+1)
		block := Block(s.T(), s.chain.(*cosmos.CosmosChain), int64(height+1))

		// check bid2 was included first
		VerifyBlock(s.T(), block, 0, TxHash(txs[1]), bundledTxs2)

		// check next block
		WaitForHeight(s.T(), s.chain.(*cosmos.CosmosChain), height+2)
		block = Block(s.T(), s.chain.(*cosmos.CosmosChain), int64(height+2))

		// check bid1 was included second
		VerifyBlock(s.T(), block, 0, TxHash(txs[0]), bundledTxs)

		// check escrow balance
		escrowAcctBalanceAfterBid := QueryAccountBalance(s.T(), s.chain, escrowAddr, params.ReserveFee.Denom)
		expectedIncrement := escrowAddressIncrement(bidAmt.Add(params.MinBidIncrement.Add(bidAmt)).Amount, params.ProposerFee)
		require.Equal(s.T(), escrowAcctBalanceBeforeBid+expectedIncrement, escrowAcctBalanceAfterBid)
	})

	s.Run("Multiple bid transactions with second bid being smaller than min bid increment", func() {
		// escrow account balance
		escrowAcctBalanceBeforeBid := QueryAccountBalance(s.T(), s.chain, escrowAddr, params.ReserveFee.Denom)

		// create bid 1
		// bank-send msg
		tx := Tx{
			User:              s.user1,
			Msgs:              []sdk.Msg{banktypes.NewMsgSend(s.user1.Address(), s.user2.Address(), sdk.NewCoins(sdk.NewCoin(s.denom, sdk.NewInt(100))))},
			SequenceIncrement: 1,
		}
		// create bid1
		bidAmt := params.ReserveFee
		bid1, bundledTxs := CreateAuctionBidMsg(s.T(), context.Background(), s.user1, s.chain.(*cosmos.CosmosChain), bidAmt, []Tx{tx})

		// create bid 2
		tx2 := Tx{
			User:              s.user2,
			Msgs:              []sdk.Msg{banktypes.NewMsgSend(s.user2.Address(), s.user3.Address(), sdk.NewCoins(sdk.NewCoin(s.denom, sdk.NewInt(100))))},
			SequenceIncrement: 1,
		}
		// create bid2 w/ higher bid than bid1
		bid2, _ := CreateAuctionBidMsg(s.T(), context.Background(), s.user2, s.chain.(*cosmos.CosmosChain), bidAmt, []Tx{tx2})

		// get chain height
		height, err := s.chain.(*cosmos.CosmosChain).Height(context.Background())
		require.NoError(s.T(), err)

		// broadcast both bids
		txs := BroadcastTxs(s.T(), context.Background(), s.chain.(*cosmos.CosmosChain), []Tx{
			{
				User:   s.user1,
				Msgs:   []sdk.Msg{bid1},
				Height: height + 1,
			},
			{
				User:       s.user2,
				Msgs:       []sdk.Msg{bid2},
				Height:     height + 1,
				ExpectFail: true,
			},
		})

		// query next block
		WaitForHeight(s.T(), s.chain.(*cosmos.CosmosChain), height+1)
		block := Block(s.T(), s.chain.(*cosmos.CosmosChain), int64(height+1))

		// check bid2 was included first
		VerifyBlock(s.T(), block, 0, TxHash(txs[0]), bundledTxs)

		// check escrow balance
		escrowAcctBalanceAfterBid := QueryAccountBalance(s.T(), s.chain, escrowAddr, params.ReserveFee.Denom)
		expectedIncrement := escrowAddressIncrement(bidAmt.Amount, params.ProposerFee)
		require.Equal(s.T(), escrowAcctBalanceBeforeBid+expectedIncrement, escrowAcctBalanceAfterBid)
	})

	s.Run("Multiple bid transactions from diff accounts with second bid being smaller than min bid increment", func() {
		// escrow account balance
		escrowAcctBalanceBeforeBid := QueryAccountBalance(s.T(), s.chain, escrowAddr, params.ReserveFee.Denom)

		// create bid 1
		// bank-send msg
		msg := Tx{
			User:              s.user1,
			Msgs:              []sdk.Msg{banktypes.NewMsgSend(s.user1.Address(), s.user2.Address(), sdk.NewCoins(sdk.NewCoin(s.denom, sdk.NewInt(100))))},
			SequenceIncrement: 1,
		}
		// create bid1
		bidAmt := params.ReserveFee
		bid1, bundledTxs := CreateAuctionBidMsg(s.T(), context.Background(), s.user1, s.chain.(*cosmos.CosmosChain), bidAmt, []Tx{msg})

		// create bid 2
		msg2 := Tx{
			User:              s.user2,
			Msgs:              []sdk.Msg{banktypes.NewMsgSend(s.user2.Address(), s.user3.Address(), sdk.NewCoins(sdk.NewCoin(s.denom, sdk.NewInt(100))))},
			SequenceIncrement: 1,
		}
		// create bid2 w/ higher bid than bid1
		bid2, _ := CreateAuctionBidMsg(s.T(), context.Background(), s.user1, s.chain.(*cosmos.CosmosChain), bidAmt, []Tx{msg2})
		// get chain height
		height, err := s.chain.(*cosmos.CosmosChain).Height(context.Background())
		require.NoError(s.T(), err)

		// broadcast both bids
		txs := BroadcastTxs(s.T(), context.Background(), s.chain.(*cosmos.CosmosChain), []Tx{
			{
				User:   s.user1,
				Msgs:   []sdk.Msg{bid1},
				Height: height + 1,
			},
			{
				User:              s.user1,
				Msgs:              []sdk.Msg{bid2},
				SequenceIncrement: 1,
				Height:            height + 1,
				ExpectFail:        true,
			},
		})

		// query next block
		WaitForHeight(s.T(), s.chain.(*cosmos.CosmosChain), height+1)
		block := Block(s.T(), s.chain.(*cosmos.CosmosChain), int64(height+1))

		// check bid2 was included first
		VerifyBlock(s.T(), block, 0, TxHash(txs[0]), bundledTxs)

		// check escrow balance
		escrowAcctBalanceAfterBid := QueryAccountBalance(s.T(), s.chain, escrowAddr, params.ReserveFee.Denom)
		expectedIncrement := escrowAddressIncrement(bidAmt.Amount, params.ProposerFee)
		require.Equal(s.T(), escrowAcctBalanceBeforeBid+expectedIncrement, escrowAcctBalanceAfterBid)
	})

	s.Run("Multiple transactions with increasing bids but first bid has same bundle so it should fail in later block", func() {
		// escrow account balance
		escrowAcctBalanceBeforeBid := QueryAccountBalance(s.T(), s.chain, escrowAddr, params.ReserveFee.Denom)

		// create bid 1
		// bank-send msg
		msg := Tx{
			User:              s.user1,
			Msgs:              []sdk.Msg{banktypes.NewMsgSend(s.user1.Address(), s.user2.Address(), sdk.NewCoins(sdk.NewCoin(s.denom, sdk.NewInt(100))))},
			SequenceIncrement: 1,
		}
		// create bid1
		bidAmt := params.ReserveFee
		bid1, bundledTxs := CreateAuctionBidMsg(s.T(), context.Background(), s.user1, s.chain.(*cosmos.CosmosChain), bidAmt, []Tx{msg})

		// create bid2 w/ higher bid than bid1
		bid2, _ := CreateAuctionBidMsg(s.T(), context.Background(), s.user1, s.chain.(*cosmos.CosmosChain), bidAmt.Add(params.MinBidIncrement), []Tx{msg})
		// get chain height
		height, err := s.chain.(*cosmos.CosmosChain).Height(context.Background())
		require.NoError(s.T(), err)

		// broadcast both bids
		txs := BroadcastTxs(s.T(), context.Background(), s.chain.(*cosmos.CosmosChain), []Tx{
			{
				User:   s.user1,
				Msgs:   []sdk.Msg{bid2},
				Height: height + 1,
			},
			{
				User:              s.user1,
				Msgs:              []sdk.Msg{bid1},
				Height:            height + 2,
				SequenceIncrement: 1,
				ExpectFail:        true,
			},
		})

		// query next block
		WaitForHeight(s.T(), s.chain.(*cosmos.CosmosChain), height+1)
		block := Block(s.T(), s.chain.(*cosmos.CosmosChain), int64(height+1))

		// check bid2 was included first
		VerifyBlock(s.T(), block, 0, TxHash(txs[0]), bundledTxs)

		// check escrow balance
		escrowAcctBalanceAfterBid := QueryAccountBalance(s.T(), s.chain, escrowAddr, params.ReserveFee.Denom)
		expectedIncrement := escrowAddressIncrement(bidAmt.Add(params.MinBidIncrement).Amount, params.ProposerFee)
		require.Equal(s.T(), escrowAcctBalanceBeforeBid+expectedIncrement, escrowAcctBalanceAfterBid)
	})

	s.Run("Multiple transactions from diff. account with increasing bids but first bid has same bundle so it should fail in later block", func() {
		// escrow account balance
		escrowAcctBalanceBeforeBid := QueryAccountBalance(s.T(), s.chain, escrowAddr, params.ReserveFee.Denom)

		// create bid 1
		// bank-send msg
		msg := Tx{
			User: s.user3,
			Msgs: []sdk.Msg{banktypes.NewMsgSend(s.user3.Address(), s.user2.Address(), sdk.NewCoins(sdk.NewCoin(s.denom, sdk.NewInt(100))))},
		}

		// create bid1
		bidAmt := params.ReserveFee
		bid1, bundledTxs := CreateAuctionBidMsg(s.T(), context.Background(), s.user1, s.chain.(*cosmos.CosmosChain), bidAmt, []Tx{msg})

		// create bid2 w/ higher bid than bid1
		bid2, _ := CreateAuctionBidMsg(s.T(), context.Background(), s.user2, s.chain.(*cosmos.CosmosChain), bidAmt.Add(params.MinBidIncrement), []Tx{msg})
		// get chain height
		height, err := s.chain.(*cosmos.CosmosChain).Height(context.Background())
		require.NoError(s.T(), err)

		// broadcast both bids
		txs := BroadcastTxs(s.T(), context.Background(), s.chain.(*cosmos.CosmosChain), []Tx{
			{
				User:   s.user2,
				Msgs:   []sdk.Msg{bid2},
				Height: height + 1,
			},
			{
				User:       s.user1,
				Msgs:       []sdk.Msg{bid1},
				Height:     height + 1,
				ExpectFail: true,
			},
		})

		// query next block
		WaitForHeight(s.T(), s.chain.(*cosmos.CosmosChain), height+1)
		block := Block(s.T(), s.chain.(*cosmos.CosmosChain), int64(height+1))

		// check bid2 was included first
		VerifyBlock(s.T(), block, 0, TxHash(txs[0]), bundledTxs)

		// check escrow balance
		escrowAcctBalanceAfterBid := QueryAccountBalance(s.T(), s.chain, escrowAddr, params.ReserveFee.Denom)
		expectedIncrement := escrowAddressIncrement(bidAmt.Add(params.MinBidIncrement).Amount, params.ProposerFee)
		require.Equal(s.T(), escrowAcctBalanceBeforeBid+expectedIncrement, escrowAcctBalanceAfterBid)
	})

	s.Run("Multiple transactions with increasing bids and different bundles", func() {
		// escrow account balance
		escrowAcctBalanceBeforeBid := QueryAccountBalance(s.T(), s.chain, escrowAddr, params.ReserveFee.Denom)

		// create bid 1
		// bank-send msg
		msg := Tx{
			User:              s.user1,
			Msgs:              []sdk.Msg{banktypes.NewMsgSend(s.user1.Address(), s.user2.Address(), sdk.NewCoins(sdk.NewCoin(s.denom, sdk.NewInt(100))))},
			SequenceIncrement: 1,
		}
		// create bid1
		bidAmt := params.ReserveFee
		bid1, bundledTxs := CreateAuctionBidMsg(s.T(), context.Background(), s.user1, s.chain.(*cosmos.CosmosChain), bidAmt, []Tx{msg})

		// create bid2
		// create a second message
		msg2 := Tx{
			User:              s.user2,
			Msgs:              []sdk.Msg{banktypes.NewMsgSend(s.user2.Address(), s.user3.Address(), sdk.NewCoins(sdk.NewCoin(s.denom, sdk.NewInt(100))))},
			SequenceIncrement: 1,
		}

		// create bid2 w/ higher bid than bid1
		bid2, bundledTxs2 := CreateAuctionBidMsg(s.T(), context.Background(), s.user2, s.chain.(*cosmos.CosmosChain), bidAmt.Add(params.MinBidIncrement), []Tx{msg2})
		// get chain height
		height, err := s.chain.(*cosmos.CosmosChain).Height(context.Background())
		require.NoError(s.T(), err)

		// broadcast both bids
		txs := BroadcastTxs(s.T(), context.Background(), s.chain.(*cosmos.CosmosChain), []Tx{
			{
				User:               s.user1,
				Msgs:               []sdk.Msg{bid1},
				Height:             height + 2,
				SkipInclusionCheck: true,
			},
			{
				User:   s.user2,
				Msgs:   []sdk.Msg{bid2},
				Height: height + 1,
			},
		})

		// query next block
		WaitForHeight(s.T(), s.chain.(*cosmos.CosmosChain), height+1)
		block := Block(s.T(), s.chain.(*cosmos.CosmosChain), int64(height+1))

		// check bid2 was included first
		VerifyBlock(s.T(), block, 0, TxHash(txs[1]), bundledTxs2)

		// query next block and check tx inclusion
		WaitForHeight(s.T(), s.chain.(*cosmos.CosmosChain), height+2)
		block = Block(s.T(), s.chain.(*cosmos.CosmosChain), int64(height+2))

		// check bid1 was included second
		VerifyBlock(s.T(), block, 0, TxHash(txs[0]), bundledTxs)

		// check escrow balance
		escrowAcctBalanceAfterBid := QueryAccountBalance(s.T(), s.chain, escrowAddr, params.ReserveFee.Denom)
		expectedIncrement := escrowAddressIncrement(bidAmt.Add(params.MinBidIncrement.Add(bidAmt)).Amount, params.ProposerFee)
		require.Equal(s.T(), escrowAcctBalanceBeforeBid+expectedIncrement, escrowAcctBalanceAfterBid)
	})
}

func (s *IntegrationTestSuite) TestInvalidBids() {
	params := QueryAuctionParams(s.T(), s.chain)
	escrowAddr := sdk.AccAddress(params.EscrowAccountAddress).String()

	s.Run("searcher is attempting to submit a bundle that includes another bid tx", func() {
		// create bid tx
		msg := Tx{
			User:              s.user1,
			Msgs:              []sdk.Msg{banktypes.NewMsgSend(s.user1.Address(), s.user2.Address(), sdk.NewCoins(sdk.NewCoin(s.denom, sdk.NewInt(100))))},
			SequenceIncrement: 2,
		}
		bidAmt := params.ReserveFee
		bid, _ := CreateAuctionBidMsg(s.T(), context.Background(), s.user1, s.chain.(*cosmos.CosmosChain), bidAmt, []Tx{msg})

		height, err := s.chain.(*cosmos.CosmosChain).Height(context.Background())
		// wrap bidTx in another tx
		wrappedBid, _ := CreateAuctionBidMsg(s.T(), context.Background(), s.user1, s.chain.(*cosmos.CosmosChain), bidAmt, []Tx{
			{
				User:              s.user1,
				Msgs:              []sdk.Msg{bid},
				SequenceIncrement: 1,
				Height:            height + 1,
			},
		})

		require.NoError(s.T(), err)

		// broadcast wrapped bid, and expect a failure
		BroadcastTxs(s.T(), context.Background(), s.chain.(*cosmos.CosmosChain), []Tx{
			{
				User:       s.user1,
				Msgs:       []sdk.Msg{wrappedBid},
				Height:     height + 1,
				ExpectFail: true,
			},
		})
	})

	s.Run("Invalid bid that is attempting to bid more than their balance", func() {
		// create bid tx
		msg := Tx{
			User:              s.user1,
			Msgs:              []sdk.Msg{banktypes.NewMsgSend(s.user1.Address(), s.user2.Address(), sdk.NewCoins(sdk.NewCoin(s.denom, sdk.NewInt(100))))},
			SequenceIncrement: 2,
		}
		bidAmt := sdk.NewCoin(s.denom, sdk.NewInt(1000000000000000000))
		bid, _ := CreateAuctionBidMsg(s.T(), context.Background(), s.user1, s.chain.(*cosmos.CosmosChain), bidAmt, []Tx{msg})

		height, err := s.chain.(*cosmos.CosmosChain).Height(context.Background())
		require.NoError(s.T(), err)

		// broadcast wrapped bid, and expect a failure
		SimulateTx(s.T(), context.Background(), s.chain.(*cosmos.CosmosChain), s.user1, height+1, true, []sdk.Msg{bid}...)
	})

	s.Run("Invalid bid that is attempting to front-run/sandwich", func() {
		// create bid tx
		msg := Tx{
			User:              s.user1,
			Msgs:              []sdk.Msg{banktypes.NewMsgSend(s.user1.Address(), s.user2.Address(), sdk.NewCoins(sdk.NewCoin(s.denom, sdk.NewInt(100))))},
			SequenceIncrement: 1,
		}
		msg2 := Tx{
			User: s.user2,
			Msgs: []sdk.Msg{banktypes.NewMsgSend(s.user2.Address(), s.user3.Address(), sdk.NewCoins(sdk.NewCoin(s.denom, sdk.NewInt(100))))},
		}
		msg3 := Tx{
			User:              s.user1,
			Msgs:              []sdk.Msg{banktypes.NewMsgSend(s.user1.Address(), s.user3.Address(), sdk.NewCoins(sdk.NewCoin(s.denom, sdk.NewInt(100))))},
			SequenceIncrement: 2,
		}

		bidAmt := params.ReserveFee
		bid, _ := CreateAuctionBidMsg(s.T(), context.Background(), s.user1, s.chain.(*cosmos.CosmosChain), bidAmt, []Tx{msg, msg2, msg3})

		height, err := s.chain.(*cosmos.CosmosChain).Height(context.Background())
		require.NoError(s.T(), err)

		// broadcast wrapped bid, and expect a failure
		SimulateTx(s.T(), context.Background(), s.chain.(*cosmos.CosmosChain), s.user1, height+1, true, []sdk.Msg{bid}...)
	})

	s.Run("Invalid bid that includes an invalid bundle tx", func() {
		// create bid tx
		msg := Tx{
			User:              s.user1,
			Msgs:              []sdk.Msg{banktypes.NewMsgSend(s.user1.Address(), s.user2.Address(), sdk.NewCoins(sdk.NewCoin(s.denom, sdk.NewInt(100))))},
			SequenceIncrement: 2,
		}
		bidAmt := params.ReserveFee
		bid, _ := CreateAuctionBidMsg(s.T(), context.Background(), s.user1, s.chain.(*cosmos.CosmosChain), bidAmt, []Tx{msg})

		height, err := s.chain.(*cosmos.CosmosChain).Height(context.Background())
		require.NoError(s.T(), err)

		// broadcast wrapped bid, and expect a failure
		BroadcastTxs(s.T(), context.Background(), s.chain.(*cosmos.CosmosChain), []Tx{
			{
				User:       s.user1,
				Msgs:       []sdk.Msg{bid},
				ExpectFail: true,
				Height:     height + 1,
			},
		})
	})

	s.Run("Invalid auction bid with a bid smaller than the reserve fee", func() {
		// create bid tx
		msg := Tx{
			User:              s.user1,
			Msgs:              []sdk.Msg{banktypes.NewMsgSend(s.user1.Address(), s.user2.Address(), sdk.NewCoins(sdk.NewCoin(s.denom, sdk.NewInt(100))))},
			SequenceIncrement: 1,
		}

		// create bid smaller than reserve
		bidAmt := sdk.NewCoin(s.denom, sdk.NewInt(0))
		bid, _ := CreateAuctionBidMsg(s.T(), context.Background(), s.user1, s.chain.(*cosmos.CosmosChain), bidAmt, []Tx{msg})

		height, err := s.chain.(*cosmos.CosmosChain).Height(context.Background())
		require.NoError(s.T(), err)

		// broadcast wrapped bid, and expect a failure
		SimulateTx(s.T(), context.Background(), s.chain.(*cosmos.CosmosChain), s.user1, height+1, true, []sdk.Msg{bid}...)
	})

	s.Run("Invalid auction bid with too many transactions in the bundle", func() {
		// create bid tx
		msgs := make([]Tx, 4)

		for i := range msgs {
			msgs[i] = Tx{
				User:              s.user1,
				Msgs:              []sdk.Msg{banktypes.NewMsgSend(s.user1.Address(), s.user2.Address(), sdk.NewCoins(sdk.NewCoin(s.denom, sdk.NewInt(100))))},
				SequenceIncrement: uint64(i + 1),
			}
		}

		// create bid smaller than reserve
		bidAmt := sdk.NewCoin(s.denom, sdk.NewInt(0))
		bid, _ := CreateAuctionBidMsg(s.T(), context.Background(), s.user1, s.chain.(*cosmos.CosmosChain), bidAmt, msgs)

		height, err := s.chain.(*cosmos.CosmosChain).Height(context.Background())
		require.NoError(s.T(), err)

		// broadcast wrapped bid, and expect a failure
		SimulateTx(s.T(), context.Background(), s.chain.(*cosmos.CosmosChain), s.user1, height+1, true, []sdk.Msg{bid}...)
	})

	s.Run("invalid auction bid that has an invalid timeout", func() {
		// create bid tx
		msg := Tx{
			User:              s.user1,
			Msgs:              []sdk.Msg{banktypes.NewMsgSend(s.user1.Address(), s.user2.Address(), sdk.NewCoins(sdk.NewCoin(s.denom, sdk.NewInt(100))))},
			SequenceIncrement: 1,
		}

		// create bid smaller than reserve
		bidAmt := sdk.NewCoin(s.denom, sdk.NewInt(0))
		bid, _ := CreateAuctionBidMsg(s.T(), context.Background(), s.user1, s.chain.(*cosmos.CosmosChain), bidAmt, []Tx{msg})

		// broadcast wrapped bid, and expect a failure
		SimulateTx(s.T(), context.Background(), s.chain.(*cosmos.CosmosChain), s.user1, 0, true, []sdk.Msg{bid}...)
	})

	s.Run("Invalid bid that includes valid transactions that are in the mempool", func() {
		// get escrow account balance before bid
		escrowAcctBalanceBeforeBid := QueryAccountBalance(s.T(), s.chain, escrowAddr, params.ReserveFee.Denom)

		// create bundle w/ a single tx
		// create message send tx
		tx := banktypes.NewMsgSend(s.user2.Address(), s.user2.Address(), sdk.NewCoins(sdk.NewCoin(s.denom, sdk.NewInt(100))))

		// create the MsgAuctioBid (this should fail b.c same tx is repeated twice)
		bidAmt := params.ReserveFee
		bid, _ := CreateAuctionBidMsg(s.T(), context.Background(), s.user1, s.chain.(*cosmos.CosmosChain), bidAmt, []Tx{
			{
				User: s.user2,
				Msgs: []sdk.Msg{
					tx,
				},
			},
			{
				User: s.user2,
				Msgs: []sdk.Msg{tx},
			},
		})

		height, err := s.chain.(*cosmos.CosmosChain).Height(context.Background())
		require.NoError(s.T(), err)

		// broadcast + wait for the tx to be included in a block
		txs := BroadcastTxs(s.T(), context.Background(), s.chain.(*cosmos.CosmosChain), []Tx{
			{
				User:       s.user1,
				Msgs:       []sdk.Msg{bid},
				Height:     height + 1,
				ExpectFail: true,
			},
			{
				User:   s.user2,
				Msgs:   []sdk.Msg{tx},
				Height: height + 1,
			},
		})

		// wait for next height
		WaitForHeight(s.T(), s.chain.(*cosmos.CosmosChain), height+1)

		// query + verify the block expect no bid
		block := Block(s.T(), s.chain.(*cosmos.CosmosChain), int64(height+1))
		VerifyBlock(s.T(), block, 0, "", txs[1:])

		// ensure that the escrow account has the correct balance (same as before)
		escrowAcctBalanceAfterBid := QueryAccountBalance(s.T(), s.chain, escrowAddr, params.ReserveFee.Denom)
		require.Equal(s.T(), escrowAcctBalanceAfterBid, escrowAcctBalanceBeforeBid)
	})
}

func escrowAddressIncrement(bid math.Int, proposerFee sdk.Dec) int64 {
	return int64(bid.Sub(sdk.NewDecFromInt(bid).Mul(proposerFee).RoundInt()).Int64())
}

// TestFreeLane tests that the application correctly handles free lanes. There are a few invariants that are tested:
//
// 1. Transactions that qualify as free should not be deducted any fees.
// 2. Transactions that do not qualify as free should be deducted the correct fees.
func (s *IntegrationTestSuite) TestFreeLane() {
	validators := QueryValidators(s.T(), s.chain.(*cosmos.CosmosChain))
	require.True(s.T(), len(validators) > 0)

	delegation := sdk.NewCoin(s.denom, sdk.NewInt(100))

	s.Run("valid free lane transaction", func() {
		// query balance of account before tx submission
		balanceBefore := QueryAccountBalance(s.T(), s.chain.(*cosmos.CosmosChain), s.user1.FormattedAddress(), s.denom)

		// create a free tx (MsgDelegate), broadcast and wait for commit
		BroadcastTxs(s.T(), context.Background(), s.chain.(*cosmos.CosmosChain), []Tx{
			{
				User: s.user1,
				Msgs: []sdk.Msg{
					stakingtypes.NewMsgDelegate(
						sdk.AccAddress(s.user1.Address()),
						sdk.ValAddress(validators[0]),
						delegation,
					),
				},
				GasPrice: 10,
			},
		})

		// check balance of account
		balanceAfter := QueryAccountBalance(s.T(), s.chain.(*cosmos.CosmosChain), s.user1.FormattedAddress(), s.denom)
		require.Equal(s.T(), balanceBefore, balanceAfter+delegation.Amount.Int64())
	})

	s.Run("normal tx with free tx in same block", func() {
		user1BalanceBefore := QueryAccountBalance(s.T(), s.chain.(*cosmos.CosmosChain), s.user1.FormattedAddress(), s.denom)
		user2BalanceBefore := QueryAccountBalance(s.T(), s.chain.(*cosmos.CosmosChain), s.user2.FormattedAddress(), s.denom)

		// user1 submits a free-tx, user2 submits a normal tx
		BroadcastTxs(s.T(), context.Background(), s.chain.(*cosmos.CosmosChain), []Tx{
			{
				User: s.user1,
				Msgs: []sdk.Msg{
					stakingtypes.NewMsgDelegate(
						sdk.AccAddress(s.user1.Address()),
						sdk.ValAddress(validators[0]),
						delegation,
					),
				},
				GasPrice: 10,
			},
			{
				User: s.user2,
				Msgs: []sdk.Msg{
					banktypes.NewMsgSend(
						sdk.AccAddress(s.user2.Address()),
						sdk.AccAddress(s.user3.Address()),
						sdk.NewCoins(sdk.NewCoin(s.denom, sdk.NewInt(100))),
					),
				},
				GasPrice: 10,
			},
		})

		// check balance after, user1 balance only diff by delegation
		user1BalanceAfter := QueryAccountBalance(s.T(), s.chain.(*cosmos.CosmosChain), s.user1.FormattedAddress(), s.denom)
		user2BalanceAfter := QueryAccountBalance(s.T(), s.chain.(*cosmos.CosmosChain), s.user2.FormattedAddress(), s.denom)

		require.Equal(s.T(), user1BalanceBefore, user1BalanceAfter+delegation.Amount.Int64())

		require.Less(s.T(), user2BalanceAfter+100, user2BalanceBefore)
	})

	s.Run("multiple free transactions in same block", func() {
		user1BalanceBefore := QueryAccountBalance(s.T(), s.chain.(*cosmos.CosmosChain), s.user1.FormattedAddress(), s.denom)
		user2BalanceBefore := QueryAccountBalance(s.T(), s.chain.(*cosmos.CosmosChain), s.user2.FormattedAddress(), s.denom)

		// user1 submits a free-tx, user2 submits a free tx
		BroadcastTxs(s.T(), context.Background(), s.chain.(*cosmos.CosmosChain), []Tx{
			{
				User: s.user1,
				Msgs: []sdk.Msg{
					stakingtypes.NewMsgDelegate(
						sdk.AccAddress(s.user1.Address()),
						sdk.ValAddress(validators[0]),
						delegation,
					),
				},
			},
			{
				User: s.user2,
				Msgs: []sdk.Msg{
					stakingtypes.NewMsgDelegate(
						sdk.AccAddress(s.user2.Address()),
						sdk.ValAddress(validators[0]),
						delegation,
					),
				},
			},
		})

		// check balance after, user1 balance only diff by delegation
		user1BalanceAfter := QueryAccountBalance(s.T(), s.chain.(*cosmos.CosmosChain), s.user1.FormattedAddress(), s.denom)
		user2BalanceAfter := QueryAccountBalance(s.T(), s.chain.(*cosmos.CosmosChain), s.user2.FormattedAddress(), s.denom)

		require.Equal(s.T(), user1BalanceBefore, user1BalanceAfter+delegation.Amount.Int64())
		require.Equal(s.T(), user2BalanceBefore, user2BalanceAfter+delegation.Amount.Int64())
	})
}

func (s *IntegrationTestSuite) TestLanes() {
	validators := QueryValidators(s.T(), s.chain.(*cosmos.CosmosChain))
	require.True(s.T(), len(validators) > 0)

	delegation := sdk.NewCoin(s.denom, sdk.NewInt(100))

	params := QueryAuctionParams(s.T(), s.chain)

	s.Run("block with mev, free, and normal tx", func() {
		user2BalanceBefore := QueryAccountBalance(s.T(), s.chain.(*cosmos.CosmosChain), s.user2.FormattedAddress(), s.denom)

		// create free-tx, bid-tx, and normal-tx\
		bid, bundledTx := CreateAuctionBidMsg(
			s.T(),
			context.Background(),
			s.user1,
			s.chain.(*cosmos.CosmosChain),
			params.ReserveFee,
			[]Tx{
				{
					User: s.user1,
					Msgs: []sdk.Msg{
						&banktypes.MsgSend{
							FromAddress: s.user1.FormattedAddress(),
							ToAddress:   s.user1.FormattedAddress(),
							Amount:      sdk.NewCoins(sdk.NewCoin(s.denom, sdk.NewInt(100))),
						},
					},
					SequenceIncrement: 1,
				},
			},
		)

		height, err := s.chain.(*cosmos.CosmosChain).Height(context.Background())
		require.NoError(s.T(), err)

		txs := BroadcastTxs(s.T(), context.Background(), s.chain.(*cosmos.CosmosChain), []Tx{
			{
				User:   s.user1,
				Msgs:   []sdk.Msg{bid},
				Height: height + 1,
			},
			{
				User: s.user2,
				Msgs: []sdk.Msg{
					stakingtypes.NewMsgDelegate(
						sdk.AccAddress(s.user2.Address()),
						sdk.ValAddress(validators[0]),
						delegation,
					),
				},
				GasPrice: 10,
			},
			{
				User: s.user3,
				Msgs: []sdk.Msg{
					&banktypes.MsgSend{
						FromAddress: s.user3.FormattedAddress(),
						ToAddress:   s.user3.FormattedAddress(),
						Amount:      sdk.NewCoins(sdk.NewCoin(s.denom, sdk.NewInt(100))),
					},
				},
			},
		})

		// check block
		WaitForHeight(s.T(), s.chain.(*cosmos.CosmosChain), height+1)
		block := Block(s.T(), s.chain.(*cosmos.CosmosChain), int64(height+1))

		VerifyBlock(s.T(), block, 0, TxHash(txs[0]), append(bundledTx, txs[1:]...))

		// check user2 balance expect no fee deduction
		user2BalanceAfter := QueryAccountBalance(s.T(), s.chain.(*cosmos.CosmosChain), s.user2.FormattedAddress(), s.denom)
		require.Equal(s.T(), user2BalanceBefore, user2BalanceAfter+delegation.Amount.Int64())
	})

	s.Run("failing MEV transaction, free, and normal tx", func() {
		user2BalanceBefore := QueryAccountBalance(s.T(), s.chain.(*cosmos.CosmosChain), s.user2.FormattedAddress(), s.denom)
		user1Balance := QueryAccountBalance(s.T(), s.chain.(*cosmos.CosmosChain), s.user1.FormattedAddress(), s.denom)
		// create free-tx, bid-tx, and normal-tx\
		bid, _ := CreateAuctionBidMsg(
			s.T(),
			context.Background(),
			s.user1,
			s.chain.(*cosmos.CosmosChain),
			params.ReserveFee,
			[]Tx{
				{
					User: s.user1,
					Msgs: []sdk.Msg{
						&banktypes.MsgSend{
							FromAddress: s.user1.FormattedAddress(),
							ToAddress:   s.user1.FormattedAddress(),
							Amount:      sdk.NewCoins(sdk.NewCoin(s.denom, sdk.NewInt(user1Balance))),
						},
					},
					SequenceIncrement: 2,
				},
				{
					User: s.user1,
					Msgs: []sdk.Msg{
						&banktypes.MsgSend{
							FromAddress: s.user1.FormattedAddress(),
							ToAddress:   s.user1.FormattedAddress(),
							Amount:      sdk.NewCoins(sdk.NewCoin(s.denom, sdk.NewInt(user1Balance))),
						},
					},
					SequenceIncrement: 2,
				},
			},
		)

		height, err := s.chain.(*cosmos.CosmosChain).Height(context.Background())
		require.NoError(s.T(), err)

		txs := BroadcastTxs(s.T(), context.Background(), s.chain.(*cosmos.CosmosChain), []Tx{
			{
				User:       s.user1,
				Msgs:       []sdk.Msg{bid},
				Height:     height + 1,
				ExpectFail: true,
			},
			{
				User: s.user2,
				Msgs: []sdk.Msg{
					stakingtypes.NewMsgDelegate(
						sdk.AccAddress(s.user2.Address()),
						sdk.ValAddress(validators[0]),
						delegation,
					),
				},
				GasPrice: 10,
			},
			{
				User: s.user3,
				Msgs: []sdk.Msg{
					&banktypes.MsgSend{
						FromAddress: s.user3.FormattedAddress(),
						ToAddress:   s.user3.FormattedAddress(),
						Amount:      sdk.NewCoins(sdk.NewCoin(s.denom, sdk.NewInt(100))),
					},
				},
			},
		})

		// check block
		WaitForHeight(s.T(), s.chain.(*cosmos.CosmosChain), height+1)
		block := Block(s.T(), s.chain.(*cosmos.CosmosChain), int64(height+1))

		VerifyBlock(s.T(), block, 0, "", txs[1:])

		// check user2 balance expect no fee deduction
		user2BalanceAfter := QueryAccountBalance(s.T(), s.chain.(*cosmos.CosmosChain), s.user2.FormattedAddress(), s.denom)
		require.Equal(s.T(), user2BalanceBefore, user2BalanceAfter+delegation.Amount.Int64())
	})

	s.Run("MEV transaction that includes transactions from the free lane", func() {
		user2BalanceBefore := QueryAccountBalance(s.T(), s.chain.(*cosmos.CosmosChain), s.user2.FormattedAddress(), s.denom)

		delegateTx := Tx{
			User: s.user2,
			Msgs: []sdk.Msg{
				&stakingtypes.MsgDelegate{
					DelegatorAddress: s.user2.FormattedAddress(),
					ValidatorAddress: sdk.ValAddress(validators[0]).String(),
					Amount:           delegation,
				},
			},
			GasPrice: 10,
		}

		bid, bundledTx := CreateAuctionBidMsg(
			s.T(),
			context.Background(),
			s.user3,
			s.chain.(*cosmos.CosmosChain),
			params.ReserveFee,
			[]Tx{
				delegateTx,
				{
					User: s.user3,
					Msgs: []sdk.Msg{
						&banktypes.MsgSend{
							FromAddress: s.user3.FormattedAddress(),
							ToAddress:   s.user3.FormattedAddress(),
							Amount:      sdk.NewCoins(sdk.NewCoin(s.denom, sdk.NewInt(100))),
						},
					},
					SequenceIncrement: 1,
				},
			},
		)

		height, err := s.chain.(*cosmos.CosmosChain).Height(context.Background())
		require.NoError(s.T(), err)

		txs := BroadcastTxs(s.T(), context.Background(), s.chain.(*cosmos.CosmosChain), []Tx{
			{
				User:   s.user3,
				Msgs:   []sdk.Msg{bid},
				Height: height + 1,
			},
			delegateTx,
		})

		// query balance after, expect no fees paid
		user2BalanceAfter := QueryAccountBalance(s.T(), s.chain.(*cosmos.CosmosChain), s.user2.FormattedAddress(), s.denom)
		s.Require().Equal(user2BalanceBefore, user2BalanceAfter+delegation.Amount.Int64())

		// check block
		WaitForHeight(s.T(), s.chain.(*cosmos.CosmosChain), height+1)
		block := Block(s.T(), s.chain.(*cosmos.CosmosChain), int64(height+1))

		// verify
		VerifyBlock(s.T(), block, 0, TxHash(txs[0]), bundledTx)
	})

	s.Run("MEV transaction that includes transaction from free lane + other free lane txs + normal txs", func() {
		user2BalanceBefore := QueryAccountBalance(s.T(), s.chain.(*cosmos.CosmosChain), s.user2.FormattedAddress(), s.denom)

		// create free-txs signed by user2 / 3
		user2DelegateTx := Tx{
			User: s.user2,
			Msgs: []sdk.Msg{
				&stakingtypes.MsgDelegate{
					DelegatorAddress: s.user2.FormattedAddress(),
					ValidatorAddress: sdk.ValAddress(validators[0]).String(),
					Amount:           delegation,
				},
			},
			GasPrice: 10,
		}

		user3DelegateTx := Tx{
			User: s.user3,
			Msgs: []sdk.Msg{
				&stakingtypes.MsgDelegate{
					DelegatorAddress: s.user3.FormattedAddress(),
					ValidatorAddress: sdk.ValAddress(validators[0]).String(),
					Amount:           delegation,
				},
			},
			GasPrice:          10,
			SequenceIncrement: 1,
		}

		// create bid-tx w/ user3 DelegateTx

		bid, bundledTx := CreateAuctionBidMsg(
			s.T(),
			context.Background(),
			s.user3,
			s.chain.(*cosmos.CosmosChain),
			params.ReserveFee,
			[]Tx{
				user3DelegateTx,
				{
					User: s.user3,
					Msgs: []sdk.Msg{
						&banktypes.MsgSend{
							FromAddress: s.user3.FormattedAddress(),
							ToAddress:   s.user3.FormattedAddress(),
							Amount:      sdk.NewCoins(sdk.NewCoin(s.denom, sdk.NewInt(100))),
						},
					},
					SequenceIncrement: 2,
				},
			},
		)

		height, err := s.chain.(*cosmos.CosmosChain).Height(context.Background())
		require.NoError(s.T(), err)

		txs := BroadcastTxs(s.T(), context.Background(), s.chain.(*cosmos.CosmosChain), []Tx{
			{
				User:   s.user3,
				Msgs:   []sdk.Msg{bid},
				Height: height + 1,
			},
			// already included above
			user2DelegateTx,
		})

		// verify block
		WaitForHeight(s.T(), s.chain.(*cosmos.CosmosChain), height+1)
		block := Block(s.T(), s.chain.(*cosmos.CosmosChain), int64(height+1))
		VerifyBlock(s.T(), block, 0, TxHash(txs[0]), append(bundledTx, txs[1:]...))

		// check user2 balance expect no fee deduction
		user2BalanceAfter := QueryAccountBalance(s.T(), s.chain.(*cosmos.CosmosChain), s.user2.FormattedAddress(), s.denom)
		require.Equal(s.T(), user2BalanceBefore, user2BalanceAfter+delegation.Amount.Int64())
	})
}
