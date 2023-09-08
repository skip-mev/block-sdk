package types

import (
	fmt "fmt"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
)

var (
	DefaultMaxBundleSize          uint32 = 2
	DefaultEscrowAccountAddress          = authtypes.NewModuleAddress(ModuleName)
	DefaultReserveFee                    = sdk.NewCoin("stake", math.NewInt(1))
	DefaultMinBidIncrement               = sdk.NewCoin("stake", math.NewInt(1))
	DefaultFrontRunningProtection        = true
	DefaultProposerFee                   = math.LegacyNewDec(0)
)

// NewParams returns a new Params instance with the provided values.
func NewParams(
	maxBundleSize uint32,
	escrowAccountAddress []byte,
	reserveFee, minBidIncrement sdk.Coin,
	frontRunningProtection bool,
	proposerFee math.LegacyDec,
) Params {
	return Params{
		MaxBundleSize:          maxBundleSize,
		EscrowAccountAddress:   escrowAccountAddress,
		ReserveFee:             reserveFee,
		MinBidIncrement:        minBidIncrement,
		FrontRunningProtection: frontRunningProtection,
		ProposerFee:            proposerFee,
	}
}

// DefaultParams returns the default x/builder parameters.
func DefaultParams() Params {
	return NewParams(
		DefaultMaxBundleSize,
		DefaultEscrowAccountAddress,
		DefaultReserveFee,
		DefaultMinBidIncrement,
		DefaultFrontRunningProtection,
		DefaultProposerFee,
	)
}

// Validate performs basic validation on the parameters.
func (p Params) Validate() error {
	if p.EscrowAccountAddress == nil {
		return fmt.Errorf("escrow account address cannot be nil")
	}

	if err := validateFee(p.ReserveFee); err != nil {
		return fmt.Errorf("invalid reserve fee (%s)", err)
	}

	if err := validateFee(p.MinBidIncrement); err != nil {
		return fmt.Errorf("invalid minimum bid increment (%s)", err)
	}

	// Minimum bid increment must always be greater than 0.
	if p.MinBidIncrement.IsLTE(sdk.NewCoin(p.MinBidIncrement.Denom, math.ZeroInt())) {
		return fmt.Errorf("minimum bid increment cannot be zero")
	}

	denoms := map[string]struct{}{
		p.ReserveFee.Denom:      {},
		p.MinBidIncrement.Denom: {},
	}

	if len(denoms) != 1 {
		return fmt.Errorf("mismatched auction fee denoms: minimum bid increment (%s), reserve fee (%s)", p.MinBidIncrement, p.ReserveFee)
	}

	return validateProposerFee(p.ProposerFee)
}

func validateFee(fee sdk.Coin) error {
	if fee.IsNil() {
		return fmt.Errorf("fee cannot be nil: %s", fee)
	}

	return fee.Validate()
}

func validateProposerFee(v math.LegacyDec) error {
	if v.IsNil() {
		return fmt.Errorf("proposer fee cannot be nil: %s", v)
	}
	if v.IsNegative() {
		return fmt.Errorf("proposer fee cannot be negative: %s", v)
	}
	if v.GT(math.LegacyOneDec()) {
		return fmt.Errorf("proposer fee too large: %s", v)
	}

	return nil
}
