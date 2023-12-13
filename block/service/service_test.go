package service_test

import (
	"context"
	"math/rand"
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/stretchr/testify/require"

	"github.com/skip-mev/block-sdk/block"
	"github.com/skip-mev/block-sdk/block/service"
	"github.com/skip-mev/block-sdk/block/service/types"
	"github.com/skip-mev/block-sdk/lanes/base"
	"github.com/skip-mev/block-sdk/lanes/free"
	"github.com/skip-mev/block-sdk/lanes/mev"
	"github.com/skip-mev/block-sdk/testutils"
)

func TestGetTxDistribution(t *testing.T) {
	config := testutils.CreateTestEncodingConfig()
	accounts := testutils.RandomAccounts(rand.New(rand.NewSource(1)), 3)
	ctx := testutils.CreateBaseSDKContext(t)

	testCases := []struct {
		name                 string
		mempool              func() *block.LanedMempool
		expectedDistribution map[string]uint64
	}{
		{
			name:    "returns correct distribution with no transactions",
			mempool: testutils.CreateMempool,
			expectedDistribution: map[string]uint64{
				mev.LaneName:  0,
				free.LaneName: 0,
				base.LaneName: 0,
			},
		},
		{
			name: "only default lane has transactions",
			mempool: func() *block.LanedMempool {
				tx1, err := testutils.CreateRandomTx(
					config.TxConfig,
					accounts[0],
					0,
					1,
					0,
					0,
					sdk.NewCoin("skip", math.NewInt(1)),
				)
				require.NoError(t, err)

				tx2, err := testutils.CreateRandomTx(
					config.TxConfig,
					accounts[1],
					0,
					1,
					0,
					0,
					sdk.NewCoin("skip", math.NewInt(1)),
				)
				require.NoError(t, err)

				mempool := testutils.CreateMempool()
				err = mempool.Insert(ctx, tx1)
				require.NoError(t, err)
				err = mempool.Insert(ctx, tx2)
				require.NoError(t, err)

				return mempool
			},
			expectedDistribution: map[string]uint64{
				mev.LaneName:  0,
				free.LaneName: 0,
				base.LaneName: 2,
			},
		},
		{
			name: "only free lane has transactions",
			mempool: func() *block.LanedMempool {
				tx1, err := testutils.CreateFreeTx(
					config.TxConfig,
					accounts[0],
					0,
					1,
					"skip",
					sdk.NewCoin("skip", math.NewInt(1)),
				)
				require.NoError(t, err)

				mempool := testutils.CreateMempool()
				err = mempool.Insert(ctx, tx1)
				require.NoError(t, err)

				return mempool
			},
			expectedDistribution: map[string]uint64{
				mev.LaneName:  0,
				free.LaneName: 1,
				base.LaneName: 0,
			},
		},
		{
			name: "only mev lane has transactions",
			mempool: func() *block.LanedMempool {
				tx1, err := testutils.CreateAuctionTxWithSigners(
					config.TxConfig,
					accounts[0],
					sdk.NewCoin("skip", math.NewInt(1)),
					0,
					0,
					accounts,
				)
				require.NoError(t, err)

				mempool := testutils.CreateMempool()
				err = mempool.Insert(ctx, tx1)
				require.NoError(t, err)

				return mempool
			},
			expectedDistribution: map[string]uint64{
				mev.LaneName:  1,
				free.LaneName: 0,
				base.LaneName: 0,
			},
		},
		{
			name: "all lanes have transactions",
			mempool: func() *block.LanedMempool {
				mevTx, err := testutils.CreateAuctionTxWithSigners(
					config.TxConfig,
					accounts[0],
					sdk.NewCoin("skip", math.NewInt(1)),
					0,
					0,
					accounts,
				)
				require.NoError(t, err)

				freeTx, err := testutils.CreateFreeTx(
					config.TxConfig,
					accounts[0],
					0,
					1,
					"skip",
					sdk.NewCoin("skip", math.NewInt(1)),
				)
				require.NoError(t, err)

				baseTx, err := testutils.CreateRandomTx(
					config.TxConfig,
					accounts[0],
					0,
					1,
					0,
					0,
					sdk.NewCoin("skip", math.NewInt(1)),
				)
				require.NoError(t, err)

				mempool := testutils.CreateMempool()
				err = mempool.Insert(ctx, mevTx)
				require.NoError(t, err)
				err = mempool.Insert(ctx, freeTx)
				require.NoError(t, err)
				err = mempool.Insert(ctx, baseTx)
				require.NoError(t, err)

				return mempool
			},
			expectedDistribution: map[string]uint64{
				mev.LaneName:  1,
				free.LaneName: 1,
				base.LaneName: 1,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mempool := tc.mempool()
			queryService := service.NewQueryService(mempool)
			ctx := context.Background()

			distributionResponse, err := queryService.GetTxDistribution(ctx, &types.GetTxDistributionRequest{})
			require.NoError(t, err)
			require.Equal(t, tc.expectedDistribution, distributionResponse.Distribution)
		})
	}
}
