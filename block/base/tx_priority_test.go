package base_test

import (
	"fmt"
	"math/rand"
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/skip-mev/block-sdk/block/base"
	"github.com/skip-mev/block-sdk/testutils"
)

const maxUint64 = "18446744073709551616" // value is 2^64

func TestTxPriority(t *testing.T) {
	acct := testutils.RandomAccounts(rand.New(rand.NewSource(1)), 1)
	txc := testutils.CreateTestEncodingConfig().TxConfig

	type testCase struct {
		name     string
		priority base.TxPriority[string]
	}

	testCases := []testCase{
		{
			"DeprecatedTxPriority",
			base.DeprecatedTxPriority(),
		},
		{
			"DefaultTxPriority",
			base.DefaultTxPriority(),
		},
	}
	t.Run("test getting a tx priority: DefaultTxPriority", func(t *testing.T) {
		priority := base.DefaultTxPriority()

		tx, err := testutils.CreateTx(txc, acct[0], 0, 0, nil, sdk.NewCoin("stake", math.NewInt(1)))
		require.NoError(t, err)

		require.Equal(t, "1,stake", priority.GetTxPriority(nil, tx))
	})

	t.Run("test with amt that is not uint64", func(t *testing.T) {
		priority := base.DefaultTxPriority()

		amt, ok := math.NewIntFromString(maxUint64)
		require.True(t, ok)
		tx, err := testutils.CreateTx(txc, acct[0], 0, 0, nil, sdk.NewCoin("stake", amt))
		require.NoError(t, err)

		require.Equal(t, maxUint64+",stake", priority.GetTxPriority(nil, tx))

		tx2, err := testutils.CreateTx(txc, acct[0], 0, 0, nil, sdk.NewCoin("stake", math.NewInt(1)))
		require.NoError(t, err)

		require.Equal(t, 1, priority.Compare(priority.GetTxPriority(nil, tx), priority.GetTxPriority(nil, tx2)))
	})

	t.Run("test invalid priorities", func(t *testing.T) {
		priority := base.DefaultTxPriority()

		invalidAmount := "a,b"
		invalidCoins := "1,stake,2"

		require.Equal(t, 0, priority.Compare(invalidAmount, ""))
		require.Equal(t, 0, priority.Compare("", invalidCoins))
	})

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("test with non-tx: %s", tc.name), func(t *testing.T) {
			require.Equal(t, "", tc.priority.GetTxPriority(nil, nil))
		})

		t.Run(fmt.Sprintf("test tx with no fee: %s", tc.name), func(t *testing.T) {
			tx, err := testutils.CreateTx(txc, acct[0], 0, 0, nil)
			require.NoError(t, err)

			require.Equal(t, "", tc.priority.GetTxPriority(nil, tx))
		})

		t.Run(fmt.Sprintf("test comparing two tx priorities: %s", tc.name), func(t *testing.T) {
			tx1, err := testutils.CreateTx(txc, acct[0], 0, 0, nil, sdk.NewCoin("stake", math.NewInt(1)))
			require.NoError(t, err)

			tx2, err := testutils.CreateTx(txc, acct[0], 0, 0, nil, sdk.NewCoin("stake", math.NewInt(2)))
			require.NoError(t, err)

			priority1 := tc.priority.GetTxPriority(nil, tx1)
			priority2 := tc.priority.GetTxPriority(nil, tx2)

			require.Equal(t, -1, tc.priority.Compare(priority1, priority2))
			require.Equal(t, 1, tc.priority.Compare(priority2, priority1))
			require.Equal(t, 0, tc.priority.Compare(priority2, priority2))
		})

		t.Run(fmt.Sprintf("test comparing two tx priorities with nil: %s", tc.name), func(t *testing.T) {
			tx1, err := testutils.CreateTx(txc, acct[0], 0, 0, nil, sdk.NewCoin("stake", math.NewInt(1)))
			require.NoError(t, err)

			priority1 := tc.priority.GetTxPriority(nil, tx1)

			require.Equal(t, 1, tc.priority.Compare(priority1, ""))
			require.Equal(t, -1, tc.priority.Compare("", priority1))
			require.Equal(t, 0, tc.priority.Compare("", ""))
		})

		t.Run(fmt.Sprintf("test with multiple fee coins: %s", tc.name), func(t *testing.T) {
			tx, err := testutils.CreateTx(txc, acct[0], 0, 0, nil, sdk.NewCoin("stake", math.NewInt(1)), sdk.NewCoin("atom", math.NewInt(2)))
			require.NoError(t, err)

			tx2, err := testutils.CreateTx(txc, acct[0], 0, 0, nil, sdk.NewCoin("stake", math.NewInt(2)), sdk.NewCoin("atom", math.NewInt(3)))
			require.NoError(t, err)

			priority1 := tc.priority.GetTxPriority(nil, tx)
			priority2 := tc.priority.GetTxPriority(nil, tx2)

			require.Equal(t, -1, tc.priority.Compare(priority1, priority2))
			require.Equal(t, 1, tc.priority.Compare(priority2, priority1))
			require.Equal(t, 0, tc.priority.Compare(priority2, priority2))
		})

		t.Run(fmt.Sprintf("test with multiple different fee coins: %s", tc.name), func(t *testing.T) {
			tx1, err := testutils.CreateTx(txc, acct[0], 0, 0, nil, sdk.NewCoin("stake", math.NewInt(1)), sdk.NewCoin("atom", math.NewInt(2)), sdk.NewCoin("btc", math.NewInt(3)))
			require.NoError(t, err)

			tx2, err := testutils.CreateTx(txc, acct[0], 0, 0, nil, sdk.NewCoin("stake", math.NewInt(2)), sdk.NewCoin("eth", math.NewInt(3)))
			require.NoError(t, err)

			priority1 := tc.priority.GetTxPriority(nil, tx1)
			priority2 := tc.priority.GetTxPriority(nil, tx2)

			require.Equal(t, 0, tc.priority.Compare(priority1, priority2))

			tx3, err := testutils.CreateTx(txc, acct[0], 0, 0, nil, sdk.NewCoin("stake", math.NewInt(2)), sdk.NewCoin("osmo", math.NewInt(3)), sdk.NewCoin("btc", math.NewInt(3)))
			require.NoError(t, err)

			priority3 := tc.priority.GetTxPriority(nil, tx3)

			require.Equal(t, 0, tc.priority.Compare(priority3, priority1))
		})

		t.Run(fmt.Sprintf("one is nil, and the other isn't: %s", tc.name), func(t *testing.T) {
			tx1, err := testutils.CreateTx(txc, acct[0], 0, 0, nil, sdk.NewCoin("stake", math.NewInt(1)), sdk.NewCoin("atom", math.NewInt(2)), sdk.NewCoin("btc", math.NewInt(3)))
			require.NoError(t, err)

			tx2, err := testutils.CreateTx(txc, acct[0], 0, 0, nil)
			require.NoError(t, err)

			priority1 := tc.priority.GetTxPriority(nil, tx1)
			priority2 := tc.priority.GetTxPriority(nil, tx2)

			require.Equal(t, 1, tc.priority.Compare(priority1, priority2))
			require.Equal(t, -1, tc.priority.Compare(priority2, priority1))
			require.Equal(t, 0, tc.priority.Compare(priority2, priority2))
		})

		t.Run(fmt.Sprintf("incorrectly ordered fee tokens: %s", tc.name), func(t *testing.T) {
			tx1, err := testutils.CreateTx(txc, acct[0], 0, 0, nil, sdk.NewCoin("stake", math.NewInt(1)), sdk.NewCoin("atom", math.NewInt(2)), sdk.NewCoin("btc", math.NewInt(3)))
			require.NoError(t, err)

			tx2, err := testutils.CreateTx(txc, acct[0], 0, 0, nil, sdk.NewCoin("atom", math.NewInt(2)), sdk.NewCoin("stake", math.NewInt(1)), sdk.NewCoin("btc", math.NewInt(3)))
			require.NoError(t, err)

			priority1 := tc.priority.GetTxPriority(nil, tx1)
			priority2 := tc.priority.GetTxPriority(nil, tx2)

			require.Equal(t, tc.priority.Compare(priority1, priority2), 0)
		})

		t.Run(fmt.Sprintf("IBC tokens: %s", tc.name), func(t *testing.T) {
			tx1, err := testutils.CreateTx(txc, acct[0], 0, 0, nil, sdk.NewCoin("ibc/7F1D3FCF4AE79E1554D670D1AD949A9BA4E4A3C76C63093E17E446A46061A7A2", math.NewInt(1)))
			require.NoError(t, err)

			tx2, err := testutils.CreateTx(txc, acct[0], 0, 0, nil, sdk.NewCoin("ibc/7F1D3FCF4AE79E1554D670D1AD949A9BA4E4A3C76C63093E17E446A46061A7A2", math.NewInt(2)))
			require.NoError(t, err)

			priority1 := tc.priority.GetTxPriority(nil, tx1)
			priority2 := tc.priority.GetTxPriority(nil, tx2)

			require.Equal(t, -1, tc.priority.Compare(priority1, priority2))
			require.Equal(t, 1, tc.priority.Compare(priority2, priority1))
			require.Equal(t, 0, tc.priority.Compare(priority2, priority2))
		})
	}
}

func BenchmarkDefaultTxPriority(b *testing.B) {
	acct := testutils.RandomAccounts(rand.New(rand.NewSource(1)), 1)
	txc := testutils.CreateTestEncodingConfig().TxConfig

	priority := base.DefaultTxPriority()

	tx1, err := testutils.CreateTx(txc, acct[0], 0, 0, nil, sdk.NewCoin("stake", math.NewInt(1)), sdk.NewCoin("atom", math.NewInt(2)), sdk.NewCoin("btc", math.NewInt(3)))
	require.NoError(b, err)

	tx2, err := testutils.CreateTx(txc, acct[0], 0, 0, nil, sdk.NewCoin("stake", math.NewInt(2)), sdk.NewCoin("eth", math.NewInt(3)), sdk.NewCoin("btc", math.NewInt(4)))
	require.NoError(b, err)

	for i := 0; i < b.N; i++ {
		priority.Compare(priority.GetTxPriority(nil, tx1), priority.GetTxPriority(nil, tx2))
	}
}

func BenchmarkDeprecatedTxPriority(b *testing.B) {
	// ignore setup
	acct := testutils.RandomAccounts(rand.New(rand.NewSource(1)), 1)
	txc := testutils.CreateTestEncodingConfig().TxConfig

	priority := base.DeprecatedTxPriority()

	tx1, err := testutils.CreateTx(txc, acct[0], 0, 0, nil, sdk.NewCoin("stake", math.NewInt(1)), sdk.NewCoin("atom", math.NewInt(2)), sdk.NewCoin("btc", math.NewInt(3)))
	require.NoError(b, err)

	tx2, err := testutils.CreateTx(txc, acct[0], 0, 0, nil, sdk.NewCoin("stake", math.NewInt(2)), sdk.NewCoin("eth", math.NewInt(3)), sdk.NewCoin("btc", math.NewInt(4)))
	require.NoError(b, err)
	// start timer
	for i := 0; i < b.N; i++ {
		priority.Compare(priority.GetTxPriority(nil, tx1), priority.GetTxPriority(nil, tx2))
	}
}
