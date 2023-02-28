package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func NewParams(maxBundleSize uint32, escrowAddr string, reserveFee, minBuyInFee, minBidIncr sdk.Coins) Params {
	return Params{
		MaxBundleSize:        maxBundleSize,
		EscrowAccountAddress: escrowAddr,
		ReserveFee:           reserveFee,
		MinBuyInFee:          minBuyInFee,
		MinBidIncrement:      minBidIncr,
	}
}

// DefaultParams returns default x/auction module parameters.
func DefaultParams() Params {
	// TODO: Choose reasonable default values.
	//
	// Ref: https://github.com/skip-mev/pob/issues/7
	return Params{
		MaxBundleSize:        0,
		EscrowAccountAddress: "",
		ReserveFee:           sdk.NewCoins(),
		MinBuyInFee:          sdk.NewCoins(),
		MinBidIncrement:      sdk.NewCoins(),
	}
}

// Validate performs basic validation on the parameters.
func (p Params) Validate() error {
	// TODO: Implement validation.
	//
	// Ref: https://github.com/skip-mev/pob/issues/6
	panic("not implemented")
}
