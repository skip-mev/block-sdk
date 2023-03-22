package abci_test

import (
	"math/rand"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/types"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	"github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	authsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
	"github.com/cosmos/cosmos-sdk/x/auth/tx"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	auctiontypes "github.com/skip-mev/pob/x/auction/types"
)

type encodingConfig struct {
	InterfaceRegistry types.InterfaceRegistry
	Codec             codec.Codec
	TxConfig          client.TxConfig
	Amino             *codec.LegacyAmino
}

func createTestEncodingConfig() encodingConfig {
	cdc := codec.NewLegacyAmino()
	interfaceRegistry := types.NewInterfaceRegistry()

	banktypes.RegisterInterfaces(interfaceRegistry)
	cryptocodec.RegisterInterfaces(interfaceRegistry)
	auctiontypes.RegisterInterfaces(interfaceRegistry)

	codec := codec.NewProtoCodec(interfaceRegistry)

	return encodingConfig{
		InterfaceRegistry: interfaceRegistry,
		Codec:             codec,
		TxConfig:          tx.NewTxConfig(codec, tx.DefaultSignModes),
		Amino:             cdc,
	}
}

type Account struct {
	PrivKey cryptotypes.PrivKey
	PubKey  cryptotypes.PubKey
	Address sdk.AccAddress
	ConsKey cryptotypes.PrivKey
}

func (acc Account) Equals(acc2 Account) bool {
	return acc.Address.Equals(acc2.Address)
}

func RandomAccounts(r *rand.Rand, n int) []Account {
	accs := make([]Account, n)

	for i := 0; i < n; i++ {
		pkSeed := make([]byte, 15)
		r.Read(pkSeed)

		accs[i].PrivKey = secp256k1.GenPrivKeyFromSecret(pkSeed)
		accs[i].PubKey = accs[i].PrivKey.PubKey()
		accs[i].Address = sdk.AccAddress(accs[i].PubKey.Address())

		accs[i].ConsKey = ed25519.GenPrivKeyFromSecret(pkSeed)
	}

	return accs
}

func createTx(txCfg client.TxConfig, account Account, nonce uint64, msgs []sdk.Msg) (authsigning.Tx, error) {
	txBuilder := txCfg.NewTxBuilder()
	if err := txBuilder.SetMsgs(msgs...); err != nil {
		return nil, err
	}

	sigV2 := signing.SignatureV2{
		PubKey: account.PrivKey.PubKey(),
		Data: &signing.SingleSignatureData{
			SignMode:  txCfg.SignModeHandler().DefaultMode(),
			Signature: nil,
		},
		Sequence: nonce,
	}
	if err := txBuilder.SetSignatures(sigV2); err != nil {
		return nil, err
	}

	return txBuilder.GetTx(), nil
}

func createRandomMsgs(acc sdk.AccAddress, numberMsgs int) []sdk.Msg {
	msgs := make([]sdk.Msg, numberMsgs)
	for i := 0; i < numberMsgs; i++ {
		msgs[i] = &banktypes.MsgSend{
			FromAddress: acc.String(),
			ToAddress:   acc.String(),
		}
	}

	return msgs
}

func createMsgAuctionBid(txCfg client.TxConfig, bidder Account, bid sdk.Coins, nonce uint64, numberMsgs int) (*auctiontypes.MsgAuctionBid, error) {
	bidMsg := &auctiontypes.MsgAuctionBid{
		Bidder:       bidder.Address.String(),
		Bid:          bid,
		Transactions: make([][]byte, numberMsgs),
	}

	for i := 0; i < numberMsgs; i++ {
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
			Sequence: nonce + uint64(i),
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

func createAuctionTxWithSigners(txCfg client.TxConfig, bidder Account, bid sdk.Coins, nonce uint64, signers []Account) (authsigning.Tx, error) {
	bidMsg := &auctiontypes.MsgAuctionBid{
		Bidder:       bidder.Address.String(),
		Bid:          bid,
		Transactions: make([][]byte, len(signers)),
	}

	for i := 0; i < len(signers); i++ {
		randomMsg := createRandomMsgs(signers[i].Address, 1)
		randomTx, err := createTx(txCfg, signers[i], 0, randomMsg)
		if err != nil {
			return nil, err
		}

		bz, err := txCfg.TxEncoder()(randomTx)
		if err != nil {
			return nil, err
		}

		bidMsg.Transactions[i] = bz
	}

	txBuilder := txCfg.NewTxBuilder()
	if err := txBuilder.SetMsgs(bidMsg); err != nil {
		return nil, err
	}

	sigV2 := signing.SignatureV2{
		PubKey: bidder.PrivKey.PubKey(),
		Data: &signing.SingleSignatureData{
			SignMode:  txCfg.SignModeHandler().DefaultMode(),
			Signature: nil,
		},
		Sequence: nonce,
	}
	if err := txBuilder.SetSignatures(sigV2); err != nil {
		return nil, err
	}

	return txBuilder.GetTx(), nil
}
