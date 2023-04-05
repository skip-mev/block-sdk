package types

import (
	fmt "fmt"

	"cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

var (
	_ sdk.Msg = &MsgUpdateParams{}
	_ sdk.Msg = &MsgAuctionBid{}
)

// GetSignBytes implements the LegacyMsg interface.
func (m MsgUpdateParams) GetSignBytes() []byte {
	return sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(&m))
}

// GetSigners returns the expected signers for a MsgUpdateParams message.
func (m MsgUpdateParams) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(m.Authority)
	return []sdk.AccAddress{addr}
}

// ValidateBasic does a sanity check on the provided data.
func (m MsgUpdateParams) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Authority); err != nil {
		return errors.Wrap(err, "invalid authority address")
	}

	return m.Params.Validate()
}

// GetSignBytes implements the LegacyMsg interface.
func (m MsgAuctionBid) GetSignBytes() []byte {
	return sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(&m))
}

// GetSigners returns the expected signers for a MsgAuctionBid message.
func (m MsgAuctionBid) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(m.Bidder)
	return []sdk.AccAddress{addr}
}

// ValidateBasic does a sanity check on the provided data.
func (m MsgAuctionBid) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Bidder); err != nil {
		return errors.Wrap(err, "invalid bidder address")
	}

	// Validate the bid.
	if m.Bid.IsNil() {
		return fmt.Errorf("no bid included")
	}

	if err := m.Bid.Validate(); err != nil {
		return errors.Wrap(err, "invalid bid")
	}

	// Validate the transactions.
	if len(m.Transactions) == 0 {
		return fmt.Errorf("no transactions included")
	}

	for _, tx := range m.Transactions {
		if len(tx) == 0 {
			return fmt.Errorf("empty transaction included")
		}
	}

	return nil
}
