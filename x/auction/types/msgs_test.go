package types_test

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
<<<<<<< HEAD:x/builder/types/msgs_test.go
	"github.com/skip-mev/block-sdk/x/builder/types"
=======

	"github.com/skip-mev/block-sdk/x/auction/types"
>>>>>>> 3c6f319 (feat(docs): rename x/builder -> x/auction (#55)):x/auction/types/msgs_test.go
)

// TestMsgAuctionBid tests the ValidateBasic method of MsgAuctionBid
func TestMsgAuctionBid(t *testing.T) {
	cases := []struct {
		description string
		msg         types.MsgAuctionBid
		expectPass  bool
	}{
		{
			description: "invalid message with empty bidder",
			msg: types.MsgAuctionBid{
				Bidder:       "",
				Bid:          sdk.Coin{},
				Transactions: [][]byte{},
			},
			expectPass: false,
		},
		{
			description: "invalid message with empty bid",
			msg: types.MsgAuctionBid{
				Bidder:       sdk.AccAddress([]byte("test")).String(),
				Bid:          sdk.Coin{},
				Transactions: [][]byte{},
			},
			expectPass: false,
		},
		{
			description: "invalid message with empty transactions",
			msg: types.MsgAuctionBid{
				Bidder:       sdk.AccAddress([]byte("test")).String(),
				Bid:          sdk.NewCoin("test", math.NewInt(100)),
				Transactions: [][]byte{},
			},
			expectPass: false,
		},
		{
			description: "valid message",
			msg: types.MsgAuctionBid{
				Bidder:       sdk.AccAddress([]byte("test")).String(),
				Bid:          sdk.NewCoin("test", math.NewInt(100)),
				Transactions: [][]byte{[]byte("test")},
			},
			expectPass: true,
		},
		{
			description: "valid message with multiple transactions",
			msg: types.MsgAuctionBid{
				Bidder:       sdk.AccAddress([]byte("test")).String(),
				Bid:          sdk.NewCoin("test", math.NewInt(100)),
				Transactions: [][]byte{[]byte("test"), []byte("test2")},
			},
			expectPass: true,
		},
		{
			description: "invalid message with empty transaction in transactions",
			msg: types.MsgAuctionBid{
				Bidder:       sdk.AccAddress([]byte("test")).String(),
				Bid:          sdk.NewCoin("test", math.NewInt(100)),
				Transactions: [][]byte{[]byte("test"), []byte("")},
			},
			expectPass: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.description, func(t *testing.T) {
			err := tc.msg.ValidateBasic()
			if tc.expectPass {
				if err != nil {
					t.Errorf("expected no error on %s, got %s", tc.description, err)
				}
			} else {
				if err == nil {
					t.Errorf("expected error on %s, got none", tc.description)
				}
			}
		})
	}
}

// TestMsgUpdateParams tests the ValidateBasic method of MsgUpdateParams
func TestMsgUpdateParams(t *testing.T) {
	cases := []struct {
		description string
		msg         types.MsgUpdateParams
		expectPass  bool
	}{
		{
			description: "invalid message with empty authority address",
			msg: types.MsgUpdateParams{
				Authority: "",
				Params:    types.Params{},
			},
			expectPass: false,
		},
		{
			description: "invalid message with invalid params (invalid escrow address)",
			msg: types.MsgUpdateParams{
				Authority: sdk.AccAddress([]byte("test")).String(),
				Params: types.Params{
					EscrowAccountAddress: nil,
					ReserveFee:           sdk.NewCoin("test", math.NewInt(100)),
					MinBidIncrement:      sdk.NewCoin("test", math.NewInt(100)),
					ProposerFee:          math.LegacyNewDecFromInt(math.NewInt(1)),
				},
			},
			expectPass: false,
		},
		{
			description: "valid message",
			msg: types.MsgUpdateParams{
				Authority: sdk.AccAddress([]byte("test")).String(),
				Params: types.Params{
					ProposerFee:          math.LegacyNewDecFromInt(math.NewInt(1)),
					EscrowAccountAddress: sdk.AccAddress([]byte("test")),
					ReserveFee:           sdk.NewCoin("test", math.NewInt(100)),
					MinBidIncrement:      sdk.NewCoin("test", math.NewInt(100)),
				},
			},
			expectPass: true,
		},
		{
			description: "invalid message with multiple fee denoms",
			msg: types.MsgUpdateParams{
				Authority: sdk.AccAddress([]byte("test")).String(),
				Params: types.Params{
					ProposerFee:          math.LegacyNewDecFromInt(math.NewInt(1)),
					EscrowAccountAddress: sdk.AccAddress([]byte("test")),
					ReserveFee:           sdk.NewCoin("test", math.NewInt(100)),
					MinBidIncrement:      sdk.NewCoin("test2", math.NewInt(100)),
				},
			},
			expectPass: false,
		},
		{
			description: "invalid message with unset fee denoms",
			msg: types.MsgUpdateParams{
				Authority: sdk.AccAddress([]byte("test")).String(),
				Params: types.Params{
					ProposerFee:          math.LegacyNewDecFromInt(math.NewInt(1)),
					EscrowAccountAddress: sdk.AccAddress([]byte("test")),
				},
			},
			expectPass: false,
		},
		{
			description: "invalid message with min bid increment equal to 0",
			msg: types.MsgUpdateParams{
				Authority: sdk.AccAddress([]byte("test")).String(),
				Params: types.Params{
					ProposerFee:          math.LegacyNewDecFromInt(math.NewInt(1)),
					EscrowAccountAddress: sdk.AccAddress([]byte("test")),
					ReserveFee:           sdk.NewCoin("test", math.NewInt(100)),
					MinBidIncrement:      sdk.NewCoin("test", math.NewInt(0)),
				},
			},
			expectPass: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.description, func(t *testing.T) {
			err := tc.msg.ValidateBasic()
			if tc.expectPass {
				if err != nil {
					t.Errorf("expected no error on %s, got %s", tc.description, err)
				}
			} else {
				if err == nil {
					t.Errorf("expected error on %s, got none", tc.description)
				}
			}
		})
	}
}
