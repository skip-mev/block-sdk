package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

var (
	_ sdk.Msg = &MsgRegisterLane{}
	_ sdk.Msg = &MsgUpdateLane{}
)

// GetSignBytes implements the LegacyMsg interface.
func (m MsgRegisterLane) GetSignBytes() []byte {
	return sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(&m))
}

// GetSigners returns the expected signers for a MsgRegisterLane message.
func (m MsgRegisterLane) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(m.Authority)
	return []sdk.AccAddress{addr}
}

// ValidateBasic does a sanity check on the provided data.
func (m MsgRegisterLane) ValidateBasic() error {
	return nil
}

// GetSignBytes implements the LegacyMsg interface.
func (m MsgUpdateLane) GetSignBytes() []byte {
	return sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(&m))
}

// GetSigners returns the expected signers for a MsgUpdateLane message.
func (m MsgUpdateLane) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(m.Authority)
	return []sdk.AccAddress{addr}
}

// ValidateBasic does a sanity check on the provided data.
func (m MsgUpdateLane) ValidateBasic() error {
	return nil
}
