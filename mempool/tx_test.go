package mempool_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	pobcodec "github.com/skip-mev/pob/codec"
	"github.com/skip-mev/pob/mempool"
	auctiontypes "github.com/skip-mev/pob/x/auction/types"
	"github.com/stretchr/testify/require"
)

func TestGetMsgAuctionBidFromTx_Valid(t *testing.T) {
	encCfg := pobcodec.CreateEncodingConfig()

	txBuilder := encCfg.TxConfig.NewTxBuilder()
	txBuilder.SetMsgs(&auctiontypes.MsgAuctionBid{})

	msg, err := mempool.GetMsgAuctionBidFromTx(txBuilder.GetTx())
	require.NoError(t, err)
	require.NotNil(t, msg)
}

func TestGetMsgAuctionBidFromTx_MultiMsgBid(t *testing.T) {
	encCfg := pobcodec.CreateEncodingConfig()

	txBuilder := encCfg.TxConfig.NewTxBuilder()
	txBuilder.SetMsgs(
		&auctiontypes.MsgAuctionBid{},
		&auctiontypes.MsgAuctionBid{},
		&banktypes.MsgSend{},
	)

	msg, err := mempool.GetMsgAuctionBidFromTx(txBuilder.GetTx())
	require.Error(t, err)
	require.Nil(t, msg)
}

func TestGetMsgAuctionBidFromTx_NoBid(t *testing.T) {
	encCfg := pobcodec.CreateEncodingConfig()

	txBuilder := encCfg.TxConfig.NewTxBuilder()
	txBuilder.SetMsgs(&banktypes.MsgSend{})

	msg, err := mempool.GetMsgAuctionBidFromTx(txBuilder.GetTx())
	require.NoError(t, err)
	require.Nil(t, msg)
}

func TestGetUnwrappedTx(t *testing.T) {
	encCfg := pobcodec.CreateEncodingConfig()

	txBuilder := encCfg.TxConfig.NewTxBuilder()
	txBuilder.SetMsgs(&auctiontypes.MsgAuctionBid{})
	tx := txBuilder.GetTx()

	bid := sdk.NewCoins(sdk.NewCoin("foo", sdk.NewInt(1000000)))
	wrappedTx := mempool.NewWrappedBidTx(tx, bid)
	unWrappedTx := mempool.UnwrapBidTx(wrappedTx)

	unwrappedBz, err := encCfg.TxConfig.TxEncoder()(unWrappedTx)
	require.NoError(t, err)

	txBz, err := encCfg.TxConfig.TxEncoder()(tx)
	require.NoError(t, err)
	require.Equal(t, txBz, unwrappedBz)
}
