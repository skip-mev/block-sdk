package networksuite

import (
	"context"

	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/client/tx"
	secp256k1 "github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	authsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
)

// Account is an abstraction around a cosmos-sdk private-key (account)
type Account struct {
	pk cryptotypes.PrivKey
}

// NewAccount returns a new account, with a randomly generated private-key.
func NewAccount() *Account {
	return &Account{
		pk: secp256k1.GenPrivKey(),
	}
}

// Address returns the address of the account.
func (a Account) Address() sdk.AccAddress {
	return sdk.AccAddress(a.pk.PubKey().Address())
}

// PubKey returns the public-key of the account.
func (a Account) PubKey() cryptotypes.PubKey {
	return a.pk.PubKey()
}

// CreateTx creates and signs a transaction, from the given messages
func (a Account) CreateTx(ctx context.Context, accNum, seq, gasLimit, fee, timeoutHeight uint64, msgs ...sdk.Msg) ([]byte, error) {
	txb := txc.NewTxBuilder()

	if err := txb.SetMsgs(msgs...); err != nil {
		return nil, err
	}

	// set params 
	txb.SetGasLimit(gasLimit)
	txb.SetFeeAmount(sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, math.NewInt(int64(fee)))))
	txb.SetTimeoutHeight(timeoutHeight)

	sigV2 := signing.SignatureV2{
		PubKey: a.pk.PubKey(),
		Data: &signing.SingleSignatureData{
			SignMode:  signing.SignMode(txc.SignModeHandler().DefaultMode()),
			Signature: nil,
		},
		Sequence: seq,
	}

	if err := txb.SetSignatures(sigV2); err != nil {
		return nil, err
	}

	// now actually sign
	signerData := authsigning.SignerData{
		ChainID: 	 chainID,
		AccountNumber: accNum,
		Sequence: 	 seq,
		PubKey: a.pk.PubKey(),
	}

	sigV2, err := tx.SignWithPrivKey(
		ctx, signing.SignMode(txc.SignModeHandler().DefaultMode()), signerData,
		txb, a.pk, txc, seq,
	)
	if err != nil {
		return nil, err
	}

	if err := txb.SetSignatures(sigV2); err != nil {
		return nil, err
	}

	// return tx
	return txc.TxEncoder()(txb.GetTx())
}
