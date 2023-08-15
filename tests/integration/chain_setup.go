package integration

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"strings"
	"testing"
	"time"

	rpctypes "github.com/cometbft/cometbft/rpc/core/types"
	comettypes "github.com/cometbft/cometbft/types"
	"github.com/cosmos/cosmos-sdk/client/tx"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	buildertypes "github.com/skip-mev/pob/x/builder/types"
	interchaintest "github.com/strangelove-ventures/interchaintest/v7"
	"github.com/strangelove-ventures/interchaintest/v7/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v7/ibc"
	"github.com/strangelove-ventures/interchaintest/v7/testutil"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// ChainBuilderFromChainSpec creates an interchaintest chain builder factory given a ChainSpec
// and returns the associated chain
func ChainBuilderFromChainSpec(t *testing.T, spec *interchaintest.ChainSpec) ibc.Chain {
	// require that NumFullNodes == NumValidators == 4
	require.Equal(t, *spec.NumValidators, 4)

	cf := interchaintest.NewBuiltinChainFactory(zaptest.NewLogger(t), []*interchaintest.ChainSpec{spec})

	chains, err := cf.Chains(t.Name())
	require.NoError(t, err)

	require.Len(t, chains, 1)
	chain := chains[0]

	_, ok := chain.(*cosmos.CosmosChain)
	require.True(t, ok)

	return chain
}

// BuildPOBInterchain creates a new Interchain testing env with the configured POB CosmosChain
func BuildPOBInterchain(t *testing.T, ctx context.Context, chain ibc.Chain) *interchaintest.Interchain {
	ic := interchaintest.NewInterchain()
	ic.AddChain(chain)

	// create docker network
	client, networkID := interchaintest.DockerSetup(t)

	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	// build the interchain
	err := ic.Build(ctx, nil, interchaintest.InterchainBuildOptions{
		SkipPathCreation: true,
		Client:           client,
		NetworkID:        networkID,
		TestName:         t.Name(),
	})
	require.NoError(t, err)

	return ic
}

// CreateTx creates a new transaction to be signed by the given user, including a provided set of messages
func CreateTx(t *testing.T, ctx context.Context, chain *cosmos.CosmosChain, user cosmos.User, seqIncrement, height uint64, GasPrice int64, msgs ...sdk.Msg) []byte {
	// create a broadcaster
	broadcaster := cosmos.NewBroadcaster(t, chain)

	// create tx factory + Client Context
	txf, err := broadcaster.GetFactory(ctx, user)
	require.NoError(t, err)

	cc, err := broadcaster.GetClientContext(ctx, user)
	require.NoError(t, err)

	txf, err = txf.Prepare(cc)
	require.NoError(t, err)

	// set timeout height
	if height != 0 {
		txf = txf.WithTimeoutHeight(height)
	}

	// get gas for tx
	_, gas, err := tx.CalculateGas(cc, txf, msgs...)
	require.NoError(t, err)
	txf.WithGas(gas)

	// update sequence number
	txf = txf.WithSequence(txf.Sequence() + seqIncrement)
	txf = txf.WithGasPrices(sdk.NewDecCoins(sdk.NewDecCoin(chain.Config().Denom, sdk.NewInt(GasPrice))).String())

	// sign the tx
	txBuilder, err := txf.BuildUnsignedTx(msgs...)
	require.NoError(t, err)

	require.NoError(t, tx.Sign(txf, cc.GetFromName(), txBuilder, true))

	// encode and return
	bz, err := cc.TxConfig.TxEncoder()(txBuilder.GetTx())
	require.NoError(t, err)
	return bz
}

// SimulateTx simulates the provided messages, and checks whether the provided failure condition is met
func SimulateTx(t *testing.T, ctx context.Context, chain *cosmos.CosmosChain, user cosmos.User, height uint64, expectFail bool, msgs ...sdk.Msg) {
	// create a broadcaster
	broadcaster := cosmos.NewBroadcaster(t, chain)

	// create tx factory + Client Context
	txf, err := broadcaster.GetFactory(ctx, user)
	require.NoError(t, err)

	cc, err := broadcaster.GetClientContext(ctx, user)
	require.NoError(t, err)

	txf, err = txf.Prepare(cc)
	require.NoError(t, err)

	// set timeout height
	if height != 0 {
		txf = txf.WithTimeoutHeight(height)
	}

	// get gas for tx
	_, _, err = tx.CalculateGas(cc, txf, msgs...)
	require.Equal(t, err != nil, expectFail)
}

type Tx struct {
	User               cosmos.User
	Msgs               []sdk.Msg
	GasPrice           int64
	SequenceIncrement  uint64
	Height             uint64
	SkipInclusionCheck bool
	ExpectFail         bool
}

// CreateAuctionBidMsg creates a new AuctionBid tx signed by the given user, the order of txs in the MsgAuctionBid will be determined by the contents + order of the MessageForUsers
func CreateAuctionBidMsg(t *testing.T, ctx context.Context, searcher cosmos.User, chain *cosmos.CosmosChain, bid sdk.Coin, txsPerUser []Tx) (*buildertypes.MsgAuctionBid, [][]byte) {
	// for each MessagesForUser get the signed bytes
	txs := make([][]byte, len(txsPerUser))
	for i, tx := range txsPerUser {
		txs[i] = CreateTx(t, ctx, chain, tx.User, tx.SequenceIncrement, tx.Height, tx.GasPrice, tx.Msgs...)
	}

	bech32SearcherAddress := searcher.FormattedAddress()
	accAddr, err := sdk.AccAddressFromBech32(bech32SearcherAddress)
	require.NoError(t, err)

	// create a message auction bid
	return buildertypes.NewMsgAuctionBid(
		accAddr,
		bid,
		txs,
	), txs
}

// BroadcastTxs broadcasts the given messages for each user. This function returns the broadcasted txs. If a message
// is not expected to be included in a block, set SkipInclusionCheck to true and the method
// will not block on the tx's inclusion in a block, otherwise this method will block on the tx's inclusion
func BroadcastTxs(t *testing.T, ctx context.Context, chain *cosmos.CosmosChain, msgsPerUser []Tx) [][]byte {
	txs := make([][]byte, len(msgsPerUser))

	for i, msg := range msgsPerUser {
		txs[i] = CreateTx(t, ctx, chain, msg.User, msg.SequenceIncrement, msg.Height, msg.GasPrice, msg.Msgs...)
	}

	// broadcast each tx
	require.True(t, len(chain.Nodes()) > 0)
	client := chain.Nodes()[0].Client

	for i, tx := range txs {
		// broadcast tx
		_, err := client.BroadcastTxSync(ctx, tx)

		// check execution was successful
		if !msgsPerUser[i].ExpectFail {
			require.NoError(t, err)
		} else {
			require.Error(t, err)
		}

	}

	// block on all txs being included in block
	eg := errgroup.Group{}
	for i, tx := range txs {
		// if we don't expect this tx to be included.. skip it
		if msgsPerUser[i].SkipInclusionCheck || msgsPerUser[i].ExpectFail {
			continue
		}

		tx := tx // pin
		eg.Go(func() error {
			return testutil.WaitForCondition(4*time.Second, 500*time.Millisecond, func() (bool, error) {
				res, err := client.Tx(context.Background(), comettypes.Tx(tx).Hash(), false)

				if err != nil || res.TxResult.Code != uint32(0) {
					return false, nil
				}
				return true, nil
			})
		})
	}

	require.NoError(t, eg.Wait())

	return txs
}

// QueryBuilderParams queries the x/builder module's params
func QueryBuilderParams(t *testing.T, chain ibc.Chain) buildertypes.Params {
	// cast chain to cosmos-chain
	cosmosChain, ok := chain.(*cosmos.CosmosChain)
	require.True(t, ok)
	// get nodes
	nodes := cosmosChain.Nodes()
	require.True(t, len(nodes) > 0)
	// make params query to first node
	resp, _, err := nodes[0].ExecQuery(context.Background(), "builder", "params")
	require.NoError(t, err)

	// unmarshal params
	var params buildertypes.Params
	err = json.Unmarshal(resp, &params)
	require.NoError(t, err)
	return params
}

// QueryValidators queries for all of the network's validators
func QueryValidators(t *testing.T, chain *cosmos.CosmosChain) []sdk.ValAddress {
	// get grpc client of the node
	grpcAddr := chain.GetHostGRPCAddress()
	cc, err := grpc.Dial(grpcAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))

	require.NoError(t, err)

	client := stakingtypes.NewQueryClient(cc)

	// query validators
	resp, err := client.Validators(context.Background(), &stakingtypes.QueryValidatorsRequest{})
	require.NoError(t, err)

	addrs := make([]sdk.ValAddress, len(resp.Validators))

	// unmarshal validators
	for i, val := range resp.Validators {
		addrBz, err := sdk.GetFromBech32(val.OperatorAddress, chain.Config().Bech32Prefix+sdk.PrefixValidator+sdk.PrefixOperator)
		require.NoError(t, err)

		addrs[i] = sdk.ValAddress(addrBz)
	}
	return addrs
}

// QueryAccountBalance queries a given account's balance on the chain
func QueryAccountBalance(t *testing.T, chain ibc.Chain, address, denom string) int64 {
	// cast the chain to a cosmos-chain
	cosmosChain, ok := chain.(*cosmos.CosmosChain)
	require.True(t, ok)
	// get nodes
	balance, err := cosmosChain.GetBalance(context.Background(), address, denom)
	require.NoError(t, err)
	return balance
}

// QueryAccountSequence
func QueryAccountSequence(t *testing.T, chain *cosmos.CosmosChain, address string) uint64 {
	// get nodes
	nodes := chain.Nodes()
	require.True(t, len(nodes) > 0)

	resp, _, err := nodes[0].ExecQuery(context.Background(), "auth", "account", address)
	require.NoError(t, err)
	// unmarshal json response
	var accResp codectypes.Any
	require.NoError(t, json.Unmarshal(resp, &accResp))

	// unmarshal into baseAccount
	var acc authtypes.BaseAccount
	require.NoError(t, acc.Unmarshal(accResp.Value))

	return acc.GetSequence()
}

// Block returns the block at the given height
func Block(t *testing.T, chain *cosmos.CosmosChain, height int64) *rpctypes.ResultBlock {
	// get nodes
	nodes := chain.Nodes()
	require.True(t, len(nodes) > 0)

	client := nodes[0].Client

	resp, err := client.Block(context.Background(), &height)
	require.NoError(t, err)

	return resp
}

// WaitForHeight waits for the chain to reach the given height
func WaitForHeight(t *testing.T, chain *cosmos.CosmosChain, height uint64) {
	// wait for next height
	err := testutil.WaitForCondition(30*time.Second, time.Second, func() (bool, error) {
		pollHeight, err := chain.Height(context.Background())
		if err != nil {
			return false, err
		}
		return pollHeight == height, nil
	})
	require.NoError(t, err)
}

// VerifyBlock takes a Block and verifies that it contains the given bid at the 0-th index, and the bundled txs immediately after
func VerifyBlock(t *testing.T, block *rpctypes.ResultBlock, offset int, bidTxHash string, txs [][]byte) {
	// verify the block
	if bidTxHash != "" {
		require.Equal(t, bidTxHash, TxHash(block.Block.Data.Txs[offset]))
		offset += 1
	}

	// verify the txs in sequence
	for i, tx := range txs {
		require.Equal(t, TxHash(tx), TxHash(block.Block.Data.Txs[i+offset]))
	}
}

func TxHash(tx []byte) string {
	return strings.ToUpper(hex.EncodeToString(comettypes.Tx(tx).Hash()))
}
