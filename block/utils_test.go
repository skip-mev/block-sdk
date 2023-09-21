package block_test

import (
	"testing"

	"cosmossdk.io/math"
	"github.com/skip-mev/block-sdk/block"
)

func TestGetMaxTxBytesForLane(t *testing.T) {
	testCases := []struct {
		name             string
		maxTxBytes       int64
		totalTxBytes     int64
		maxGasLimit      uint64
		totalGasLimit    uint64
		ratio            math.LegacyDec
		expectedTxBytes  int64
		expectedGasLimit uint64
	}{
		{
			"ratio is zero",
			100,
			50,
			100,
			50,
			math.LegacyZeroDec(),
			50,
			50,
		},
		{
			"ratio is zero",
			100,
			100,
			0,
			0,
			math.LegacyZeroDec(),
			0,
			0,
		},
		{
			"ratio is zero",
			100,
			150,
			0,
			0,
			math.LegacyZeroDec(),
			0,
			0,
		},
		{
			"ratio is 1",
			100,
			50,
			0,
			0,
			math.LegacyOneDec(),
			100,
			0,
		},
		{
			"ratio is 10%",
			100,
			50,
			0,
			0,
			math.LegacyMustNewDecFromStr("0.1"),
			10,
			0,
		},
		{
			"ratio is 25%",
			100,
			50,
			0,
			0,
			math.LegacyMustNewDecFromStr("0.25"),
			25,
			0,
		},
		{
			"ratio is 50%",
			101,
			50,
			0,
			0,
			math.LegacyMustNewDecFromStr("0.5"),
			50,
			0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := block.GetLaneLimit(
				tc.maxTxBytes, tc.totalTxBytes,
				tc.maxGasLimit, tc.totalGasLimit,
				tc.ratio,
			)

			if actual.MaxTxBytesLimit != tc.expectedTxBytes {
				t.Errorf("expected %d, got %d", tc.expectedGasLimit, actual.MaxTxBytesLimit)
			}

			if actual.MaxGasLimit != tc.expectedGasLimit {
				t.Errorf("expected %d, got %d", tc.expectedGasLimit, actual.MaxGasLimit)
			}
		})
	}
}
