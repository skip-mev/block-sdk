package utils_test

import (
	"testing"

	"cosmossdk.io/math"
	"github.com/skip-mev/pob/blockbuster/utils"
)

func TestGetMaxTxBytesForLane(t *testing.T) {
	testCases := []struct {
		name         string
		maxTxBytes   int64
		totalTxBytes int64
		ratio        math.LegacyDec
		expected     int64
	}{
		{
			"ratio is zero",
			100,
			50,
			math.LegacyZeroDec(),
			50,
		},
		{
			"ratio is zero",
			100,
			100,
			math.LegacyZeroDec(),
			0,
		},
		{
			"ratio is zero",
			100,
			150,
			math.LegacyZeroDec(),
			0,
		},
		{
			"ratio is 1",
			100,
			50,
			math.LegacyOneDec(),
			100,
		},
		{
			"ratio is 10%",
			100,
			50,
			math.LegacyMustNewDecFromStr("0.1"),
			10,
		},
		{
			"ratio is 25%",
			100,
			50,
			math.LegacyMustNewDecFromStr("0.25"),
			25,
		},
		{
			"ratio is 50%",
			101,
			50,
			math.LegacyMustNewDecFromStr("0.5"),
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
