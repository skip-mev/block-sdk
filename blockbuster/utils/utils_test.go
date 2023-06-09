package utils_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/skip-mev/pob/blockbuster/utils"
)

func TestGetMaxTxBytesForLane(t *testing.T) {
	testCases := []struct {
		name         string
		maxTxBytes   int64
		totalTxBytes int64
		ratio        sdk.Dec
		expected     int64
	}{
		{
			"ratio is zero",
			100,
			50,
			sdk.ZeroDec(),
			50,
		},
		{
			"ratio is zero",
			100,
			100,
			sdk.ZeroDec(),
			0,
		},
		{
			"ratio is zero",
			100,
			150,
			sdk.ZeroDec(),
			0,
		},
		{
			"ratio is 1",
			100,
			50,
			sdk.OneDec(),
			100,
		},
		{
			"ratio is 10%",
			100,
			50,
			sdk.MustNewDecFromStr("0.1"),
			10,
		},
		{
			"ratio is 25%",
			100,
			50,
			sdk.MustNewDecFromStr("0.25"),
			25,
		},
		{
			"ratio is 50%",
			101,
			50,
			sdk.MustNewDecFromStr("0.5"),
			50,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := utils.GetMaxTxBytesForLane(tc.maxTxBytes, tc.totalTxBytes, tc.ratio)
			if actual != tc.expected {
				t.Errorf("expected %d, got %d", tc.expected, actual)
			}
		})
	}
}
