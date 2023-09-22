package block_test

import (
	"testing"

	"cosmossdk.io/math"
	"github.com/skip-mev/block-sdk/block"
)

func TestGetMaxTxBytesForLane(t *testing.T) {
	testCases := []struct {
		name              string
		maxTxBytes        int64
		totalTxBytesUsed  int64
		maxGasLimit       uint64
		totalGasLimitUsed uint64
		ratio             math.LegacyDec
		expectedTxBytes   int64
		expectedGasLimit  uint64
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
			50,
			25,
			math.LegacyZeroDec(),
			0,
			25,
		},
		{
			"ratio is zero",
			100,
			150,
			100,
			150,
			math.LegacyZeroDec(),
			0,
			0,
		},
		{
			"ratio is 1",
			100,
			0,
			75,
			0,
			math.LegacyOneDec(),
			100,
			75,
		},
		{
			"ratio is 10%",
			100,
			0,
			75,
			0,
			math.LegacyMustNewDecFromStr("0.1"),
			10,
			7,
		},
		{
			"ratio is 25%",
			100,
			0,
			80,
			0,
			math.LegacyMustNewDecFromStr("0.25"),
			25,
			20,
		},
		{
			"ratio is 50%",
			101,
			0,
			75,
			0,
			math.LegacyMustNewDecFromStr("0.5"),
			50,
			37,
		},
		{
			"ratio is 33%",
			100,
			0,
			75,
			0,
			math.LegacyMustNewDecFromStr("0.33"),
			33,
			24,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := block.GetLaneLimits(
				tc.maxTxBytes, tc.totalTxBytesUsed,
				tc.maxGasLimit, tc.totalGasLimitUsed,
				tc.ratio,
			)

			if actual.MaxTxBytes != tc.expectedTxBytes {
				t.Errorf("expected tx bytes %d, got %d", tc.expectedTxBytes, actual.MaxTxBytes)
			}

			if actual.MaxGas != tc.expectedGasLimit {
				t.Errorf("expected gas limit %d, got %d", tc.expectedGasLimit, actual.MaxGas)
			}
		})
	}
}
