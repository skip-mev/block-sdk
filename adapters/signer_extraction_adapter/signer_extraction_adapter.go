package signer_extraction

import (
	"fmt"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth/signing"
)

type SignerData struct {
	Signer sdk.AccAddress
	Sequence uint64
}

// SignerExtractionAdapter is an interface used to determine how the signers of a transaction should be extracted
// from the transaction.
type SignerExtractionAdapter interface {	
	GetSigners(sdk.Tx) ([]SignerData, error)
}

var _ SignerExtractionAdapter = DefaultSignerExtractionAdapter{}

// DefaultSignerExtractionAdapter is the default implementation of SignerExtractionAdapter. It extracts the signers
// from a cosmos-sdk tx via GetSignaturesV2.
type DefaultSignerExtractionAdapter struct {}

func NewDefaultSignerExtractionAdapter() DefaultSignerExtractionAdapter {
	return DefaultSignerExtractionAdapter{}
}

func (DefaultSignerExtractionAdapter) GetSigners(tx sdk.Tx) ([]SignerData, error) {
	sigTx, ok := tx.(signing.SigVerifiableTx)
	if !ok {
		return nil, fmt.Errorf("tx of type %T does not implement SigVerifiableTx", tx)
	}

	sigs, err := sigTx.GetSignaturesV2()
	if err != nil {
		return nil, err
	}

	signers := make([]SignerData, len(sigs))
	for i, sig := range sigs {
		signers[i] = SignerData{
			Signer: sig.PubKey.Address().Bytes(),
			Sequence: sig.Sequence,
		}
	}

	return signers, nil
}
