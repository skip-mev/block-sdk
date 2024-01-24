package types_test

import (
	"math/rand"
	"testing"
	"time"

	"cosmossdk.io/math"

	"github.com/skip-mev/block-sdk/v2/testutils"

	"github.com/skip-mev/block-sdk/v2/x/blocksdk/types"
)

func TestMsgUpdateLane_ValidateBasic(t *testing.T) {
	testAcc := testutils.RandomAccounts(rand.New(rand.NewSource(time.Now().Unix())), 1)[0]

	tests := []struct {
		name    string
		msg     types.MsgUpdateLane
		wantErr bool
	}{
		{
			name: "invalid empty authority",
			msg: types.MsgUpdateLane{
				Authority: "",
				Lane: types.Lane{
					Id:            "free",
					MaxBlockSpace: math.LegacyOneDec(),
					Order:         0,
				},
			},
			wantErr: true,
		},
		{
			name: "invalid authority",
			msg: types.MsgUpdateLane{
				Authority: "invalid",
				Lane: types.Lane{
					Id:            "free",
					MaxBlockSpace: math.LegacyOneDec(),
					Order:         0,
				},
			},
			wantErr: true,
		},
		{
			name: "invalid lane",
			msg: types.MsgUpdateLane{
				Authority: testAcc.Address.String(),
				Lane: types.Lane{
					Id:            "",
					MaxBlockSpace: math.LegacyOneDec(),
					Order:         0,
				},
			},
			wantErr: true,
		},
		{
			name: "valid",
			msg: types.MsgUpdateLane{
				Authority: testAcc.Address.String(),
				Lane: types.Lane{
					Id:            "free",
					MaxBlockSpace: math.LegacyOneDec(),
					Order:         0,
				},
			},
			wantErr: false,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.msg.ValidateBasic(); (err != nil) != tc.wantErr {
				t.Errorf("ValidateBasic() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}
