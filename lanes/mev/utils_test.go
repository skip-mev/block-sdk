package mev_test

import (
	"testing"

	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/stretchr/testify/require"

	"github.com/skip-mev/block-sdk/v2/lanes/mev"
	testutils "github.com/skip-mev/block-sdk/v2/testutils"
	auctiontypes "github.com/skip-mev/block-sdk/v2/x/auction/types"
)

func TestGetMsgAuctionBidFromTx_Valid(t *testing.T) {
	encCfg := testutils.CreateTestEncodingConfig()

	txBuilder := encCfg.TxConfig.NewTxBuilder()
	txBuilder.SetMsgs(&auctiontypes.MsgAuctionBid{})

	msg, err := mev.GetMsgAuctionBidFromTx(txBuilder.GetTx())
	require.NoError(t, err)
	require.NotNil(t, msg)
}

func TestGetMsgAuctionBidFromTx_MultiMsgBid(t *testing.T) {
	encCfg := testutils.CreateTestEncodingConfig()

	txBuilder := encCfg.TxConfig.NewTxBuilder()
	txBuilder.SetMsgs(
		&auctiontypes.MsgAuctionBid{},
		&auctiontypes.MsgAuctionBid{},
		&banktypes.MsgSend{},
	)

	msg, err := mev.GetMsgAuctionBidFromTx(txBuilder.GetTx())
	require.Error(t, err)
	require.Nil(t, msg)
}

func TestGetMsgAuctionBidFromTx_NoBid(t *testing.T) {
	encCfg := testutils.CreateTestEncodingConfig()

	txBuilder := encCfg.TxConfig.NewTxBuilder()
	txBuilder.SetMsgs(&banktypes.MsgSend{})

	msg, err := mev.GetMsgAuctionBidFromTx(txBuilder.GetTx())
	require.NoError(t, err)
	require.Nil(t, msg)
}
