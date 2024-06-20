package utils_test

import (
	"math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/skip-mev/block-sdk/v2/block/utils"
	"github.com/skip-mev/block-sdk/v2/testutils"
)

var (
	numAccounts   = 5
	numTxsPerAcct = 100
	cacheSize     = 500
)

func BenchmarkCacheDecoding(b *testing.B) {
	encodingCfg := testutils.CreateTestEncodingConfig()
	decoder := encodingCfg.TxConfig.TxDecoder()

	random := rand.New(rand.NewSource(time.Now().Unix()))
	account := testutils.RandomAccounts(random, numAccounts)

	txs := make([][]byte, numAccounts*numTxsPerAcct)
	for i := 0; i < numAccounts; i++ {
		for j := 0; j < numTxsPerAcct; j++ {
			txBytes, err := testutils.CreateRandomTxBz(
				encodingCfg.TxConfig,
				account[i],
				uint64(j),
				1,
				2,
				0,
			)
			require.NoError(b, err)

			txs[i*numTxsPerAcct+j] = txBytes
		}
	}

	cacheTxDecoder, err := utils.NewCacheTxDecoder(decoder, uint64(cacheSize))
	require.NoError(b, err)
	decoder = cacheTxDecoder.TxDecoder()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		for _, txBytes := range txs {
			_, err := decoder(txBytes)
			require.NoError(b, err)
		}

		for _, txBytes := range txs {
			_, err := decoder(txBytes)
			require.NoError(b, err)
		}
	}
}

func BenchmarkStandardDecoding(b *testing.B) {
	encodingCfg := testutils.CreateTestEncodingConfig()
	decoder := encodingCfg.TxConfig.TxDecoder()

	random := rand.New(rand.NewSource(time.Now().Unix()))
	account := testutils.RandomAccounts(random, numAccounts)

	txs := make([][]byte, numAccounts*numTxsPerAcct)
	for i := 0; i < numAccounts; i++ {
		for j := 0; j < numTxsPerAcct; j++ {
			txBytes, err := testutils.CreateRandomTxBz(
				encodingCfg.TxConfig,
				account[i],
				uint64(j),
				1,
				2,
				0,
			)
			require.NoError(b, err)

			txs[i*numTxsPerAcct+j] = txBytes
		}
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		for _, txBytes := range txs {
			_, err := decoder(txBytes)
			require.NoError(b, err)
		}

		for _, txBytes := range txs {
			_, err := decoder(txBytes)
			require.NoError(b, err)
		}
	}
}

func TestNewCacheTxDecoder(t *testing.T) {
	encodingCfg := testutils.CreateTestEncodingConfig()
	decoder := encodingCfg.TxConfig.TxDecoder()

	_, err := utils.NewDefaultCacheTxDecoder(decoder)
	require.NoError(t, err)

	_, err = utils.NewCacheTxDecoder(decoder, 100)
	require.NoError(t, err)

	_, err = utils.NewCacheTxDecoder(nil, 100)
	require.Error(t, err)
}

func TestDecode(t *testing.T) {
	encodingCfg := testutils.CreateTestEncodingConfig()
	decoder := encodingCfg.TxConfig.TxDecoder()

	random := rand.New(rand.NewSource(time.Now().Unix()))
	account := testutils.RandomAccounts(random, 1)

	t.Run("decode valid tx and check that it is cached", func(t *testing.T) {
		txBytes, err := testutils.CreateRandomTxBz(
			encodingCfg.TxConfig,
			account[0],
			0,
			1,
			1,
			0,
		)
		require.NoError(t, err)

		cacheTxDecoder, err := utils.NewDefaultCacheTxDecoder(decoder)
		require.NoError(t, err)

		decoder := cacheTxDecoder.TxDecoder()

		tx, err := decoder(txBytes)
		require.NoError(t, err)
		require.NotNil(t, tx)
		require.Equal(t, 1, cacheTxDecoder.Len())
		require.True(t, cacheTxDecoder.Contains(txBytes))

		// decode the same tx again
		tx, err = decoder(txBytes)
		require.NoError(t, err)
		require.NotNil(t, tx)
		require.Equal(t, 1, cacheTxDecoder.Len())
		require.True(t, cacheTxDecoder.Contains(txBytes))
	})

	t.Run("decode invalid tx", func(t *testing.T) {
		cacheTxDecoder, err := utils.NewDefaultCacheTxDecoder(decoder)
		require.NoError(t, err)

		decoder := cacheTxDecoder.TxDecoder()
		tx, err := decoder([]byte("invalid tx"))
		require.Error(t, err)
		require.Nil(t, tx)
		require.Equal(t, 0, cacheTxDecoder.Len())
	})

	t.Run("decode multiple txs without hitting limit", func(t *testing.T) {
		cacheTxDecoder, err := utils.NewCacheTxDecoder(decoder, 100)
		require.NoError(t, err)

		for i := 0; i < 100; i++ {
			txBytes, err := testutils.CreateRandomTxBz(
				encodingCfg.TxConfig,
				account[0],
				uint64(i),
				1,
				1,
				0,
			)
			require.NoError(t, err)

			decoder := cacheTxDecoder.TxDecoder()
			tx, err := decoder(txBytes)
			require.NoError(t, err)
			require.NotNil(t, tx)
			require.Equal(t, i+1, cacheTxDecoder.Len())
			require.True(t, cacheTxDecoder.Contains(txBytes))
		}
		require.Equal(t, 100, cacheTxDecoder.Len())
	})

	t.Run("decode multiple txs hitting limit", func(t *testing.T) {
		maxSize := uint64(2)
		cacheTxDecoder, err := utils.NewCacheTxDecoder(decoder, maxSize)
		require.NoError(t, err)

		for i := 0; i < int(maxSize*3); i++ {
			txBytes, err := testutils.CreateRandomTxBz(
				encodingCfg.TxConfig,
				account[0],
				uint64(i),
				1,
				1,
				0,
			)
			require.NoError(t, err)

			decoder := cacheTxDecoder.TxDecoder()
			tx, err := decoder(txBytes)
			require.NoError(t, err)
			require.NotNil(t, tx)
			require.True(t, cacheTxDecoder.Contains(txBytes))

			if i < int(maxSize) {
				require.Equal(t, i+1, cacheTxDecoder.Len())
			} else {
				require.Equal(t, int(maxSize), cacheTxDecoder.Len())
			}
		}
		require.Equal(t, int(maxSize), cacheTxDecoder.Len())
	})
}
