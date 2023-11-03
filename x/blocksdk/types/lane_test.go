package types_test

import (
	"testing"

	"cosmossdk.io/math"

	"github.com/skip-mev/block-sdk/x/blocksdk/types"
)

func TestLane_ValidateBasic(t *testing.T) {
	tests := []struct {
		name    string
		lane    types.Lane
		wantErr bool
	}{
		{
			name: "invalid no id",
			lane: types.Lane{
				Id:            "",
				MaxBlockSpace: math.LegacyOneDec(),
				Order:         0,
			},
			wantErr: true,
		},
		{
			name: "invalid maxblockspace nil",
			lane: types.Lane{
				Id:    "free",
				Order: 0,
			},
			wantErr: true,
		},
		{
			name: "invalid maxblockspace negative",
			lane: types.Lane{
				Id:            "free",
				MaxBlockSpace: math.LegacyMustNewDecFromStr("-1.0"),
				Order:         0,
			},
			wantErr: true,
		},
		{
			name: "invalid maxblockspace greater than 1",
			lane: types.Lane{
				Id:            "free",
				MaxBlockSpace: math.LegacyMustNewDecFromStr("1.1"),
				Order:         0,
			},
			wantErr: true,
		},
		{
			name: "valid",
			lane: types.Lane{
				Id:            "free",
				MaxBlockSpace: math.LegacyOneDec(),
				Order:         0,
			},
			wantErr: false,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.lane.ValidateBasic(); (err != nil) != tc.wantErr {
				t.Errorf("ValidateBasic() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}

func TestLanes_ValidateBasic(t *testing.T) {
	tests := []struct {
		name    string
		lanes   types.Lanes
		wantErr bool
	}{
		{
			name: "invalid validate basic (no id)",
			lanes: types.Lanes{
				types.Lane{
					Id:            "",
					MaxBlockSpace: math.LegacyOneDec(),
					Order:         0,
				},
			},
			wantErr: true,
		},
		{
			name: "invalid duplicate IDs",
			lanes: types.Lanes{
				types.Lane{
					Id:            "free",
					MaxBlockSpace: math.LegacyOneDec(),
					Order:         0,
				},
				types.Lane{
					Id:            "free",
					MaxBlockSpace: math.LegacyOneDec(),
					Order:         1,
				},
			},
			wantErr: true,
		},
		{
			name: "invalid duplicate orders",
			lanes: types.Lanes{
				types.Lane{
					Id:            "free",
					MaxBlockSpace: math.LegacyOneDec(),
					Order:         0,
				},
				types.Lane{
					Id:            "mev",
					MaxBlockSpace: math.LegacyOneDec(),
					Order:         0,
				},
			},
			wantErr: true,
		},
		{
			name: "invalid duplicate orders",
			lanes: types.Lanes{
				types.Lane{
					Id:            "free",
					MaxBlockSpace: math.LegacyOneDec(),
					Order:         0,
				},
				types.Lane{
					Id:            "mev",
					MaxBlockSpace: math.LegacyOneDec(),
					Order:         0,
				},
			},
			wantErr: true,
		},
		{
			name: "invalid orders non-monotonic",
			lanes: types.Lanes{
				types.Lane{
					Id:            "free",
					MaxBlockSpace: math.LegacyOneDec(),
					Order:         0,
				},
				types.Lane{
					Id:            "mev",
					MaxBlockSpace: math.LegacyOneDec(),
					Order:         2,
				},
			},
			wantErr: true,
		},
		{
			name: "valid single",
			lanes: types.Lanes{
				types.Lane{
					Id:            "free",
					MaxBlockSpace: math.LegacyOneDec(),
					Order:         0,
				},
			},
			wantErr: false,
		},
		{
			name: "valid multiple",
			lanes: types.Lanes{
				types.Lane{
					Id:            "free",
					MaxBlockSpace: math.LegacyOneDec(),
					Order:         0,
				},
				types.Lane{
					Id:            "mev",
					MaxBlockSpace: math.LegacyOneDec(),
					Order:         1,
				},
			},
			wantErr: false,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.lanes.ValidateBasic(); (err != nil) != tc.wantErr {
				t.Errorf("ValidateBasic() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}
