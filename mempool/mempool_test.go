package mempool_test

import (
	"math/rand"
	"testing"
	"time"

	"github.com/cometbft/cometbft/libs/log"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	"github.com/cosmos/cosmos-sdk/client"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/skip-mev/pob/mempool"
	auctiontypes "github.com/skip-mev/pob/x/auction/types"
	"github.com/stretchr/testify/require"
)

func TestAuctionMempool(t *testing.T) {
	encCfg := createTestEncodingConfig()
	amp := mempool.NewAuctionMempool(encCfg.TxConfig.TxEncoder())
	ctx := sdk.NewContext(nil, cmtproto.Header{}, false, log.NewNopLogger())
	rng := rand.New(rand.NewSource(time.Now().Unix()))
	accounts := RandomAccounts(rng, 5)

	accNonces := map[string]uint64{}
	for _, acc := range accounts {
		accNonces[acc.Address.String()] = 0
	}

	// insert a bunch of normal txs
	for i := 0; i < 1000; i++ {
		p := rng.Int63n(500-1) + 1
		j := rng.Intn(len(accounts))
		acc := accounts[j]
		txBuilder := encCfg.TxConfig.NewTxBuilder()

		msgs := []sdk.Msg{
			&banktypes.MsgSend{
				FromAddress: acc.Address.String(),
				ToAddress:   acc.Address.String(),
			},
		}
		err := txBuilder.SetMsgs(msgs...)
		require.NoError(t, err)

		sigV2 := signing.SignatureV2{
			PubKey: acc.PrivKey.PubKey(),
			Data: &signing.SingleSignatureData{
				SignMode:  encCfg.TxConfig.SignModeHandler().DefaultMode(),
				Signature: nil,
			},
			Sequence: accNonces[acc.Address.String()],
		}
		err = txBuilder.SetSignatures(sigV2)
		require.NoError(t, err)

		accNonces[acc.Address.String()]++

		require.NoError(t, amp.Insert(ctx.WithPriority(p), txBuilder.GetTx()))
	}

	require.Nil(t, amp.SelectTopAuctionBidTx())

	// insert bid transactions
	var highestBid sdk.Coins
	biddingAccs := RandomAccounts(rng, 100)

	for _, acc := range biddingAccs {
		p := rng.Int63n(500-1) + 1
		txBuilder := encCfg.TxConfig.NewTxBuilder()

		// keep track of highest bid
		bid := sdk.NewCoins(sdk.NewInt64Coin("foo", p))
		if bid.IsAllGT(highestBid) {
			highestBid = bid
		}

		bidMsg, err := createMsgAuctionBid(encCfg.TxConfig, acc, bid)
		require.NoError(t, err)

		msgs := []sdk.Msg{bidMsg}
		err = txBuilder.SetMsgs(msgs...)
		require.NoError(t, err)

		sigV2 := signing.SignatureV2{
			PubKey: acc.PrivKey.PubKey(),
			Data: &signing.SingleSignatureData{
				SignMode:  encCfg.TxConfig.SignModeHandler().DefaultMode(),
				Signature: nil,
			},
			Sequence: 0,
		}
		err = txBuilder.SetSignatures(sigV2)
		require.NoError(t, err)

		require.NoError(t, amp.Insert(ctx.WithPriority(p), txBuilder.GetTx()))

		// Insert the referenced txs just to ensure that they are removed from the
		// mempool in cases where they exist.
		for _, refRawTx := range bidMsg.GetTransactions() {
			refTx, err := encCfg.TxConfig.TxDecoder()(refRawTx)
			require.NoError(t, err)
			require.NoError(t, amp.Insert(ctx.WithPriority(0), refTx))
		}
	}

	expectedCount := 1000 + 100 + 200
	require.Equal(t, expectedCount, amp.CountTx())

	// select the top bid and misc txs
	bidTx := amp.SelectTopAuctionBidTx()
	require.Len(t, bidTx.GetMsgs(), 1)
	require.Equal(t, highestBid, bidTx.GetMsgs()[0].(*auctiontypes.MsgAuctionBid).Bid)

	// remove bid tx, which should also removed the referenced txs
	require.NoError(t, amp.Remove(bidTx))
	require.Equal(t, expectedCount-3, amp.CountTx())
}

func createMsgAuctionBid(txCfg client.TxConfig, bidder Account, bid sdk.Coins) (*auctiontypes.MsgAuctionBid, error) {
	bidMsg := &auctiontypes.MsgAuctionBid{
		Bidder:       bidder.Address.String(),
		Bid:          bid,
		Transactions: make([][]byte, 2),
	}

	for i := 0; i < 2; i++ {
		txBuilder := txCfg.NewTxBuilder()

		msgs := []sdk.Msg{
			&banktypes.MsgSend{
				FromAddress: bidder.Address.String(),
				ToAddress:   bidder.Address.String(),
			},
		}
		if err := txBuilder.SetMsgs(msgs...); err != nil {
			return nil, err
		}

		sigV2 := signing.SignatureV2{
			PubKey: bidder.PrivKey.PubKey(),
			Data: &signing.SingleSignatureData{
				SignMode:  txCfg.SignModeHandler().DefaultMode(),
				Signature: nil,
			},
			Sequence: uint64(i + 1),
		}
		if err := txBuilder.SetSignatures(sigV2); err != nil {
			return nil, err
		}

		bz, err := txCfg.TxEncoder()(txBuilder.GetTx())
		if err != nil {
			return nil, err
		}

		bidMsg.Transactions[i] = bz
	}

	return bidMsg, nil
}
