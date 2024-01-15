package base

import (
	"fmt"
	"math/rand"
	"testing"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	signerextraction "github.com/skip-mev/block-sdk/adapters/signer_extraction_adapter"
	"github.com/skip-mev/block-sdk/testutils"
)

func TestMempoolComparison(t *testing.T) {
	acct := testutils.RandomAccounts(rand.New(rand.NewSource(1)), 2)
	txc := testutils.CreateTestEncodingConfig().TxConfig
	ctx := testutils.CreateBaseSDKContext(t)
	mp := NewMempool(
		DefaultTxPriority(),
		txc.TxEncoder(),
		signerextraction.NewDefaultAdapter(),
		PriorityNonceComparator(signerextraction.NewDefaultAdapter(), DefaultTxPriority()),
		1000,
	)
	t.Run("test same account, same nonce", func(t *testing.T) {
		tx1, err := testutils.CreateTx(txc, acct[0], 0, 0, nil, sdk.NewCoin("stake", sdkmath.NewInt(1)))
		require.NoError(t, err)
		tx2, err := testutils.CreateTx(txc, acct[0], 0, 0, nil, sdk.NewCoin("stake", sdkmath.NewInt(2)))
		require.NoError(t, err)
		output, err := mp.Compare(ctx, tx1, tx2)
		require.Error(t, err, fmt.Errorf("the two transactions have the same seqence number"))
		require.Equal(t, 0, output)
	})
	t.Run("test same account, tx1 gt amount, tx1 gt nonce", func(t *testing.T) {
		tx1, err := testutils.CreateTx(txc, acct[0], 1, 0, nil, sdk.NewCoin("stake", sdkmath.NewInt(2)))
		require.NoError(t, err)
		tx2, err := testutils.CreateTx(txc, acct[0], 0, 0, nil, sdk.NewCoin("stake", sdkmath.NewInt(1)))
		require.NoError(t, err)
		output, err := mp.Compare(ctx, tx1, tx2)
		require.NoError(t, err)
		require.Equal(t, -1, output)
	})
	t.Run("test same account, tx1 lt amount, tx1 gt nonce", func(t *testing.T) {
		tx1, err := testutils.CreateTx(txc, acct[0], 1, 0, nil, sdk.NewCoin("stake", sdkmath.NewInt(1)))
		require.NoError(t, err)
		tx2, err := testutils.CreateTx(txc, acct[0], 0, 0, nil, sdk.NewCoin("stake", sdkmath.NewInt(2)))
		require.NoError(t, err)
		output, err := mp.Compare(ctx, tx1, tx2)
		require.NoError(t, err)
		require.Equal(t, -1, output)
	})
	t.Run("test same account, tx1 lt amount, tx1 lt nonce", func(t *testing.T) {
		tx1, err := testutils.CreateTx(txc, acct[0], 0, 0, nil, sdk.NewCoin("stake", sdkmath.NewInt(1)))
		require.NoError(t, err)
		tx2, err := testutils.CreateTx(txc, acct[0], 1, 0, nil, sdk.NewCoin("stake", sdkmath.NewInt(2)))
		require.NoError(t, err)
		output, err := mp.Compare(ctx, tx1, tx2)
		require.NoError(t, err)
		require.Equal(t, 1, output)
	})
	t.Run("test diff account, tx1 lt amount, tx1 gt nonce", func(t *testing.T) {
		tx1, err := testutils.CreateTx(txc, acct[0], 1, 0, nil, sdk.NewCoin("stake", sdkmath.NewInt(1)))
		require.NoError(t, err)
		tx2, err := testutils.CreateTx(txc, acct[1], 0, 0, nil, sdk.NewCoin("stake", sdkmath.NewInt(2)))
		require.NoError(t, err)
		output, err := mp.Compare(ctx, tx1, tx2)
		require.NoError(t, err)
		require.Equal(t, -1, output)
	})
	t.Run("test diff account, tx1 lt amount, tx1 gt nonce, diff denoms", func(t *testing.T) {
		tx1, err := testutils.CreateTx(txc, acct[0], 1, 0, nil, sdk.NewCoin("nonstake", sdkmath.NewInt(1)))
		require.NoError(t, err)
		tx2, err := testutils.CreateTx(txc, acct[1], 0, 0, nil, sdk.NewCoin("stake", sdkmath.NewInt(2)))
		require.NoError(t, err)
		output, err := mp.Compare(ctx, tx1, tx2)
		require.NoError(t, err)
		require.Equal(t, 0, output)
	})
	t.Run("test diff account, tx1 gt amount, tx1 gt nonce, diff denoms", func(t *testing.T) {
		tx1, err := testutils.CreateTx(txc, acct[0], 1, 0, nil, sdk.NewCoin("nonstake", sdkmath.NewInt(2)))
		require.NoError(t, err)
		tx2, err := testutils.CreateTx(txc, acct[1], 0, 0, nil, sdk.NewCoin("stake", sdkmath.NewInt(1)))
		require.NoError(t, err)
		output, err := mp.Compare(ctx, tx1, tx2)
		require.NoError(t, err)
		require.Equal(t, 0, output)
	})
}
