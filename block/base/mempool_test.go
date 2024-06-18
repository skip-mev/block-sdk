package base_test

import (
	"fmt"
	"math/rand"
	"testing"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	signerextraction "github.com/skip-mev/block-sdk/v2/adapters/signer_extraction_adapter"
	"github.com/skip-mev/block-sdk/v2/block/base"
	"github.com/skip-mev/block-sdk/v2/testutils"
)

type txGen struct {
	acc    testutils.Account
	nonce  uint64
	amount sdk.Coin
}

var (
	numAccounts   = 10
	numTxsPerAcct = 100
)

func BenchmarkContains(b *testing.B) {
	acct := testutils.RandomAccounts(rand.New(rand.NewSource(1)), numAccounts)
	txc := testutils.CreateTestEncodingConfig().TxConfig

	mp := base.NewMempool(
		base.DefaultTxPriority(),
		signerextraction.NewDefaultAdapter(),
		1000,
	)

	txs := make([]sdk.Tx, numAccounts*numTxsPerAcct)
	for i := 0; i < numAccounts; i++ {
		for j := 0; j < numTxsPerAcct; j++ {
			tx, err := testutils.CreateTx(txc, acct[i], uint64(j), 0, nil, sdk.NewCoin("stake", sdkmath.NewInt(1)))
			require.NoError(b, err)
			err = mp.Insert(sdk.Context{}, tx)
			require.NoError(b, err)
			txs[i*numTxsPerAcct+j] = tx
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, tx := range txs {
			found := mp.Contains(tx)
			if !found {
				b.Fatalf("tx not found in mempool")
			}
		}
	}
}

func TestMempoolComparison(t *testing.T) {
	acct := testutils.RandomAccounts(rand.New(rand.NewSource(1)), 2)
	txc := testutils.CreateTestEncodingConfig().TxConfig
	ctx := testutils.CreateBaseSDKContext(t)
	mp := base.NewMempool(
		base.DefaultTxPriority(),
		signerextraction.NewDefaultAdapter(),
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
		require.Equal(t, 0, output)
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

func TestMempoolSelect(t *testing.T) {
	acct := testutils.RandomAccounts(rand.New(rand.NewSource(1)), 2)
	txc := testutils.CreateTestEncodingConfig().TxConfig
	ctx := testutils.CreateBaseSDKContext(t)
	se := signerextraction.NewDefaultAdapter()
	mp := base.NewMempool(
		base.DefaultTxPriority(),
		se,
		1000,
	)
	tests := []struct {
		name     string
		inputs   []txGen
		expected []txGen
	}{
		{
			name: "test1",
			inputs: []txGen{
				{
					acc:    acct[0],
					amount: sdk.NewCoin("stake", sdkmath.NewInt(1)),
					nonce:  0,
				},
				{
					acc:    acct[0],
					amount: sdk.NewCoin("notstake", sdkmath.NewInt(2)),
					nonce:  1,
				},
			},
			expected: []txGen{
				{
					acc:    acct[0],
					amount: sdk.NewCoin("stake", sdkmath.NewInt(1)),
					nonce:  0,
				},
				{
					acc:    acct[0],
					amount: sdk.NewCoin("notstake", sdkmath.NewInt(2)),
					nonce:  1,
				},
			},
		},
	}
	for _, tc := range tests {
		// insert all input txs
		t.Run(tc.name, func(t *testing.T) {
			for _, tx := range tc.inputs {
				inputTx, err := testutils.CreateTx(txc, tx.acc, tx.nonce, 0, nil, tx.amount)
				require.NoError(t, err)
				err = mp.Insert(ctx, inputTx)
				require.NoError(t, err)
			}
		})
		// extract all txs via select
		var output []sdk.Tx
		for iter := mp.Select(ctx, nil); iter != nil; iter = iter.Next() {
			output = append(output, iter.Tx())
		}
		// validate the order matches the expected order
		require.Equal(t, len(tc.expected), len(output))
		for i, tx := range output {

			sigs, err := se.GetSigners(tx)
			require.NoError(t, err)
			require.Equal(t, 1, len(sigs))
			require.Equal(t, tc.expected[i].acc.Address, sigs[0].Signer)
			feeTx, ok := tx.(sdk.FeeTx)
			require.True(t, ok)
			require.Equal(t, tc.expected[i].amount.Denom, feeTx.GetFee().GetDenomByIndex(0))
			require.Equal(t, tc.expected[i].amount.Amount, feeTx.GetFee().AmountOf(feeTx.GetFee().GetDenomByIndex(0)))
			require.Equal(t, tc.expected[i].nonce, sigs[0].Sequence)

		}
	}
}
