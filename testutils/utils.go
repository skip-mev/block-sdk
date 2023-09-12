package test

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
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
<<<<<<< HEAD
	buildertypes "github.com/skip-mev/block-sdk/x/builder/types"
=======
	"github.com/cosmos/gogoproto/proto"

	auctiontypes "github.com/skip-mev/block-sdk/x/auction/types"
>>>>>>> 3c6f319 (feat(docs): rename x/builder -> x/auction (#55))
)

type EncodingConfig struct {
	InterfaceRegistry types.InterfaceRegistry
	Codec             codec.Codec
	TxConfig          client.TxConfig
	Amino             *codec.LegacyAmino
}

func CreateTestEncodingConfig() EncodingConfig {
	cdc := codec.NewLegacyAmino()
	interfaceRegistry := types.NewInterfaceRegistry()

	banktypes.RegisterInterfaces(interfaceRegistry)
	cryptocodec.RegisterInterfaces(interfaceRegistry)
	auctiontypes.RegisterInterfaces(interfaceRegistry)
	stakingtypes.RegisterInterfaces(interfaceRegistry)

	codec := codec.NewProtoCodec(interfaceRegistry)

	return EncodingConfig{
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

func CreateTx(txCfg client.TxConfig, account Account, nonce, timeout uint64, msgs []sdk.Msg, fees ...sdk.Coin) (authsigning.Tx, error) {
	txBuilder := txCfg.NewTxBuilder()
	if err := txBuilder.SetMsgs(msgs...); err != nil {
		return nil, err
	}

	sigV2 := signing.SignatureV2{
		PubKey: account.PrivKey.PubKey(),
		Data: &signing.SingleSignatureData{
			SignMode:  signing.SignMode_SIGN_MODE_DIRECT,
			Signature: nil,
		},
		Sequence: nonce,
	}
	if err := txBuilder.SetSignatures(sigV2); err != nil {
		return nil, err
	}

	txBuilder.SetTimeoutHeight(timeout)

	txBuilder.SetFeeAmount(fees)

	return txBuilder.GetTx(), nil
}

func CreateFreeTx(txCfg client.TxConfig, account Account, nonce, timeout uint64, validator string, amount sdk.Coin, fees ...sdk.Coin) (authsigning.Tx, error) {
	msgs := []sdk.Msg{
		&stakingtypes.MsgDelegate{
			DelegatorAddress: account.Address.String(),
			ValidatorAddress: validator,
			Amount:           amount,
		},
	}

	return CreateTx(txCfg, account, nonce, timeout, msgs, fees...)
}

func CreateRandomTx(txCfg client.TxConfig, account Account, nonce, numberMsgs, timeout uint64, fees ...sdk.Coin) (authsigning.Tx, error) {
	msgs := make([]sdk.Msg, numberMsgs)
	for i := 0; i < int(numberMsgs); i++ {
		msgs[i] = &banktypes.MsgSend{
			FromAddress: account.Address.String(),
			ToAddress:   account.Address.String(),
		}
	}

	txBuilder := txCfg.NewTxBuilder()
	if err := txBuilder.SetMsgs(msgs...); err != nil {
		return nil, err
	}

	sigV2 := signing.SignatureV2{
		PubKey: account.PrivKey.PubKey(),
		Data: &signing.SingleSignatureData{
			SignMode:  signing.SignMode_SIGN_MODE_DIRECT,
			Signature: nil,
		},
		Sequence: nonce,
	}
	if err := txBuilder.SetSignatures(sigV2); err != nil {
		return nil, err
	}

	txBuilder.SetTimeoutHeight(timeout)

	txBuilder.SetFeeAmount(fees)

	return txBuilder.GetTx(), nil
}

func CreateRandomTxBz(txCfg client.TxConfig, account Account, nonce, numberMsgs, timeout uint64) ([]byte, error) {
	tx, err := CreateRandomTx(txCfg, account, nonce, numberMsgs, timeout)
	if err != nil {
		return nil, err
	}

	return txCfg.TxEncoder()(tx)
}

func CreateTxWithSigners(txCfg client.TxConfig, nonce, timeout uint64, signers []Account) (authsigning.Tx, error) {
	msgs := []sdk.Msg{}
	for _, signer := range signers {
		msg := CreateRandomMsgs(signer.Address, 1)
		msgs = append(msgs, msg...)
	}

	txBuilder := txCfg.NewTxBuilder()
	if err := txBuilder.SetMsgs(msgs...); err != nil {
		return nil, err
	}

	sigV2 := signing.SignatureV2{
		PubKey: signers[0].PrivKey.PubKey(),
		Data: &signing.SingleSignatureData{
			SignMode:  signing.SignMode_SIGN_MODE_DIRECT,
			Signature: nil,
		},
		Sequence: nonce,
	}

	if err := txBuilder.SetSignatures(sigV2); err != nil {
		return nil, err
	}

	txBuilder.SetTimeoutHeight(timeout)

	return txBuilder.GetTx(), nil
}

func CreateAuctionTx(txCfg client.TxConfig, bidder Account, bid sdk.Coin, nonce, timeout uint64, signers []Account) (authsigning.Tx, []authsigning.Tx, error) {
	bidMsg := &auctiontypes.MsgAuctionBid{
		Bidder:       bidder.Address.String(),
		Bid:          bid,
		Transactions: make([][]byte, len(signers)),
	}

	txs := []authsigning.Tx{}

	for i := 0; i < len(signers); i++ {
		randomMsg := CreateRandomMsgs(signers[i].Address, 1)
		randomTx, err := CreateTx(txCfg, signers[i], 0, timeout, randomMsg)
		if err != nil {
			return nil, nil, err
		}

		bz, err := txCfg.TxEncoder()(randomTx)
		if err != nil {
			return nil, nil, err
		}

		bidMsg.Transactions[i] = bz
		txs = append(txs, randomTx)
	}

	txBuilder := txCfg.NewTxBuilder()
	if err := txBuilder.SetMsgs(bidMsg); err != nil {
		return nil, nil, err
	}

	sigV2 := signing.SignatureV2{
		PubKey: bidder.PrivKey.PubKey(),
		Data: &signing.SingleSignatureData{
			SignMode:  signing.SignMode_SIGN_MODE_DIRECT,
			Signature: nil,
		},
		Sequence: nonce,
	}
	if err := txBuilder.SetSignatures(sigV2); err != nil {
		return nil, nil, err
	}

	txBuilder.SetTimeoutHeight(timeout)

	return txBuilder.GetTx(), txs, nil
}

func CreateAuctionTxWithSigners(txCfg client.TxConfig, bidder Account, bid sdk.Coin, nonce, timeout uint64, signers []Account) (authsigning.Tx, error) {
	bidMsg := &auctiontypes.MsgAuctionBid{
		Bidder:       bidder.Address.String(),
		Bid:          bid,
		Transactions: make([][]byte, len(signers)),
	}

	for i := 0; i < len(signers); i++ {
		randomMsg := CreateRandomMsgs(signers[i].Address, 1)
		randomTx, err := CreateTx(txCfg, signers[i], 0, timeout, randomMsg)
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
			SignMode:  signing.SignMode_SIGN_MODE_DIRECT,
			Signature: nil,
		},
		Sequence: nonce,
	}
	if err := txBuilder.SetSignatures(sigV2); err != nil {
		return nil, err
	}

	txBuilder.SetTimeoutHeight(timeout)

	return txBuilder.GetTx(), nil
}

func CreateAuctionTxWithSignerBz(txCfg client.TxConfig, bidder Account, bid sdk.Coin, nonce, timeout uint64, signers []Account) ([]byte, error) {
	bidTx, err := CreateAuctionTxWithSigners(txCfg, bidder, bid, nonce, timeout, signers)
	if err != nil {
		return nil, err
	}

	bz, err := txCfg.TxEncoder()(bidTx)
	if err != nil {
		return nil, err
	}

	return bz, nil
}

func CreateRandomMsgs(acc sdk.AccAddress, numberMsgs int) []sdk.Msg {
	msgs := make([]sdk.Msg, numberMsgs)
	for i := 0; i < numberMsgs; i++ {
		msgs[i] = &banktypes.MsgSend{
			FromAddress: acc.String(),
			ToAddress:   acc.String(),
		}
	}

	return msgs
}

func CreateMsgAuctionBid(txCfg client.TxConfig, bidder Account, bid sdk.Coin, nonce uint64, numberMsgs int) (*auctiontypes.MsgAuctionBid, error) {
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
				SignMode:  signing.SignMode_SIGN_MODE_DIRECT,
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
