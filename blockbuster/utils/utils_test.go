package utils

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/skip-mev/pob/blockbuster"
)

func TestGetMaxTxBytesForLane(t *testing.T) {
	testCases := []struct {
		name     string
		proposal *blockbuster.Proposal
		ratio    sdk.Dec
		expected int64
	}{
		{
			"ratio is zero",
			&blockbuster.Proposal{
				MaxTxBytes:   100,
				TotalTxBytes: 50,
			},
			sdk.ZeroDec(),
			50,
		},
		{
			"ratio is zero",
			&blockbuster.Proposal{
				MaxTxBytes:   100,
				TotalTxBytes: 100,
			},
			sdk.ZeroDec(),
			0,
		},
		{
			"ratio is zero",
			&blockbuster.Proposal{
				MaxTxBytes:   100,
				TotalTxBytes: 150,
			},
			sdk.ZeroDec(),
			0,
		},
		{
			"ratio is 1",
			&blockbuster.Proposal{
				MaxTxBytes:   100,
				TotalTxBytes: 50,
			},
			sdk.OneDec(),
			100,
		},
		{
			"ratio is 10%",
			&blockbuster.Proposal{
				MaxTxBytes:   100,
				TotalTxBytes: 50,
			},
			sdk.MustNewDecFromStr("0.1"),
			10,
		},
		{
			"ratio is 25%",
			&blockbuster.Proposal{
				MaxTxBytes:   100,
				TotalTxBytes: 50,
			},
			sdk.MustNewDecFromStr("0.25"),
			25,
		},
		{
			"ratio is 50%",
			&blockbuster.Proposal{
				MaxTxBytes:   101,
				TotalTxBytes: 50,
			},
			sdk.MustNewDecFromStr("0.5"),
			50,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := GetMaxTxBytesForLane(tc.proposal, tc.ratio)
			if actual != tc.expected {
				t.Errorf("expected %d, got %d", tc.expected, actual)
			}
		})
	}
}
