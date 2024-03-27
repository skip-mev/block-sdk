package e2e

import (
	"context"
	"math/rand"
	"time"

	"cosmossdk.io/math"
	rpctypes "github.com/cometbft/cometbft/rpc/core/types"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	interchaintest "github.com/strangelove-ventures/interchaintest/v7"
	"github.com/strangelove-ventures/interchaintest/v7/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v7/ibc"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/skip-mev/block-sdk/lanes/base"
	"github.com/skip-mev/block-sdk/lanes/free"
)

var (
	initBalance = sdk.NewInt(1000000000000)
)

type committedTx struct {
	tx  []byte
	res *rpctypes.ResultTx
}

// E2ETestSuite runs the Block SDK e2e test-suite against a given interchaintest specification
type E2ETestSuite struct {
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
	// fuzzusers
	fuzzusers []ibc.Wallet

	// overrides for key-ring configuration of the broadcaster
	broadcasterOverrides *KeyringOverride

	// broadcaster is the RPC interface to the ITS network
	bc *cosmos.Broadcaster
}

func NewE2ETestSuiteFromSpec(spec *interchaintest.ChainSpec) *E2ETestSuite {
	return &E2ETestSuite{
		spec:  spec,
		denom: "stake",
	}
}

func (s *E2ETestSuite) WithDenom(denom string) *E2ETestSuite {
	s.denom = denom

	// update the bech32 prefixes
	sdk.GetConfig().SetBech32PrefixForAccount(s.denom, s.denom+sdk.PrefixPublic)
	sdk.GetConfig().SetBech32PrefixForValidator(s.denom+sdk.PrefixValidator, s.denom+sdk.PrefixValidator+sdk.PrefixPublic)
	sdk.GetConfig().Seal()
	return s
}

func (s *E2ETestSuite) WithKeyringOptions(cdc codec.Codec, opts keyring.Option) {
	s.broadcasterOverrides = &KeyringOverride{
		cdc:            cdc,
		keyringOptions: opts,
	}
}

func (s *E2ETestSuite) SetupSuite() {
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

	for i := 0; i < 10; i++ {
		s.fuzzusers = append(s.fuzzusers, interchaintest.GetAndFundTestUsers(s.T(), ctx, s.T().Name(), initBalance, s.chain)[0])
	}

	// create the broadcaster
	s.T().Log("creating broadcaster")
	s.setupBroadcaster()
}

func (s *E2ETestSuite) TearDownSuite() {
	// close the interchain
	s.ic.Close()
}

func (s *E2ETestSuite) SetupSubTest() {
	// wait for 1 block height
	// query height
	height, err := s.chain.(*cosmos.CosmosChain).Height(context.Background())
	require.NoError(s.T(), err)
	WaitForHeight(s.T(), s.chain.(*cosmos.CosmosChain), height+3)
	s.T().Logf("reached height %d", height+2)
}

func (s *E2ETestSuite) TestQueryParams() {
	// query params
	params := QueryAuctionParams(s.T(), s.chain)

	// expect validate to pass
	require.NoError(s.T(), params.Validate())
}

func (s *E2ETestSuite) TestMempoolService() {
	resp, err := QueryMempool(s.T(), s.chain)
	s.Require().NoError(err)
	s.Require().Len(resp.Distribution, 3)
}

// TestValidBids tests the execution of various valid auction bids. There are a few
// invariants that are tested:
//
//  1. The order of transactions in a bundle is preserved when bids are valid.
//  2. All transactions execute as expected.
//  3. The balance of the escrow account should be updated correctly.
//  4. Top of block bids will be included in block proposals before other transactions
func (s *E2ETestSuite) TestValidBids() {
	params := QueryAuctionParams(s.T(), s.chain)
	escrowAddr := sdk.AccAddress(params.EscrowAccountAddress).String()

	s.Run("Valid Auction Bid", func() {
		// get escrow account balance before bid
		escrowAcctBalanceBeforeBid := QueryAccountBalance(s.T(), s.chain, escrowAddr, params.ReserveFee.Denom)

		// create bundle w/ a single tx
		// create message send tx
		tx := banktypes.NewMsgSend(s.user1.Address(), s.user2.Address(), sdk.NewCoins(sdk.NewCoin(s.denom, sdk.NewInt(100))))

		height, err := s.chain.(*cosmos.CosmosChain).Height(context.Background())
		require.NoError(s.T(), err)
		nextBlockHeight := height + 1

		// create the MsgAuctioBid
		bidAmt := params.ReserveFee
		bid, bundledTxs := s.CreateAuctionBidMsg(context.Background(), s.user1, s.chain.(*cosmos.CosmosChain), bidAmt, []Tx{
			{
				User: s.user1,
				Msgs: []sdk.Msg{
					tx,
				},
				SequenceIncrement: 1,
				Height:            nextBlockHeight,
			},
		})

		// broadcast + wait for the tx to be included in a block
		res := s.BroadcastTxs(context.Background(), s.chain.(*cosmos.CosmosChain), []Tx{
			{
				User:   s.user1,
				Msgs:   []sdk.Msg{bid},
				Height: nextBlockHeight,
			},
		})
		WaitForHeight(s.T(), s.chain.(*cosmos.CosmosChain), nextBlockHeight)

		// verify the block
		expectedBlock := [][]byte{
			res[0],
			bundledTxs[0],
		}
		VerifyBlockWithExpectedBlock(s.T(), s.chain.(*cosmos.CosmosChain), nextBlockHeight, expectedBlock)

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

		height, err := s.chain.(*cosmos.CosmosChain).Height(context.Background())
		require.NoError(s.T(), err)
		nextBlockHeight := height + 1

		// create the MsgAuctionBid
		bidAmt := params.ReserveFee
		bid, bundledTxs := s.CreateAuctionBidMsg(context.Background(), s.user1, s.chain.(*cosmos.CosmosChain), bidAmt, []Tx{
			{
				User:              s.user1,
				Msgs:              msgs[0:1],
				SequenceIncrement: 1,
				Height:            nextBlockHeight,
			},
		})

		// create the messages to be broadcast
		txs := make([]Tx, 0)
		txs = append(txs, Tx{
			User:   s.user1,
			Msgs:   []sdk.Msg{bid},
			Height: nextBlockHeight,
		})

		txs = append(txs, Tx{
			User: s.user2,
			Msgs: msgs[1:2],
		})

		// broadcast + wait for the tx to be included in a block
		res := s.BroadcastTxs(context.Background(), s.chain.(*cosmos.CosmosChain), txs)
		WaitForHeight(s.T(), s.chain.(*cosmos.CosmosChain), nextBlockHeight)

		// verify the block
		expectedBlock := [][]byte{
			res[0],
			bundledTxs[0],
			res[1],
		}
		VerifyBlockWithExpectedBlock(s.T(), s.chain.(*cosmos.CosmosChain), nextBlockHeight, expectedBlock)

		// ensure that escrow account has the correct balance
		escrowAcctBalanceAfterBid := QueryAccountBalance(s.T(), s.chain, escrowAddr, params.ReserveFee.Denom)
		expectedIncrement := escrowAddressIncrement(bidAmt.Amount, params.ProposerFee)
		require.Equal(s.T(), escrowAcctBalanceBeforeBid+expectedIncrement, escrowAcctBalanceAfterBid)
	})

	s.Run("iterative bidding from the same account", func() {
		// get escrow account balance before bid
		escrowAcctBalanceBeforeBid := QueryAccountBalance(s.T(), s.chain, escrowAddr, params.ReserveFee.Denom)

		height, err := s.chain.(*cosmos.CosmosChain).Height(context.Background())
		require.NoError(s.T(), err)
		nextBlockHeight := height + 2

		WaitForHeight(s.T(), s.chain.(*cosmos.CosmosChain), height+1)

		// create multi-tx valid bundle
		// bank-send msg
		txs := make([]Tx, 2)
		txs[0] = Tx{
			User:              s.user1,
			Msgs:              []sdk.Msg{banktypes.NewMsgSend(s.user1.Address(), s.user2.Address(), sdk.NewCoins(sdk.NewCoin(s.denom, sdk.NewInt(100))))},
			SequenceIncrement: 1,
			Height:            nextBlockHeight,
		}
		txs[1] = Tx{
			User:              s.user1,
			Msgs:              []sdk.Msg{banktypes.NewMsgSend(s.user1.Address(), s.user3.Address(), sdk.NewCoins(sdk.NewCoin(s.denom, sdk.NewInt(100))))},
			SequenceIncrement: 2,
			Height:            nextBlockHeight,
		}
		// create bundle
		bidAmt := params.ReserveFee
		bid, bundledTxs := s.CreateAuctionBidMsg(context.Background(), s.user1, s.chain.(*cosmos.CosmosChain), bidAmt, txs)
		// create 2 more bundle w same txs from same user
		bid2, _ := s.CreateAuctionBidMsg(context.Background(), s.user1, s.chain.(*cosmos.CosmosChain), bidAmt.Add(params.MinBidIncrement), txs)
		bid3, _ := s.CreateAuctionBidMsg(context.Background(), s.user1, s.chain.(*cosmos.CosmosChain), bidAmt.Add(params.MinBidIncrement).Add(params.MinBidIncrement), txs)

		// broadcast all bids
		broadcastedTxs := s.BroadcastTxs(context.Background(), s.chain.(*cosmos.CosmosChain), []Tx{
			{
				User:               s.user1,
				Msgs:               []sdk.Msg{bid},
				Height:             nextBlockHeight,
				SkipInclusionCheck: true,
			},
			{
				User:               s.user1,
				Msgs:               []sdk.Msg{bid2},
				Height:             nextBlockHeight,
				SkipInclusionCheck: true,
			},
			{
				User:   s.user1,
				Msgs:   []sdk.Msg{bid3},
				Height: nextBlockHeight,
			},
		})
		WaitForHeight(s.T(), s.chain.(*cosmos.CosmosChain), nextBlockHeight)

		// verify the block
		expectedBlock := [][]byte{
			broadcastedTxs[2],
			bundledTxs[0],
			bundledTxs[1],
		}
		VerifyBlockWithExpectedBlock(s.T(), s.chain.(*cosmos.CosmosChain), nextBlockHeight, expectedBlock)

		//  check escrow account balance
		escrowAcctBalanceAfterBid := QueryAccountBalance(s.T(), s.chain, escrowAddr, params.ReserveFee.Denom)
		expectedIncrement := escrowAddressIncrement(bidAmt.Add(params.MinBidIncrement.Add(params.MinBidIncrement)).Amount, params.ProposerFee)
		require.Equal(s.T(), escrowAcctBalanceBeforeBid+expectedIncrement, escrowAcctBalanceAfterBid)
	})

	s.Run("bid with a bundle with transactions that are already in the mempool", func() {
		// reset
		height, err := s.chain.(*cosmos.CosmosChain).Height(context.Background())
		require.NoError(s.T(), err)
		nextBlockHeight := height + 2

		// escrow account balance
		escrowAcctBalanceBeforeBid := QueryAccountBalance(s.T(), s.chain, escrowAddr, params.ReserveFee.Denom)

		// wait for the next height
		WaitForHeight(s.T(), s.chain.(*cosmos.CosmosChain), height+1)

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
		bid, bundledTxs := s.CreateAuctionBidMsg(context.Background(), s.user2, s.chain.(*cosmos.CosmosChain), bidAmt, txs)

		txsToBroadcast := []Tx{
			{
				User:   s.user2,
				Msgs:   []sdk.Msg{bid},
				Height: nextBlockHeight,
			},
			{
				User: s.user3,
				Msgs: []sdk.Msg{banktypes.NewMsgSend(s.user3.Address(), s.user1.Address(), sdk.NewCoins(sdk.NewCoin(s.denom, math.NewInt(100))))},
			},
		}

		// broadcast txs in the bundle to network + bundle + extra
		broadcastedTxs := s.BroadcastTxs(context.Background(), s.chain.(*cosmos.CosmosChain), append(txsToBroadcast, txs...))
		WaitForHeight(s.T(), s.chain.(*cosmos.CosmosChain), nextBlockHeight)

		// Verify the block
		expectedBlock := [][]byte{
			broadcastedTxs[0],
			bundledTxs[0],
			bundledTxs[1],
			broadcastedTxs[1],
		}
		VerifyBlockWithExpectedBlock(s.T(), s.chain.(*cosmos.CosmosChain), nextBlockHeight, expectedBlock)

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
func (s *E2ETestSuite) TestMultipleBids() {
	params := QueryAuctionParams(s.T(), s.chain)
	escrowAddr := sdk.AccAddress(params.EscrowAccountAddress).String()

	s.Run("Multiple bid transactions with second bid being smaller than min bid increment", func() {
		// get chain height
		height, err := s.chain.(*cosmos.CosmosChain).Height(context.Background())
		require.NoError(s.T(), err)
		nextBlockHeight := height + 2

		WaitForHeight(s.T(), s.chain.(*cosmos.CosmosChain), height+1)

		// escrow account balance
		escrowAcctBalanceBeforeBid := QueryAccountBalance(s.T(), s.chain, escrowAddr, params.ReserveFee.Denom)

		// create bid 1
		// bank-send msg
		bidAmt := params.ReserveFee
		bundleTx1 := Tx{
			User:              s.user1,
			Msgs:              []sdk.Msg{banktypes.NewMsgSend(s.user1.Address(), s.user2.Address(), sdk.NewCoins(sdk.NewCoin(s.denom, sdk.NewInt(100))))},
			SequenceIncrement: 1,
			Height:            nextBlockHeight,
		}
		// create bid1
		bid1, bundledTxs := s.CreateAuctionBidMsg(context.Background(), s.user1, s.chain.(*cosmos.CosmosChain), bidAmt, []Tx{bundleTx1})

		// create bid 2
		bundleTx2 := Tx{
			User:              s.user2,
			Msgs:              []sdk.Msg{banktypes.NewMsgSend(s.user2.Address(), s.user3.Address(), sdk.NewCoins(sdk.NewCoin(s.denom, sdk.NewInt(100))))},
			SequenceIncrement: 1,
			Height:            nextBlockHeight,
		}
		bid2, _ := s.CreateAuctionBidMsg(context.Background(), s.user2, s.chain.(*cosmos.CosmosChain), bidAmt, []Tx{bundleTx2})

		txs := []Tx{
			{
				User:   s.user1,
				Msgs:   []sdk.Msg{bid1},
				Height: nextBlockHeight,
			},
			{
				User:       s.user2,
				Msgs:       []sdk.Msg{bid2},
				Height:     nextBlockHeight,
				ExpectFail: true,
			},
		}

		// broadcast both bids (wait for the first to be committed)
		resp := s.BroadcastTxs(context.Background(), s.chain.(*cosmos.CosmosChain), txs)
		WaitForHeight(s.T(), s.chain.(*cosmos.CosmosChain), nextBlockHeight)

		// verify the block
		expectedBlock := [][]byte{
			resp[0],
			bundledTxs[0],
		}
		VerifyBlockWithExpectedBlock(s.T(), s.chain.(*cosmos.CosmosChain), nextBlockHeight, expectedBlock)

		// check escrow balance
		escrowAcctBalanceAfterBid := QueryAccountBalance(s.T(), s.chain, escrowAddr, params.ReserveFee.Denom)
		expectedIncrement := escrowAddressIncrement(bidAmt.Amount, params.ProposerFee)
		require.Equal(s.T(), escrowAcctBalanceBeforeBid+expectedIncrement, escrowAcctBalanceAfterBid)
	})

	s.Run("Multiple transactions from diff. account with increasing bids and first bid should fail in later block", func() {
		// get chain height
		height, err := s.chain.(*cosmos.CosmosChain).Height(context.Background())
		require.NoError(s.T(), err)
		nextBlockHeight := height + 2

		WaitForHeight(s.T(), s.chain.(*cosmos.CosmosChain), height+1)

		// escrow account balance
		escrowAcctBalanceBeforeBid := QueryAccountBalance(s.T(), s.chain, escrowAddr, params.ReserveFee.Denom)

		// create bid 1
		bidAmt := params.ReserveFee
		bundleTx1 := Tx{
			User: s.user3,
			Msgs: []sdk.Msg{banktypes.NewMsgSend(s.user3.Address(), s.user2.Address(), sdk.NewCoins(sdk.NewCoin(s.denom, sdk.NewInt(100))))},
		}
		bid1, _ := s.CreateAuctionBidMsg(context.Background(), s.user1, s.chain.(*cosmos.CosmosChain), bidAmt, []Tx{bundleTx1})

		// create bid2 w/ higher bid than bid1
		bidAmt = params.ReserveFee.Add(params.MinBidIncrement)
		bundleTx2 := Tx{
			User:              s.user2,
			Msgs:              []sdk.Msg{banktypes.NewMsgSend(s.user2.Address(), s.user1.Address(), sdk.NewCoins(sdk.NewCoin(s.denom, math.NewInt(100))))},
			Height:            nextBlockHeight,
			SequenceIncrement: 1,
		}
		bid2, bundledTxs := s.CreateAuctionBidMsg(context.Background(), s.user2, s.chain.(*cosmos.CosmosChain), bidAmt, []Tx{bundleTx2})

		txs := []Tx{
			{
				User:               s.user1,
				Msgs:               []sdk.Msg{bid1},
				Height:             nextBlockHeight,
				SkipInclusionCheck: true,
			},
			{
				User:   s.user2,
				Msgs:   []sdk.Msg{bid2},
				Height: nextBlockHeight,
			},
		}

		// broadcast both bids (wait for the second to be committed)
		resp := s.BroadcastTxs(context.Background(), s.chain.(*cosmos.CosmosChain), txs)
		WaitForHeight(s.T(), s.chain.(*cosmos.CosmosChain), nextBlockHeight)

		// verify the block
		expectedBlock := [][]byte{
			resp[1],
			bundledTxs[0],
		}
		VerifyBlockWithExpectedBlock(s.T(), s.chain.(*cosmos.CosmosChain), nextBlockHeight, expectedBlock)

		// Wait for next block and ensure other bid did not get included
		WaitForHeight(s.T(), s.chain.(*cosmos.CosmosChain), nextBlockHeight+1)
		VerifyBlockWithExpectedBlock(s.T(), s.chain.(*cosmos.CosmosChain), nextBlockHeight+1, nil)

		// check escrow balance
		escrowAcctBalanceAfterBid := QueryAccountBalance(s.T(), s.chain, escrowAddr, params.ReserveFee.Denom)
		expectedIncrement := escrowAddressIncrement(bidAmt.Amount, params.ProposerFee)
		require.Equal(s.T(), escrowAcctBalanceBeforeBid+expectedIncrement, escrowAcctBalanceAfterBid)
	})
}

func (s *E2ETestSuite) TestInvalidBids() {
	params := QueryAuctionParams(s.T(), s.chain)

	s.Run("searcher is attempting to submit a bundle that includes another bid tx", func() {
		height, err := s.chain.(*cosmos.CosmosChain).Height(context.Background())
		require.NoError(s.T(), err)

		// create bid tx
		msg := Tx{
			User:              s.user1,
			Msgs:              []sdk.Msg{banktypes.NewMsgSend(s.user1.Address(), s.user2.Address(), sdk.NewCoins(sdk.NewCoin(s.denom, sdk.NewInt(100))))},
			SequenceIncrement: 2,
			Height:            height + 1,
		}
		bidAmt := params.ReserveFee
		bid, _ := s.CreateAuctionBidMsg(context.Background(), s.user1, s.chain.(*cosmos.CosmosChain), bidAmt, []Tx{msg})

		// wrap bidTx in another tx
		wrappedBid, _ := s.CreateAuctionBidMsg(context.Background(), s.user1, s.chain.(*cosmos.CosmosChain), bidAmt, []Tx{
			{
				User:              s.user1,
				Msgs:              []sdk.Msg{bid},
				SequenceIncrement: 1,
				Height:            height + 1,
			},
		})

		// broadcast wrapped bid, and expect a failure
		s.BroadcastTxs(context.Background(), s.chain.(*cosmos.CosmosChain), []Tx{
			{
				User:       s.user1,
				Msgs:       []sdk.Msg{wrappedBid},
				Height:     height + 1,
				ExpectFail: true,
			},
		})
	})

	s.Run("Invalid bid that is attempting to bid more than their balance", func() {
		height, err := s.chain.(*cosmos.CosmosChain).Height(context.Background())
		require.NoError(s.T(), err)

		// create bid tx
		msg := Tx{
			User:              s.user1,
			Msgs:              []sdk.Msg{banktypes.NewMsgSend(s.user1.Address(), s.user2.Address(), sdk.NewCoins(sdk.NewCoin(s.denom, sdk.NewInt(100))))},
			SequenceIncrement: 1,
			Height:            height + 1,
		}
		bidAmt := sdk.NewCoin(s.denom, math.NewInt(1000000000000000000))
		bid, _ := s.CreateAuctionBidMsg(context.Background(), s.user1, s.chain.(*cosmos.CosmosChain), bidAmt, []Tx{msg})

		// broadcast wrapped bid, and expect a failure
		s.BroadcastTxs(context.Background(), s.chain.(*cosmos.CosmosChain), []Tx{
			{
				User:       s.user1,
				Msgs:       []sdk.Msg{bid},
				Height:     height + 1,
				ExpectFail: true,
			},
		})
		WaitForHeight(s.T(), s.chain.(*cosmos.CosmosChain), height+1)

		// verify the block
		VerifyBlockWithExpectedBlock(s.T(), s.chain.(*cosmos.CosmosChain), height+1, nil)
	})

	s.Run("Invalid bid that is attempting to front-run/sandwich", func() {
		height, err := s.chain.(*cosmos.CosmosChain).Height(context.Background())
		require.NoError(s.T(), err)
		nextBlockHeight := height + 1

		// create bid tx
		msg := Tx{
			User:              s.user1,
			Msgs:              []sdk.Msg{banktypes.NewMsgSend(s.user1.Address(), s.user2.Address(), sdk.NewCoins(sdk.NewCoin(s.denom, math.NewInt(100))))},
			SequenceIncrement: 1,
			Height:            nextBlockHeight,
		}
		msg2 := Tx{
			User: s.user2,
			Msgs: []sdk.Msg{banktypes.NewMsgSend(s.user2.Address(), s.user3.Address(), sdk.NewCoins(sdk.NewCoin(s.denom, sdk.NewInt(100))))},
		}
		msg3 := Tx{
			User:              s.user1,
			Msgs:              []sdk.Msg{banktypes.NewMsgSend(s.user1.Address(), s.user3.Address(), sdk.NewCoins(sdk.NewCoin(s.denom, sdk.NewInt(100))))},
			SequenceIncrement: 2,
			Height:            nextBlockHeight,
		}

		bidAmt := params.ReserveFee
		bid, _ := s.CreateAuctionBidMsg(context.Background(), s.user1, s.chain.(*cosmos.CosmosChain), bidAmt, []Tx{msg, msg2, msg3})

		// broadcast wrapped bid, and expect a failure
		s.BroadcastTxs(context.Background(), s.chain.(*cosmos.CosmosChain), []Tx{
			{
				User:       s.user1,
				Msgs:       []sdk.Msg{bid},
				Height:     height + 1,
				ExpectFail: true,
			},
		})
		WaitForHeight(s.T(), s.chain.(*cosmos.CosmosChain), height+1)

		// verify the block
		VerifyBlockWithExpectedBlock(s.T(), s.chain.(*cosmos.CosmosChain), height+1, nil)
	})

	s.Run("Invalid bid that includes an invalid bundle tx", func() {
		height, err := s.chain.(*cosmos.CosmosChain).Height(context.Background())
		require.NoError(s.T(), err)

		// create bid tx
		msg := Tx{
			User:              s.user1,
			Msgs:              []sdk.Msg{banktypes.NewMsgSend(s.user1.Address(), s.user2.Address(), sdk.NewCoins(sdk.NewCoin(s.denom, math.NewInt(100))))},
			SequenceIncrement: 2,
			Height:            height + 1,
		}
		bidAmt := params.ReserveFee
		bid, _ := s.CreateAuctionBidMsg(context.Background(), s.user1, s.chain.(*cosmos.CosmosChain), bidAmt, []Tx{msg})

		// broadcast wrapped bid, and expect a failure
		s.BroadcastTxs(context.Background(), s.chain.(*cosmos.CosmosChain), []Tx{
			{
				User:       s.user1,
				Msgs:       []sdk.Msg{bid},
				Height:     height + 1,
				ExpectFail: true,
			},
		})
		WaitForHeight(s.T(), s.chain.(*cosmos.CosmosChain), height+1)

		// verify the block
		VerifyBlockWithExpectedBlock(s.T(), s.chain.(*cosmos.CosmosChain), height+1, nil)
	})

	s.Run("Invalid auction bid with a bid smaller than the reserve fee", func() {
		height, err := s.chain.(*cosmos.CosmosChain).Height(context.Background())
		require.NoError(s.T(), err)

		// create bid tx
		msg := Tx{
			User:              s.user1,
			Msgs:              []sdk.Msg{banktypes.NewMsgSend(s.user1.Address(), s.user2.Address(), sdk.NewCoins(sdk.NewCoin(s.denom, sdk.NewInt(100))))},
			SequenceIncrement: 1,
			Height:            height + 1,
		}

		// create bid smaller than reserve
		bidAmt := sdk.NewCoin(s.denom, sdk.NewInt(0))
		bid, _ := s.CreateAuctionBidMsg(context.Background(), s.user1, s.chain.(*cosmos.CosmosChain), bidAmt, []Tx{msg})

		// broadcast wrapped bid, and expect a failure
		s.BroadcastTxs(context.Background(), s.chain.(*cosmos.CosmosChain), []Tx{
			{
				User:       s.user1,
				Msgs:       []sdk.Msg{bid},
				Height:     height + 1,
				ExpectFail: true,
			},
		})
		WaitForHeight(s.T(), s.chain.(*cosmos.CosmosChain), height+1)

		// verify the block
		VerifyBlockWithExpectedBlock(s.T(), s.chain.(*cosmos.CosmosChain), height+1, nil)
	})

	s.Run("Invalid auction bid with too many transactions in the bundle", func() {
		height, err := s.chain.(*cosmos.CosmosChain).Height(context.Background())
		require.NoError(s.T(), err)

		// create bid tx
		msgs := make([]Tx, 4)
		for i := range msgs {
			msgs[i] = Tx{
				User:              s.user1,
				Msgs:              []sdk.Msg{banktypes.NewMsgSend(s.user1.Address(), s.user2.Address(), sdk.NewCoins(sdk.NewCoin(s.denom, sdk.NewInt(100))))},
				SequenceIncrement: int64(i + 1),
				Height:            height + 1,
			}
		}

		// create bid smaller than reserve
		bidAmt := sdk.NewCoin(s.denom, sdk.NewInt(0))
		bid, _ := s.CreateAuctionBidMsg(context.Background(), s.user1, s.chain.(*cosmos.CosmosChain), bidAmt, msgs)

		// broadcast wrapped bid, and expect a failure
		s.BroadcastTxs(context.Background(), s.chain.(*cosmos.CosmosChain), []Tx{
			{
				User:       s.user1,
				Msgs:       []sdk.Msg{bid},
				Height:     height + 1,
				ExpectFail: true,
			},
		})
		WaitForHeight(s.T(), s.chain.(*cosmos.CosmosChain), height+1)

		// verify the block
		VerifyBlockWithExpectedBlock(s.T(), s.chain.(*cosmos.CosmosChain), height+1, nil)
	})

	s.Run("invalid auction bid that has an invalid timeout", func() {
		height, err := s.chain.(*cosmos.CosmosChain).Height(context.Background())
		require.NoError(s.T(), err)

		// create bid tx
		msg := Tx{
			User:              s.user1,
			Msgs:              []sdk.Msg{banktypes.NewMsgSend(s.user1.Address(), s.user2.Address(), sdk.NewCoins(sdk.NewCoin(s.denom, sdk.NewInt(100))))},
			SequenceIncrement: 1,
			Height:            height + 1,
		}

		bidAmt := sdk.NewCoin(s.denom, params.ReserveFee.Amount)
		bid, _ := s.CreateAuctionBidMsg(context.Background(), s.user1, s.chain.(*cosmos.CosmosChain), bidAmt, []Tx{msg})

		// broadcast wrapped bid, and expect a failure
		s.BroadcastTxs(context.Background(), s.chain.(*cosmos.CosmosChain), []Tx{
			{
				User:       s.user1,
				Msgs:       []sdk.Msg{bid},
				Height:     height + 2,
				ExpectFail: true,
			},
		})
		WaitForHeight(s.T(), s.chain.(*cosmos.CosmosChain), height+1)

		// verify the block
		VerifyBlockWithExpectedBlock(s.T(), s.chain.(*cosmos.CosmosChain), height+1, nil)
	})

	s.Run("Invalid bid that includes valid transactions that are in the mempool", func() {
		height, err := s.chain.(*cosmos.CosmosChain).Height(context.Background())
		require.NoError(s.T(), err)

		// create message send tx
		msgSend := banktypes.NewMsgSend(s.user2.Address(), s.user3.Address(), sdk.NewCoins(sdk.NewCoin(s.denom, math.NewInt(100))))

		// create the MsgAuctioBid (this should fail b.c same tx is repeated twice)
		bidAmt := params.ReserveFee
		bid, _ := s.CreateAuctionBidMsg(context.Background(), s.user1, s.chain.(*cosmos.CosmosChain), bidAmt, []Tx{
			{
				User: s.user2,
				Msgs: []sdk.Msg{msgSend},
			},
			{
				User: s.user2,
				Msgs: []sdk.Msg{msgSend},
			},
		})

		// broadcast + wait for the tx to be included in a block
		txs := s.BroadcastTxs(context.Background(), s.chain.(*cosmos.CosmosChain), []Tx{
			{
				User:       s.user1,
				Msgs:       []sdk.Msg{bid},
				Height:     height + 1,
				ExpectFail: true,
			},
			{
				User:   s.user2,
				Msgs:   []sdk.Msg{msgSend},
				Height: height + 1,
			},
		})

		// wait for next height
		WaitForHeight(s.T(), s.chain.(*cosmos.CosmosChain), height+1)

		// verify the block
		VerifyBlockWithExpectedBlock(s.T(), s.chain.(*cosmos.CosmosChain), height+1, [][]byte{txs[1]})
	})

	s.Run("searcher does not set the timeout height on their transactions", func() {
		height, err := s.chain.(*cosmos.CosmosChain).Height(context.Background())
		require.NoError(s.T(), err)

		// create bid tx
		msg := Tx{
			User:              s.user1,
			Msgs:              []sdk.Msg{banktypes.NewMsgSend(s.user1.Address(), s.user2.Address(), sdk.NewCoins(sdk.NewCoin(s.denom, math.NewInt(100))))},
			SequenceIncrement: 1,
		}

		bidAmt := sdk.NewCoin(s.denom, params.ReserveFee.Amount)
		bid, _ := s.CreateAuctionBidMsg(context.Background(), s.user1, s.chain.(*cosmos.CosmosChain), bidAmt, []Tx{msg})

		// broadcast wrapped bid, and expect a failure
		bidTx := Tx{
			User:       s.user1,
			Msgs:       []sdk.Msg{bid},
			ExpectFail: true,
			Height:     height + 1,
		}

		s.BroadcastTxs(context.Background(), s.chain.(*cosmos.CosmosChain), []Tx{bidTx})
		WaitForHeight(s.T(), s.chain.(*cosmos.CosmosChain), height+1)

		// verify the block
		VerifyBlockWithExpectedBlock(s.T(), s.chain.(*cosmos.CosmosChain), height+1, nil)
	})

	s.Run("timeout height on searchers txs in bundle do not match bid timeout", func() {
		height, err := s.chain.(*cosmos.CosmosChain).Height(context.Background())
		require.NoError(s.T(), err)

		// create bid tx
		msg := Tx{
			User:              s.user1,
			Msgs:              []sdk.Msg{banktypes.NewMsgSend(s.user1.Address(), s.user2.Address(), sdk.NewCoins(sdk.NewCoin(s.denom, math.NewInt(100))))},
			SequenceIncrement: 1,
			Height:            height + 3,
		}

		bidAmt := sdk.NewCoin(s.denom, params.ReserveFee.Amount)
		bid, _ := s.CreateAuctionBidMsg(context.Background(), s.user1, s.chain.(*cosmos.CosmosChain), bidAmt, []Tx{msg})

		// broadcast wrapped bid, and expect a failure
		bidTx := Tx{
			User:       s.user1,
			Msgs:       []sdk.Msg{bid},
			ExpectFail: true,
			Height:     height + 1,
		}

		s.BroadcastTxs(context.Background(), s.chain.(*cosmos.CosmosChain), []Tx{bidTx})
		WaitForHeight(s.T(), s.chain.(*cosmos.CosmosChain), height+1)

		// verify the block
		VerifyBlockWithExpectedBlock(s.T(), s.chain.(*cosmos.CosmosChain), height+1, nil)
	})
}

// TestFreeLane tests that the application correctly handles free lanes. There are a few invariants that are tested:
//
// 1. Transactions that qualify as free should not be deducted any fees.
// 2. Transactions that do not qualify as free should be deducted the correct fees.
func (s *E2ETestSuite) TestFreeLane() {
	validators := QueryValidators(s.T(), s.chain.(*cosmos.CosmosChain))
	require.True(s.T(), len(validators) > 0)

	delegation := sdk.NewCoin(s.denom, sdk.NewInt(100))

	s.Run("valid free lane transaction", func() {
		// query balance of account before tx submission
		balanceBefore := QueryAccountBalance(s.T(), s.chain.(*cosmos.CosmosChain), s.user1.FormattedAddress(), s.denom)

		height, err := s.chain.(*cosmos.CosmosChain).Height(context.Background())
		require.NoError(s.T(), err)

		// create a free tx (MsgDelegate), broadcast and wait for commit
		s.BroadcastTxs(context.Background(), s.chain.(*cosmos.CosmosChain), []Tx{
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

		// wait for next block
		WaitForHeight(s.T(), s.chain.(*cosmos.CosmosChain), height+1)

		// check balance of account
		balanceAfter := QueryAccountBalance(s.T(), s.chain.(*cosmos.CosmosChain), s.user1.FormattedAddress(), s.denom)
		require.Equal(s.T(), balanceBefore, balanceAfter+delegation.Amount.Int64())
	})

	s.Run("normal tx with free tx in same block", func() {
		user1BalanceBefore := QueryAccountBalance(s.T(), s.chain.(*cosmos.CosmosChain), s.user1.FormattedAddress(), s.denom)
		user2BalanceBefore := QueryAccountBalance(s.T(), s.chain.(*cosmos.CosmosChain), s.user2.FormattedAddress(), s.denom)

		height, err := s.chain.(*cosmos.CosmosChain).Height(context.Background())
		require.NoError(s.T(), err)

		// user1 submits a free-tx, user2 submits a normal tx
		s.BroadcastTxs(context.Background(), s.chain.(*cosmos.CosmosChain), []Tx{
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

		// wait for next block
		WaitForHeight(s.T(), s.chain.(*cosmos.CosmosChain), height+1)

		// check balance after, user1 balance only diff by delegation
		user1BalanceAfter := QueryAccountBalance(s.T(), s.chain.(*cosmos.CosmosChain), s.user1.FormattedAddress(), s.denom)
		user2BalanceAfter := QueryAccountBalance(s.T(), s.chain.(*cosmos.CosmosChain), s.user2.FormattedAddress(), s.denom)

		require.Equal(s.T(), user1BalanceBefore, user1BalanceAfter+delegation.Amount.Int64())

		require.Less(s.T(), user2BalanceAfter+100, user2BalanceBefore)
	})

	s.Run("multiple free transactions in same block", func() {
		user1BalanceBefore := QueryAccountBalance(s.T(), s.chain.(*cosmos.CosmosChain), s.user1.FormattedAddress(), s.denom)
		user2BalanceBefore := QueryAccountBalance(s.T(), s.chain.(*cosmos.CosmosChain), s.user2.FormattedAddress(), s.denom)

		height, err := s.chain.(*cosmos.CosmosChain).Height(context.Background())
		require.NoError(s.T(), err)

		// user1 submits a free-tx, user2 submits a free tx
		s.BroadcastTxs(context.Background(), s.chain.(*cosmos.CosmosChain), []Tx{
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

		// wait for next block
		WaitForHeight(s.T(), s.chain.(*cosmos.CosmosChain), height+1)

		// check balance after, user1 balance only diff by delegation
		user1BalanceAfter := QueryAccountBalance(s.T(), s.chain.(*cosmos.CosmosChain), s.user1.FormattedAddress(), s.denom)
		user2BalanceAfter := QueryAccountBalance(s.T(), s.chain.(*cosmos.CosmosChain), s.user2.FormattedAddress(), s.denom)

		require.Equal(s.T(), user1BalanceBefore, user1BalanceAfter+delegation.Amount.Int64())
		require.Equal(s.T(), user2BalanceBefore, user2BalanceAfter+delegation.Amount.Int64())
	})
}

func (s *E2ETestSuite) TestLanes() {
	validators := QueryValidators(s.T(), s.chain.(*cosmos.CosmosChain))
	require.True(s.T(), len(validators) > 0)

	delegation := sdk.NewCoin(s.denom, sdk.NewInt(100))

	params := QueryAuctionParams(s.T(), s.chain)

	s.Run("block with mev, free, and normal tx", func() {
		user2BalanceBefore := QueryAccountBalance(s.T(), s.chain.(*cosmos.CosmosChain), s.user2.FormattedAddress(), s.denom)

		height, err := s.chain.(*cosmos.CosmosChain).Height(context.Background())
		require.NoError(s.T(), err)
		nextBlockHeight := height + 2

		// create free-tx, bid-tx, and normal-tx\
		bid, bundledTx := s.CreateAuctionBidMsg(
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
					Height:            nextBlockHeight,
				},
			},
		)

		txsToBroadcast := []Tx{
			{
				User:   s.user1,
				Msgs:   []sdk.Msg{bid},
				Height: nextBlockHeight,
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
		}

		WaitForHeight(s.T(), s.chain.(*cosmos.CosmosChain), nextBlockHeight-1)

		txs := s.BroadcastTxs(context.Background(), s.chain.(*cosmos.CosmosChain), txsToBroadcast)
		WaitForHeight(s.T(), s.chain.(*cosmos.CosmosChain), nextBlockHeight)

		// verify the block
		VerifyBlockWithExpectedBlock(s.T(), s.chain.(*cosmos.CosmosChain), nextBlockHeight, [][]byte{txs[0], bundledTx[0], txs[1], txs[2]})

		// check user2 balance expect no fee deduction
		user2BalanceAfter := QueryAccountBalance(s.T(), s.chain.(*cosmos.CosmosChain), s.user2.FormattedAddress(), s.denom)
		require.Equal(s.T(), user2BalanceBefore, user2BalanceAfter+delegation.Amount.Int64())
	})

	s.Run("failing MEV transaction, free, and normal tx", func() {
		user2BalanceBefore := QueryAccountBalance(s.T(), s.chain.(*cosmos.CosmosChain), s.user2.FormattedAddress(), s.denom)
		user1Balance := QueryAccountBalance(s.T(), s.chain.(*cosmos.CosmosChain), s.user1.FormattedAddress(), s.denom)

		height, err := s.chain.(*cosmos.CosmosChain).Height(context.Background())
		require.NoError(s.T(), err)
		nextBlockHeight := height + 2

		// create free-tx, bid-tx, and normal-tx\
		bid, _ := s.CreateAuctionBidMsg(
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
					Height:            nextBlockHeight,
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
					Height:            nextBlockHeight,
				},
			},
		)

		WaitForHeight(s.T(), s.chain.(*cosmos.CosmosChain), nextBlockHeight-1)
		txs := s.BroadcastTxs(context.Background(), s.chain.(*cosmos.CosmosChain), []Tx{
			{
				User:       s.user1,
				Msgs:       []sdk.Msg{bid},
				Height:     nextBlockHeight,
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
		WaitForHeight(s.T(), s.chain.(*cosmos.CosmosChain), nextBlockHeight)

		// verify the block
		VerifyBlockWithExpectedBlock(s.T(), s.chain.(*cosmos.CosmosChain), nextBlockHeight, [][]byte{txs[1], txs[2]})

		// check user2 balance expect no fee deduction
		user2BalanceAfter := QueryAccountBalance(s.T(), s.chain.(*cosmos.CosmosChain), s.user2.FormattedAddress(), s.denom)
		require.Equal(s.T(), user2BalanceBefore, user2BalanceAfter+delegation.Amount.Int64())
	})

	s.Run("MEV transaction that includes transactions from the free lane", func() {
		user2BalanceBefore := QueryAccountBalance(s.T(), s.chain.(*cosmos.CosmosChain), s.user2.FormattedAddress(), s.denom)

		height, err := s.chain.(*cosmos.CosmosChain).Height(context.Background())
		require.NoError(s.T(), err)
		nextBlockHeight := height + 2

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

		bid, bundledTx := s.CreateAuctionBidMsg(
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
					Height:            nextBlockHeight,
				},
			},
		)

		WaitForHeight(s.T(), s.chain.(*cosmos.CosmosChain), nextBlockHeight-1)
		txs := s.BroadcastTxs(context.Background(), s.chain.(*cosmos.CosmosChain), []Tx{
			{
				User:   s.user3,
				Msgs:   []sdk.Msg{bid},
				Height: nextBlockHeight,
			},
		})
		WaitForHeight(s.T(), s.chain.(*cosmos.CosmosChain), nextBlockHeight)

		// verify the block
		VerifyBlockWithExpectedBlock(s.T(), s.chain.(*cosmos.CosmosChain), nextBlockHeight, [][]byte{txs[0], bundledTx[0], bundledTx[1]})

		// query balance after, expect no fees paid
		user2BalanceAfter := QueryAccountBalance(s.T(), s.chain.(*cosmos.CosmosChain), s.user2.FormattedAddress(), s.denom)
		s.Require().Equal(user2BalanceBefore, user2BalanceAfter+delegation.Amount.Int64())
	})

	s.Run("MEV transaction that includes transaction from free lane + other free lane txs + normal txs", func() {
		user2BalanceBefore := QueryAccountBalance(s.T(), s.chain.(*cosmos.CosmosChain), s.user2.FormattedAddress(), s.denom)

		height, err := s.chain.(*cosmos.CosmosChain).Height(context.Background())
		require.NoError(s.T(), err)
		nextBlockHeight := height + 2

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
			Height:            nextBlockHeight,
		}

		// create bid-tx w/ user3 DelegateTx

		bid, bundledTx := s.CreateAuctionBidMsg(
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
					Height:            nextBlockHeight,
				},
			},
		)

		WaitForHeight(s.T(), s.chain.(*cosmos.CosmosChain), nextBlockHeight-1)
		txs := s.BroadcastTxs(context.Background(), s.chain.(*cosmos.CosmosChain), []Tx{
			{
				User:   s.user3,
				Msgs:   []sdk.Msg{bid},
				Height: nextBlockHeight,
			},
			// already included above
			user2DelegateTx,
		})
		WaitForHeight(s.T(), s.chain.(*cosmos.CosmosChain), nextBlockHeight)

		// verify block
		VerifyBlockWithExpectedBlock(s.T(), s.chain.(*cosmos.CosmosChain), nextBlockHeight, [][]byte{txs[0], bundledTx[0], bundledTx[1], txs[1]})

		// check user2 balance expect no fee deduction
		user2BalanceAfter := QueryAccountBalance(s.T(), s.chain.(*cosmos.CosmosChain), s.user2.FormattedAddress(), s.denom)
		require.Equal(s.T(), user2BalanceBefore, user2BalanceAfter+delegation.Amount.Int64())
	})
}

func (s *E2ETestSuite) TestNetwork() {
	amountToTest := time.NewTicker(time.Second * 30)
	defer amountToTest.Stop()

	numTxs := 10
	sendAmount := sdk.NewCoins(sdk.NewCoin(s.denom, math.NewInt(100)))
	delegation := sdk.NewCoin(s.denom, math.NewInt(100))
	validators := QueryValidators(s.T(), s.chain.(*cosmos.CosmosChain))

	s.Run("can produce blocks with only default transactions", func() {
		for {
			select {
			case <-amountToTest.C:
				return
			default:
				height, err := s.chain.(*cosmos.CosmosChain).Height(context.Background())
				s.NoError(err)
				WaitForHeight(s.T(), s.chain.(*cosmos.CosmosChain), height+1)

				s.T().Logf("height: %d", height+1)

				for i := 0; i < numTxs; i++ {
					for _, user := range s.fuzzusers {
						fee := rand.Int63n(100000)
						sequenceOffset := int64(i)

						normalTx := s.CreateDummyNormalTx(user, s.user1, sendAmount, sequenceOffset, fee)
						s.BroadcastTxs(context.Background(), s.chain.(*cosmos.CosmosChain), []Tx{normalTx})
					}
				}

				resp, err := QueryMempool(s.T(), s.chain)
				s.NoError(err)
				s.Require().True(resp.Distribution[base.LaneName] > 0)
			}
		}
	})

	s.Run("can produce blocks with only free transactions", func() {
		for {
			select {
			case <-amountToTest.C:
				return
			default:
				height, err := s.chain.(*cosmos.CosmosChain).Height(context.Background())
				s.NoError(err)
				WaitForHeight(s.T(), s.chain.(*cosmos.CosmosChain), height+1)

				s.T().Logf("height: %d", height+1)

				for i := 0; i < numTxs; i++ {
					for _, user := range s.fuzzusers {
						sequenceOffset := int64(i)

						freeTx := s.CreateDummyFreeTx(user, validators[0], delegation, sequenceOffset)
						s.BroadcastTxs(context.Background(), s.chain.(*cosmos.CosmosChain), []Tx{freeTx})
					}
				}

				resp, err := QueryMempool(s.T(), s.chain)
				s.NoError(err)
				s.Require().True(resp.Distribution[free.LaneName] > 0)
			}
		}
	})

	s.Run("can produce blocks with only MEV transactions", func() {
		for {
			select {
			case <-amountToTest.C:
				return
			default:
				height, err := s.chain.(*cosmos.CosmosChain).Height(context.Background())
				s.NoError(err)
				WaitForHeight(s.T(), s.chain.(*cosmos.CosmosChain), height+1)

				s.T().Logf("height: %d", height+1)

				for i := 0; i < numTxs; i++ {
					for _, user := range s.fuzzusers {
						bid := rand.Int63n(1000000)
						bidAmount := sdk.NewCoin(s.denom, math.NewInt(bid))

						mevTx := s.CreateDummyAuctionBidTx(
							height+2,
							user,
							bidAmount,
						)
						s.BroadcastTxs(context.Background(), s.chain.(*cosmos.CosmosChain), []Tx{mevTx})
					}
				}
			}
		}
	})

	amountToTest.Reset(1 * time.Minute)
	s.Run("can produce blocks with all types of transactions", func() {
		for {
			select {
			case <-amountToTest.C:
				return
			default:
				height, err := s.chain.(*cosmos.CosmosChain).Height(context.Background())
				s.NoError(err)
				WaitForHeight(s.T(), s.chain.(*cosmos.CosmosChain), height+1)

				s.T().Logf("height: %d", height+1)

				txs := []Tx{}

				for i := 0; i < numTxs; i++ {
					for _, user := range s.fuzzusers[0:3] {
						bid := rand.Int63n(1000000)
						bidAmount := sdk.NewCoin(s.denom, math.NewInt(bid))

						bidTx := s.CreateDummyAuctionBidTx(
							height+2,
							user,
							bidAmount,
						)
						txs = append(txs, bidTx)
					}
				}

				for i := 0; i < numTxs; i++ {
					for _, user := range s.fuzzusers[3:6] {
						sequenceOffset := int64(i)

						freeTx := s.CreateDummyFreeTx(user, validators[0], delegation, sequenceOffset)
						txs = append(txs, freeTx)

					}
				}

				for i := 0; i < numTxs; i++ {
					for _, user := range s.fuzzusers[6:10] {
						fee := rand.Int63n(100000)
						sequenceOffset := int64(i)
						normalTx := s.CreateDummyNormalTx(user, s.user1, sendAmount, sequenceOffset, fee)
						txs = append(txs, normalTx)
					}
				}

				s.BroadcastTxs(context.Background(), s.chain.(*cosmos.CosmosChain), txs)
			}
		}
	})

	// Wait for 1 minute for the network to stabilize
	amountToTest.Reset(1 * time.Minute)
	s.Run("can produce empty blocks", func() {
		for {
			select {
			case <-amountToTest.C:
				return
			default:
				height, err := s.chain.(*cosmos.CosmosChain).Height(context.Background())
				s.NoError(err)
				WaitForHeight(s.T(), s.chain.(*cosmos.CosmosChain), height+1)

				s.T().Logf("height: %d", height+1)
			}
		}
	})
}
