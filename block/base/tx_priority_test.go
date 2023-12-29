package base_test

import (
	"math/rand"
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/skip-mev/block-sdk/block/base"
	"github.com/skip-mev/block-sdk/testutils"
)

func TestDefaultTxPriority(t *testing.T) {
	acct := testutils.RandomAccounts(rand.New(rand.NewSource(1)), 1)
	txc := testutils.CreateTestEncodingConfig().TxConfig

	priority := base.DefaultTxPriority()

	t.Run("test getting a tx priority", func(t *testing.T) {
		tx, err := testutils.CreateTx(txc, acct[0], 0, 0, nil, sdk.NewCoin("stake", math.NewInt(1)))
		require.NoError(t, err)

		require.Equal(t, "1,stake", priority.GetTxPriority(nil, tx))
	})

	t.Run("test tx with no fee", func(t *testing.T) {
		tx, err := testutils.CreateTx(txc, acct[0], 0, 0, nil)
		require.NoError(t, err)

		require.Equal(t, "", priority.GetTxPriority(nil, tx))
	})

	t.Run("test comparing two tx priorities", func(t *testing.T) {
		tx1, err := testutils.CreateTx(txc, acct[0], 0, 0, nil, sdk.NewCoin("stake", math.NewInt(1)))
		require.NoError(t, err)

		tx2, err := testutils.CreateTx(txc, acct[0], 0, 0, nil, sdk.NewCoin("stake", math.NewInt(2)))
		require.NoError(t, err)

		priority1 := priority.GetTxPriority(nil, tx1)
		priority2 := priority.GetTxPriority(nil, tx2)

		require.Equal(t, -1, priority.Compare(priority1, priority2))
		require.Equal(t, 1, priority.Compare(priority2, priority1))
		require.Equal(t, 0, priority.Compare(priority2, priority2))
	})

	t.Run("test comparing two tx priorities with nil", func(t *testing.T) {
		tx1, err := testutils.CreateTx(txc, acct[0], 0, 0, nil, sdk.NewCoin("stake", math.NewInt(1)))
		require.NoError(t, err)

		priority1 := priority.GetTxPriority(nil, tx1)

		require.Equal(t, 1, priority.Compare(priority1, ""))
		require.Equal(t, -1, priority.Compare("", priority1))
		require.Equal(t, 0, priority.Compare("", ""))
	})

	t.Run("test with multiple fee coins", func(t *testing.T) {
		tx, err := testutils.CreateTx(txc, acct[0], 0, 0, nil, sdk.NewCoin("stake", math.NewInt(1)), sdk.NewCoin("atom", math.NewInt(2)))
		require.NoError(t, err)

		tx2, err := testutils.CreateTx(txc, acct[0], 0, 0, nil, sdk.NewCoin("stake", math.NewInt(2)), sdk.NewCoin("atom", math.NewInt(3)))
		require.NoError(t, err)

		priority1 := priority.GetTxPriority(nil, tx)
		priority2 := priority.GetTxPriority(nil, tx2)

		require.Equal(t, -1, priority.Compare(priority1, priority2))
		require.Equal(t, 1, priority.Compare(priority2, priority1))
		require.Equal(t, 0, priority.Compare(priority2, priority2))
	})

	t.Run("test with multiple different fee coins", func(t *testing.T) {
		tx1, err := testutils.CreateTx(txc, acct[0], 0, 0, nil, sdk.NewCoin("stake", math.NewInt(1)), sdk.NewCoin("atom", math.NewInt(2)), sdk.NewCoin("btc", math.NewInt(3)))
		require.NoError(t, err)

		tx2, err := testutils.CreateTx(txc, acct[0], 0, 0, nil, sdk.NewCoin("stake", math.NewInt(2)), sdk.NewCoin("eth", math.NewInt(3)), sdk.NewCoin("btc", math.NewInt(4)))
		require.NoError(t, err)

		priority1 := priority.GetTxPriority(nil, tx1)
		priority2 := priority.GetTxPriority(nil, tx2)

		require.Equal(t, 0, priority.Compare(priority1, priority2))
	})

	t.Run("one is nil, and the other isn't", func(t *testing.T) {
		tx1, err := testutils.CreateTx(txc, acct[0], 0, 0, nil, sdk.NewCoin("stake", math.NewInt(1)), sdk.NewCoin("atom", math.NewInt(2)), sdk.NewCoin("btc", math.NewInt(3)))
		require.NoError(t, err)

		tx2, err := testutils.CreateTx(txc, acct[0], 0, 0, nil)
		require.NoError(t, err)

		priority1 := priority.GetTxPriority(nil, tx1)
		priority2 := priority.GetTxPriority(nil, tx2)

		require.Equal(t, 1, priority.Compare(priority1, priority2))
		require.Equal(t, -1, priority.Compare(priority2, priority1))
		require.Equal(t, 0, priority.Compare(priority2, priority2))
	})
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
