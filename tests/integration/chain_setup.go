package integration

import (
	"context"
	"testing"

	interchaintest "github.com/strangelove-ventures/interchaintest/v7"
	"github.com/strangelove-ventures/interchaintest/v7/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v7/ibc"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	buildertypes "github.com/skip-mev/pob/x/builder/types"
)

// ChainBuilderFromChainSpec creates an interchaintest chain builder factory given a ChainSpec
// and returns the associated chain
func ChainBuilderFromChainSpec(t *testing.T, spec *interchaintest.ChainSpec) ibc.Chain {
	// require that NumFullNodes == NumValidators == 4
	require.Equal(t, spec.NumFullNodes, spec.NumValidators)
	require.Equal(t, spec.NumFullNodes, 4)

	cf := interchaintest.NewBuiltinChainFactory(zaptest.NewLogger(t), []*interchaintest.ChainSpec{spec})

	chains, err := cf.Chains(t.Name())
	require.NoError(t, err)

	require.Len(t, chains, 1)
	
	return chains[0]
}

// BuildPOBInterchain creates a new Interchain testing env with the configured POB CosmosChain
func BuildPOBInterchain(t *testing.T, ctx context.Context, chain ibc.Chain) *interchaintest.Interchain {
	ic := interchaintest.NewInterchain()
	ic.AddChain(chain)

	// create docker network
	client, networkID := interchaintest.DockerSetup(t)

	// build the interchain
	err := ic.Build(ctx, nil, interchaintest.InterchainBuildOptions{
		SkipPathCreation: true,
		Client: 		 client,
		NetworkID: 		 networkID,
		TestName: 		 t.Name(),
	})
	require.NoError(t, err)

	return ic
}

// CreateTx creates a new transaction to be signed by the given user, including a provided set of messages
func CreateTx(t *testing.T, ctx context.Context, chain *cosmos.CosmosChain, user cosmos.User, msgs ...sdk.Msg) []byte {
	// create a broadcaster
	broadcaster := cosmos.NewBroadcaster(t, chain)

	// create tx factory + Client Context
	txf, err := broadcaster.GetFactory(ctx, user)
	require.NoError(t, err)

	cc, err := broadcaster.GetClientContext(ctx, user)
	require.NoError(t, err)

	txf, err = txf.Prepare(cc)
	require.NoError(t, err)
	
	// get gas for tx
	_, gas, err := tx.CalculateGas(cc, txf, msgs...)
	require.NoError(t, err)
	txf.WithGas(gas)

	// sign the tx
	txBuilder, err := txf.BuildUnsignedTx(msgs...)
	require.NoError(t, err)

	require.NoError(t, tx.Sign(txf, cc.GetFromName(), txBuilder, true))

	// encode and return
	bz, err := cc.TxConfig.TxEncoder()(txBuilder.GetTx())
	require.NoError(t, err)
	return bz
}

type MessagesForUser struct {
	User cosmos.User
	Msgs []sdk.Msg
}

// CreateAuctionBidMsg creates a new AuctionBid tx signed by the given user, the order of txs in the MsgAuctionBid will be determined by the contents + order of the MessageForUsers
func CreateAuctionBidMsg(t *testing.T, ctx context.Context, searcher cosmos.User, chain *cosmos.CosmosChain, bid sdk.Coin, users []MessagesForUser) *buildertypes.MsgAuctionBid {
	// for each MessagesForUser get the signed bytes
	txs := make([][]byte, len(users))
	for i, user := range users {
		txs[i] = CreateTx(t, ctx, chain, user.User, user.Msgs...)
	}

	bech32SearcherAddress := searcher.FormattedAddress()
	accAddr, err := sdk.AccAddressFromBech32(bech32SearcherAddress)
	require.NoError(t, err)

	// create a message auction bid
	return buildertypes.NewMsgAuctionBid(
		accAddr,
		bid,
		txs,
	)
}

// BroadcastMsg broadcasts the given messages as a tx signed by the given sender, it blocks until a response from the chain is received
// and fails if a timeout occurs
func BroadcastMsg(t *testing.T, ctx context.Context, sender cosmos.User, chain *cosmos.CosmosChain, msgs ...sdk.Msg) (sdk.TxResponse) {
	// create a broadcaster
	broadcaster := cosmos.NewBroadcaster(t, chain)

	resp, err := cosmos.BroadcastTx(ctx, broadcaster, sender, msgs...)
	require.NoError(t, err)
	return resp
}
