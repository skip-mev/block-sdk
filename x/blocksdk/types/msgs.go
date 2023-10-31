package types

import (
	"cosmossdk.io/errors"
	"cosmossdk.io/math"
	"fmt"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

var (
	_ sdk.Msg = &MsgRegisterLane{}
	_ sdk.Msg = &MsgUpdateLane{}
)

// GetSignBytes implements the LegacyMsg interface.
func (m *MsgRegisterLane) GetSignBytes() []byte {
	return sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(&m))
}

// GetSigners returns the expected signers for a MsgRegisterLane message.
func (m *MsgRegisterLane) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(m.Authority)
	return []sdk.AccAddress{addr}
}

// ValidateBasic does a sanity check on the provided data.
func (m *MsgRegisterLane) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Authority); err != nil {
		return errors.Wrap(err, "invalid authority address")
	}

	if m.Lane.Id == "" {
		return fmt.Errorf("lane must have an id specified")
	}

	if m.Lane.MaxBlockSpace.IsNil() || m.Lane.MaxBlockSpace.IsNegative() || m.Lane.MaxBlockSpace.GT(math.LegacyOneDec()) {
		return fmt.Errorf("max block space must be set to a value between 0 and 1")
	}

	return nil
}

// GetSignBytes implements the LegacyMsg interface.
func (m *MsgUpdateLane) GetSignBytes() []byte {
	return sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(&m))
}

// GetSigners returns the expected signers for a MsgUpdateLane message.
func (m *MsgUpdateLane) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(m.Authority)
	return []sdk.AccAddress{addr}
}

// ValidateBasic does a sanity check on the provided data.
func (m *MsgUpdateLane) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Authority); err != nil {
		return errors.Wrap(err, "invalid authority address")
	}

	if m.Lane.Id == "" {
		return fmt.Errorf("lane must have an id specified")
	}

	if m.Lane.MaxBlockSpace.IsNil() || m.Lane.MaxBlockSpace.IsNegative() || m.Lane.MaxBlockSpace.GT(math.LegacyOneDec()) {
		return fmt.Errorf("max block space must be set to a value between 0 and 1")
	}

	return nil
}
