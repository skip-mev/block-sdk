package base_test

import (
	"context"
	"math/rand"
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/skip-mev/block-sdk/block/base"
	"github.com/skip-mev/block-sdk/testutils"
	"github.com/stretchr/testify/require"
)

func TestNopTxPriority(t *testing.T) {
	txp := base.DefaultTxPriority()

	txc := testutils.CreateTestEncodingConfig().TxConfig
	accounts := testutils.RandomAccounts(rand.New(rand.NewSource(1)), 2)

	t.Run("tx priority is zero, regardless of fees", func(t *testing.T) {
		tx, err := testutils.CreateTx(txc, accounts[0], 0, 0, nil, sdk.NewCoin("stake", math.NewInt(1)))
		require.NoError(t, err)

		priority := txp.GetTxPriority(context.Background(), tx)
		require.Equal(t, 0, priority)
	})

	t.Run("tx priorities are always equal", func(t *testing.T) {
		tx1, err := testutils.CreateTx(txc, accounts[0], 0, 0, nil, sdk.NewCoin("stake", math.NewInt(1)))
		require.NoError(t, err)

		tx2, err := testutils.CreateTx(txc, accounts[1], 0, 0, nil, sdk.NewCoin("stake", math.NewInt(2)))
		require.NoError(t, err)

		priority1 := txp.GetTxPriority(context.Background(), tx1)
		priority2 := txp.GetTxPriority(context.Background(), tx2)

		require.Equal(t, priority1, priority2)
	})
}
