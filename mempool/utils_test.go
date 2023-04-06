package mempool_test

import (
	"testing"

	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	pobcodec "github.com/skip-mev/pob/codec"
	"github.com/skip-mev/pob/mempool"
	buildertypes "github.com/skip-mev/pob/x/builder/types"
	"github.com/stretchr/testify/require"
)

func TestGetMsgAuctionBidFromTx_Valid(t *testing.T) {
	encCfg := pobcodec.CreateEncodingConfig()

	txBuilder := encCfg.TxConfig.NewTxBuilder()
	txBuilder.SetMsgs(&buildertypes.MsgAuctionBid{})

	msg, err := mempool.GetMsgAuctionBidFromTx(txBuilder.GetTx())
	require.NoError(t, err)
	require.NotNil(t, msg)
}

func TestGetMsgAuctionBidFromTx_MultiMsgBid(t *testing.T) {
	encCfg := pobcodec.CreateEncodingConfig()

	txBuilder := encCfg.TxConfig.NewTxBuilder()
	txBuilder.SetMsgs(
		&buildertypes.MsgAuctionBid{},
		&buildertypes.MsgAuctionBid{},
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
