package proposals_test

import (
	"math/rand"
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/skip-mev/block-sdk/block/proposals"
	"github.com/skip-mev/block-sdk/block/proposals/types"
	"github.com/skip-mev/block-sdk/block/utils"
	"github.com/skip-mev/block-sdk/block/utils/mocks"
	"github.com/skip-mev/block-sdk/testutils"
	"github.com/stretchr/testify/require"
)

func TestUpdateProposal(t *testing.T) {
	encodingConfig := testutils.CreateTestEncodingConfig()

	// Create a few random accounts
	random := rand.New(rand.NewSource(1))
	accounts := testutils.RandomAccounts(random, 5)

	lane := mocks.NewLane(t)

	lane.On("Name").Return("test").Maybe()
	lane.On("GetMaxBlockSpace").Return(math.LegacyNewDec(1)).Maybe()

	t.Run("can update with no transactions", func(t *testing.T) {
		proposal := proposals.NewProposal(nil, 100, 100)

		err := proposal.UpdateProposal(lane, nil)
		require.NoError(t, err)

		// Ensure that the proposal is empty.
		require.Equal(t, 0, len(proposal.Txs))
		require.Equal(t, int64(0), proposal.Info.BlockSize)
		require.Equal(t, uint64(0), proposal.Info.GasLimit)
		require.Equal(t, 0, len(proposal.Info.TxsByLane))

		block, err := proposal.GetProposalWithInfo()
		require.NoError(t, err)
		require.Equal(t, 1, len(block))
	})

	t.Run("can update with a single transaction", func(t *testing.T) {
		tx, err := testutils.CreateRandomTx(
			encodingConfig.TxConfig,
			accounts[0],
			0,
			1,
			0,
			100,
		)
		require.NoError(t, err)

		txBzs, err := utils.GetEncodedTxs(encodingConfig.TxConfig.TxEncoder(), []sdk.Tx{tx})
		require.NoError(t, err)

		size := len(txBzs[0])
		gasLimit := 100
		proposal := proposals.NewProposal(encodingConfig.TxConfig.TxEncoder(), int64(size), uint64(gasLimit))

		err = proposal.UpdateProposal(lane, []sdk.Tx{tx})
		require.NoError(t, err)

		// Ensure that the proposal is not empty.
		require.Equal(t, 1, len(proposal.Txs))
		require.Equal(t, int64(size), proposal.Info.BlockSize)
		require.Equal(t, uint64(gasLimit), proposal.Info.GasLimit)
		require.Equal(t, 1, len(proposal.Info.TxsByLane))
		require.Equal(t, uint64(1), proposal.Info.TxsByLane["test"])

		// Ensure that the proposal can be marshalled.
		block, err := proposal.GetProposalWithInfo()
		require.NoError(t, err)
		require.Equal(t, 2, len(block))
		require.Equal(t, txBzs[0], block[1])
	})

	t.Run("can update with multiple transactions", func(t *testing.T) {
		txs := make([]sdk.Tx, 0)

		for i := 0; i < 10; i++ {
			tx, err := testutils.CreateRandomTx(
				encodingConfig.TxConfig,
				accounts[0],
				0,
				uint64(i),
				0,
				100,
			)
			require.NoError(t, err)

			txs = append(txs, tx)
		}

		txBzs, err := utils.GetEncodedTxs(encodingConfig.TxConfig.TxEncoder(), txs)
		require.NoError(t, err)

		size := 0
		gasLimit := uint64(0)
		for _, txBz := range txBzs {
			size += len(txBz)
			gasLimit += 100
		}

		proposal := proposals.NewProposal(encodingConfig.TxConfig.TxEncoder(), int64(size), gasLimit)

		err = proposal.UpdateProposal(lane, txs)
		require.NoError(t, err)

		// Ensure that the proposal is not empty.
		require.Equal(t, len(txs), len(proposal.Txs))
		require.Equal(t, int64(size), proposal.Info.BlockSize)
		require.Equal(t, gasLimit, proposal.Info.GasLimit)
		require.Equal(t, uint64(10), proposal.Info.TxsByLane["test"])

		// Ensure that the proposal can be marshalled.
		block, err := proposal.GetProposalWithInfo()
		require.NoError(t, err)
		require.Equal(t, 11, len(block))

		for i := 0; i < 10; i++ {
			require.Equal(t, txBzs[i], block[i+1])
		}
	})

	t.Run("rejects an update with duplicate transactions", func(t *testing.T) {
		tx, err := testutils.CreateRandomTx(
			encodingConfig.TxConfig,
			accounts[0],
			0,
			1,
			0,
			100,
		)
		require.NoError(t, err)

		txBzs, err := utils.GetEncodedTxs(encodingConfig.TxConfig.TxEncoder(), []sdk.Tx{tx})
		require.NoError(t, err)

		size := int64(len(txBzs[0]))
		gasLimit := uint64(100)
		proposal := proposals.NewProposal(encodingConfig.TxConfig.TxEncoder(), size, gasLimit)

		err = proposal.UpdateProposal(lane, []sdk.Tx{tx})
		require.NoError(t, err)

		// Ensure that the proposal is empty.
		require.Equal(t, 1, len(proposal.Txs))
		require.Equal(t, size, proposal.Info.BlockSize)
		require.Equal(t, gasLimit, proposal.Info.GasLimit)
		require.Equal(t, 1, len(proposal.Info.TxsByLane))
		require.Equal(t, uint64(1), proposal.Info.TxsByLane["test"])

		otherlane := mocks.NewLane(t)

		otherlane.On("Name").Return("test").Maybe()
		otherlane.On("GetMaxBlockSpace").Return(math.LegacyNewDec(1)).Maybe()

		// Attempt to add the same transaction again.
		err = proposal.UpdateProposal(otherlane, []sdk.Tx{tx})
		require.Error(t, err)

		require.Equal(t, 1, len(proposal.Txs))
		require.Equal(t, size, proposal.Info.BlockSize)
		require.Equal(t, gasLimit, proposal.Info.GasLimit)
		require.Equal(t, 1, len(proposal.Info.TxsByLane))
		require.Equal(t, uint64(1), proposal.Info.TxsByLane["test"])

		// Ensure that the proposal can be marshalled.
		block, err := proposal.GetProposalWithInfo()
		require.NoError(t, err)
		require.Equal(t, 2, len(block))
		require.Equal(t, txBzs[0], block[1])
	})

	t.Run("rejects an update with duplicate lane updates", func(t *testing.T) {
		tx, err := testutils.CreateRandomTx(
			encodingConfig.TxConfig,
			accounts[0],
			0,
			1,
			0,
			100,
		)
		require.NoError(t, err)

		tx2, err := testutils.CreateRandomTx(
			encodingConfig.TxConfig,
			accounts[1],
			0,
			1,
			0,
			100,
		)
		require.NoError(t, err)

		txBzs, err := utils.GetEncodedTxs(encodingConfig.TxConfig.TxEncoder(), []sdk.Tx{tx, tx2})
		require.NoError(t, err)

		size := len(txBzs[0]) + len(txBzs[1])
		gasLimit := 200
		proposal := proposals.NewProposal(encodingConfig.TxConfig.TxEncoder(), int64(size), uint64(gasLimit))

		err = proposal.UpdateProposal(lane, []sdk.Tx{tx})
		require.NoError(t, err)

		err = proposal.UpdateProposal(lane, []sdk.Tx{tx2})
		require.Error(t, err)

		// Ensure that the proposal is not empty.
		require.Equal(t, 1, len(proposal.Txs))
		require.Equal(t, int64(len(txBzs[0])), proposal.Info.BlockSize)
		require.Equal(t, uint64(100), proposal.Info.GasLimit)
		require.Equal(t, 1, len(proposal.Info.TxsByLane))
		require.Equal(t, uint64(1), proposal.Info.TxsByLane["test"])

		// Ensure that the proposal can be marshalled.
		block, err := proposal.GetProposalWithInfo()
		require.NoError(t, err)
		require.Equal(t, 2, len(block))
		require.Equal(t, txBzs[0], block[1])
	})

	t.Run("rejects an update where lane limit is smaller (block size)", func(t *testing.T) {
		tx, err := testutils.CreateRandomTx(
			encodingConfig.TxConfig,
			accounts[0],
			0,
			1,
			0,
			100,
		)
		require.NoError(t, err)

		txBzs, err := utils.GetEncodedTxs(encodingConfig.TxConfig.TxEncoder(), []sdk.Tx{tx})
		require.NoError(t, err)

		size := len(txBzs[0])
		gasLimit := 100
		proposal := proposals.NewProposal(encodingConfig.TxConfig.TxEncoder(), int64(size), uint64(gasLimit))

		lane := mocks.NewLane(t)

		lane.On("Name").Return("test").Maybe()
		lane.On("GetMaxBlockSpace").Return(math.LegacyMustNewDecFromStr("0.5")).Maybe()

		err = proposal.UpdateProposal(lane, []sdk.Tx{tx})
		require.Error(t, err)

		// Ensure that the proposal is empty.
		require.Equal(t, 0, len(proposal.Txs))
		require.Equal(t, int64(0), proposal.Info.BlockSize)
		require.Equal(t, uint64(0), proposal.Info.GasLimit)
		require.Equal(t, 0, len(proposal.Info.TxsByLane))

		// Ensure that the proposal can be marshalled.
		block, err := proposal.GetProposalWithInfo()
		require.NoError(t, err)
		require.Equal(t, 1, len(block))
	})

	t.Run("rejects an update where the lane limit is smaller (gas limit)", func(t *testing.T) {
		tx, err := testutils.CreateRandomTx(
			encodingConfig.TxConfig,
			accounts[0],
			0,
			1,
			0,
			100,
		)
		require.NoError(t, err)

		txBzs, err := utils.GetEncodedTxs(encodingConfig.TxConfig.TxEncoder(), []sdk.Tx{tx})
		require.NoError(t, err)

		size := len(txBzs[0])
		gasLimit := 100
		proposal := proposals.NewProposal(encodingConfig.TxConfig.TxEncoder(), int64(size), uint64(gasLimit))

		lane := mocks.NewLane(t)

		lane.On("Name").Return("test").Maybe()
		lane.On("GetMaxBlockSpace").Return(math.LegacyMustNewDecFromStr("0.5")).Maybe()

		err = proposal.UpdateProposal(lane, []sdk.Tx{tx})
		require.Error(t, err)

		// Ensure that the proposal is empty.
		require.Equal(t, 0, len(proposal.Txs))
		require.Equal(t, int64(0), proposal.Info.BlockSize)
		require.Equal(t, 0, len(proposal.Info.TxsByLane))
		require.Equal(t, uint64(0), proposal.Info.GasLimit)

		// Ensure that the proposal can be marshalled.
		block, err := proposal.GetProposalWithInfo()
		require.NoError(t, err)
		require.Equal(t, 1, len(block))
	})

	t.Run("rejects an update where the proposal exceeds max block size", func(t *testing.T) {
		tx, err := testutils.CreateRandomTx(
			encodingConfig.TxConfig,
			accounts[0],
			0,
			1,
			0,
			100,
		)
		require.NoError(t, err)

		txBzs, err := utils.GetEncodedTxs(encodingConfig.TxConfig.TxEncoder(), []sdk.Tx{tx})
		require.NoError(t, err)

		size := len(txBzs[0])
		gasLimit := 100
		proposal := proposals.NewProposal(encodingConfig.TxConfig.TxEncoder(), int64(size)-1, uint64(gasLimit))

		err = proposal.UpdateProposal(lane, []sdk.Tx{tx})
		require.Error(t, err)

		// Ensure that the proposal is empty.
		require.Equal(t, 0, len(proposal.Txs))
		require.Equal(t, int64(0), proposal.Info.BlockSize)
		require.Equal(t, uint64(0), proposal.Info.GasLimit)
		require.Equal(t, 0, len(proposal.Info.TxsByLane))

		// Ensure that the proposal can be marshalled.
		block, err := proposal.GetProposalWithInfo()
		require.NoError(t, err)
		require.Equal(t, 1, len(block))
	})

	t.Run("rejects an update where the proposal exceeds max gas limit", func(t *testing.T) {
		tx, err := testutils.CreateRandomTx(
			encodingConfig.TxConfig,
			accounts[0],
			0,
			1,
			0,
			100,
		)
		require.NoError(t, err)

		txBzs, err := utils.GetEncodedTxs(encodingConfig.TxConfig.TxEncoder(), []sdk.Tx{tx})
		require.NoError(t, err)

		size := len(txBzs[0])
		gasLimit := 100
		proposal := proposals.NewProposal(encodingConfig.TxConfig.TxEncoder(), int64(size), uint64(gasLimit)-1)

		err = proposal.UpdateProposal(lane, []sdk.Tx{tx})
		require.Error(t, err)

		// Ensure that the proposal is empty.
		require.Equal(t, 0, len(proposal.Txs))
		require.Equal(t, int64(0), proposal.Info.BlockSize)
		require.Equal(t, uint64(0), proposal.Info.GasLimit)
		require.Equal(t, 0, len(proposal.Info.TxsByLane))

		// Ensure that the proposal can be marshalled.
		block, err := proposal.GetProposalWithInfo()
		require.NoError(t, err)
		require.Equal(t, 1, len(block))
	})

	t.Run("can add transactions from multiple lanes", func(t *testing.T) {
		tx, err := testutils.CreateRandomTx(
			encodingConfig.TxConfig,
			accounts[0],
			0,
			1,
			0,
			100,
		)
		require.NoError(t, err)

		tx2, err := testutils.CreateRandomTx(
			encodingConfig.TxConfig,
			accounts[1],
			0,
			1,
			0,
			100,
		)
		require.NoError(t, err)

		txBzs, err := utils.GetEncodedTxs(encodingConfig.TxConfig.TxEncoder(), []sdk.Tx{tx, tx2})
		require.NoError(t, err)

		proposal := proposals.NewProposal(encodingConfig.TxConfig.TxEncoder(), 10000, 10000)

		err = proposal.UpdateProposal(lane, []sdk.Tx{tx})
		require.NoError(t, err)

		otherlane := mocks.NewLane(t)
		otherlane.On("Name").Return("test2")
		otherlane.On("GetMaxBlockSpace").Return(math.LegacyMustNewDecFromStr("1.0"))

		err = proposal.UpdateProposal(otherlane, []sdk.Tx{tx2})
		require.NoError(t, err)

		size := len(txBzs[0]) + len(txBzs[1])
		gasLimit := 200

		// Ensure that the proposal is not empty.
		require.Equal(t, 2, len(proposal.Txs))
		require.Equal(t, int64(size), proposal.Info.BlockSize)
		require.Equal(t, uint64(gasLimit), proposal.Info.GasLimit)
		require.Equal(t, 2, len(proposal.Info.TxsByLane))
		require.Equal(t, uint64(1), proposal.Info.TxsByLane["test"])
		require.Equal(t, uint64(1), proposal.Info.TxsByLane["test2"])

		// Ensure that the proposal can be marshalled.
		block, err := proposal.GetProposalWithInfo()
		require.NoError(t, err)
		require.Equal(t, 3, len(block))
		require.Equal(t, txBzs[0], block[1])
		require.Equal(t, txBzs[1], block[2])
	})
}

func TestGetLaneLimits(t *testing.T) {
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
			proposal := proposals.Proposal{
				Info: types.ProposalInfo{
					MaxBlockSize: tc.maxTxBytes,
					BlockSize:    tc.totalTxBytesUsed,
					MaxGasLimit:  tc.maxGasLimit,
					GasLimit:     tc.totalGasLimitUsed,
				},
			}

			res := proposal.GetLaneLimits(tc.ratio)

			if res.MaxTxBytes != tc.expectedTxBytes {
				t.Errorf("expected tx bytes %d, got %d", tc.expectedTxBytes, res.MaxTxBytes)
			}

			if res.MaxGasLimit != tc.expectedGasLimit {
				t.Errorf("expected gas limit %d, got %d", tc.expectedGasLimit, res.MaxGasLimit)
			}
		})
	}
}
