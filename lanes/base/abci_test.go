package base_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/rand"

	"cosmossdk.io/log"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	signer_extraction "github.com/skip-mev/block-sdk/v2/adapters/signer_extraction_adapter"
	"github.com/skip-mev/block-sdk/v2/block"
	"github.com/skip-mev/block-sdk/v2/block/base"
	"github.com/skip-mev/block-sdk/v2/block/mocks"
	"github.com/skip-mev/block-sdk/v2/block/proposals"
	"github.com/skip-mev/block-sdk/v2/block/utils"
	defaultlane "github.com/skip-mev/block-sdk/v2/lanes/base"
	testutils "github.com/skip-mev/block-sdk/v2/testutils"
)

func (s *BaseTestSuite) TestPrepareLane() {
	s.Run("should not build a proposal when amount configured to lane is too small", func() {
		// Create a basic transaction that should not in the proposal
		tx, err := testutils.CreateRandomTx(
			s.encodingConfig.TxConfig,
			s.accounts[0],
			0,
			1,
			0,
			1,
			sdk.NewCoin(s.gasTokenDenom, math.NewInt(1)),
		)
		s.Require().NoError(err)

		expectedExecution := map[sdk.Tx]bool{
			tx: true,
		}
		lane := s.initLane(math.LegacyMustNewDecFromStr("0.5"), expectedExecution)
		s.Require().NoError(lane.Insert(s.ctx, tx))
		s.Require().Equal(1, lane.CountTx())

		txBz, err := s.encodingConfig.TxConfig.TxEncoder()(tx)
		s.Require().NoError(err)

		emptyProposal := proposals.NewProposal(
			log.NewNopLogger(),
			int64(len(txBz)),
			1,
		)

		finalProposal, err := lane.PrepareLane(s.ctx, emptyProposal, block.NoOpPrepareLanesHandler())
		s.Require().NoError(err)

		// Ensure the proposal is empty
		s.Require().Equal(0, len(finalProposal.Txs))
		s.Require().Equal(int64(0), finalProposal.Info.BlockSize)
		s.Require().Equal(uint64(0), finalProposal.Info.GasLimit)
		s.Require().Equal(0, lane.CountTx())
		s.Require().False(lane.Contains(tx))
	})

	s.Run("should not build a proposal when gas configured to lane is too small", func() {
		// Create a basic transaction that should not in the proposal
		tx, err := testutils.CreateRandomTx(
			s.encodingConfig.TxConfig,
			s.accounts[0],
			0,
			1,
			0,
			10,
			sdk.NewCoin(s.gasTokenDenom, math.NewInt(1)),
		)
		s.Require().NoError(err)

		expectedExecution := map[sdk.Tx]bool{
			tx: true,
		}
		lane := s.initLane(math.LegacyMustNewDecFromStr("0.5"), expectedExecution)

		// Insert the transaction into the lane
		s.Require().NoError(lane.Insert(s.ctx, tx))
		s.Require().Equal(1, lane.CountTx())

		txBz, err := s.encodingConfig.TxConfig.TxEncoder()(tx)
		s.Require().NoError(err)

		limit := proposals.LaneLimits{
			MaxTxBytes:  int64(len(txBz)) * 10,
			MaxGasLimit: 10,
		}
		emptyProposal := proposals.NewProposal(
			log.NewNopLogger(),
			limit.MaxTxBytes,
			limit.MaxGasLimit,
		)

		finalProposal, err := lane.PrepareLane(s.ctx, emptyProposal, block.NoOpPrepareLanesHandler())
		s.Require().NoError(err)

		// Ensure the proposal is empty
		s.Require().Equal(0, len(finalProposal.Txs))
		s.Require().Equal(int64(0), finalProposal.Info.BlockSize)
		s.Require().Equal(uint64(0), finalProposal.Info.GasLimit)
		s.Require().Equal(0, lane.CountTx())
		s.Require().False(lane.Contains(tx))
	})

	s.Run("should not build a proposal when gas configured to lane is too small p2", func() {
		// Create a basic transaction that should not in the proposal
		tx, err := testutils.CreateRandomTx(
			s.encodingConfig.TxConfig,
			s.accounts[0],
			0,
			1,
			0,
			10,
			sdk.NewCoin(s.gasTokenDenom, math.NewInt(1)),
		)
		s.Require().NoError(err)

		expectedExecution := map[sdk.Tx]bool{
			tx: true,
		}
		lane := s.initLane(math.LegacyMustNewDecFromStr("0.5"), expectedExecution)

		// Insert the transaction into the lane
		s.Require().NoError(lane.Insert(s.ctx, tx))

		txBz, err := s.encodingConfig.TxConfig.TxEncoder()(tx)
		s.Require().NoError(err)

		// Create a proposal
		limit := proposals.LaneLimits{
			MaxTxBytes:  int64(len(txBz)) * 10, // have enough space for 10 of these txs
			MaxGasLimit: 10,
		}
		emptyProposal := proposals.NewProposal(
			log.NewNopLogger(),
			limit.MaxTxBytes,
			limit.MaxGasLimit,
		)

		finalProposal, err := lane.PrepareLane(s.ctx, emptyProposal, block.NoOpPrepareLanesHandler())
		s.Require().NoError(err)

		// Ensure the proposal is empty
		s.Require().Equal(0, len(finalProposal.Txs))
		s.Require().Equal(int64(0), finalProposal.Info.BlockSize)
		s.Require().Equal(uint64(0), finalProposal.Info.GasLimit)
		s.Require().Equal(0, lane.CountTx())
	})

	s.Run("should be able to build a proposal with a tx that just fits in", func() {
		// Create a basic transaction that should just fit in with the gas limit
		// and max size
		tx, err := testutils.CreateRandomTx(
			s.encodingConfig.TxConfig,
			s.accounts[0],
			0,
			1,
			0,
			10,
			sdk.NewCoin(s.gasTokenDenom, math.NewInt(1)),
		)
		s.Require().NoError(err)

		expectedExecution := map[sdk.Tx]bool{
			tx: true,
		}
		lane := s.initLane(math.LegacyOneDec(), expectedExecution)

		s.Require().NoError(lane.Insert(s.ctx, tx))

		txBz, err := s.encodingConfig.TxConfig.TxEncoder()(tx)
		s.Require().NoError(err)

		limit := proposals.LaneLimits{
			MaxTxBytes:  int64(len(txBz)),
			MaxGasLimit: 10,
		}
		emptyProposal := proposals.NewProposal(
			log.NewNopLogger(),
			limit.MaxTxBytes,
			limit.MaxGasLimit,
		)

		finalProposal, err := lane.PrepareLane(s.ctx, emptyProposal, block.NoOpPrepareLanesHandler())
		s.Require().NoError(err)

		// Ensure the proposal is not empty and contains the transaction
		s.Require().Equal(1, len(finalProposal.Txs))
		s.Require().Equal(limit.MaxTxBytes, finalProposal.Info.BlockSize)
		s.Require().Equal(uint64(10), finalProposal.Info.GasLimit)
		s.Require().Equal(txBz, finalProposal.Txs[0])
	})

	s.Run("should not build a proposal with a that fails verify tx", func() {
		// Create a basic transaction that should not in the proposal
		tx, err := testutils.CreateRandomTx(
			s.encodingConfig.TxConfig,
			s.accounts[0],
			0,
			1,
			0,
			10,
			sdk.NewCoin(s.gasTokenDenom, math.NewInt(1)),
		)
		s.Require().NoError(err)

		// We expect the transaction to fail verify tx
		expectedExecution := map[sdk.Tx]bool{
			tx: false,
		}
		lane := s.initLane(math.LegacyOneDec(), expectedExecution)

		s.Require().NoError(lane.Insert(s.ctx, tx))

		txBz, err := s.encodingConfig.TxConfig.TxEncoder()(tx)
		s.Require().NoError(err)

		emptyProposal := proposals.NewProposal(
			log.NewNopLogger(),
			int64(len(txBz)),
			10,
		)

		finalProposal, err := lane.PrepareLane(s.ctx, emptyProposal, block.NoOpPrepareLanesHandler())
		s.Require().NoError(err)

		// Ensure the proposal is empty
		s.Require().Equal(0, len(finalProposal.Txs))
		s.Require().Equal(int64(0), finalProposal.Info.BlockSize)
		s.Require().Equal(uint64(0), finalProposal.Info.GasLimit)

		// Ensure the transaction is removed from the lane
		s.Require().False(lane.Contains(tx))
		s.Require().Equal(0, lane.CountTx())
	})

	s.Run("should order transactions correctly in the proposal", func() {
		tx1, err := testutils.CreateRandomTx(
			s.encodingConfig.TxConfig,
			s.accounts[0],
			0,
			1,
			0,
			10,
			sdk.NewCoin(s.gasTokenDenom, math.NewInt(2)),
		)
		s.Require().NoError(err)

		tx2, err := testutils.CreateRandomTx(
			s.encodingConfig.TxConfig,
			s.accounts[1],
			0,
			1,
			0,
			10,
			sdk.NewCoin(s.gasTokenDenom, math.NewInt(1)),
		)
		s.Require().NoError(err)

		expectedExecution := map[sdk.Tx]bool{
			tx1: true,
			tx2: true,
		}
		lane := s.initLane(math.LegacyOneDec(), expectedExecution)

		s.Require().NoError(lane.Insert(s.ctx, tx1))
		s.Require().NoError(lane.Insert(s.ctx, tx2))

		txBz1, err := s.encodingConfig.TxConfig.TxEncoder()(tx1)
		s.Require().NoError(err)

		txBz2, err := s.encodingConfig.TxConfig.TxEncoder()(tx2)
		s.Require().NoError(err)

		size := int64(len(txBz1)) + int64(len(txBz2))
		gasLimit := uint64(20)
		emptyProposal := proposals.NewProposal(
			log.NewNopLogger(),
			size,
			gasLimit,
		)

		finalProposal, err := lane.PrepareLane(s.ctx, emptyProposal, block.NoOpPrepareLanesHandler())
		s.Require().NoError(err)

		// Ensure the proposal is ordered correctly
		s.Require().Equal(2, len(finalProposal.Txs))
		s.Require().Equal(size, finalProposal.Info.BlockSize)
		s.Require().Equal(gasLimit, finalProposal.Info.GasLimit)
		s.Require().Equal([][]byte{txBz1, txBz2}, finalProposal.Txs)
	})

	s.Run("should order transactions correctly in the proposal (with different insertion)", func() {
		tx1, err := testutils.CreateRandomTx(
			s.encodingConfig.TxConfig,
			s.accounts[1],
			1,
			1,
			0,
			1,
			sdk.NewCoin(s.gasTokenDenom, math.NewInt(1)),
		)
		s.Require().NoError(err)

		tx2, err := testutils.CreateRandomTx(
			s.encodingConfig.TxConfig,
			s.accounts[1],
			0,
			1,
			0,
			1,
			sdk.NewCoin(s.gasTokenDenom, math.NewInt(2)),
		)
		s.Require().NoError(err)

		expectedExecution := map[sdk.Tx]bool{
			tx1: true,
			tx2: true,
		}
		lane := s.initLane(math.LegacyOneDec(), expectedExecution)

		s.Require().NoError(lane.Insert(s.ctx, tx1))
		s.Require().NoError(lane.Insert(s.ctx, tx2))

		txBz1, err := s.encodingConfig.TxConfig.TxEncoder()(tx1)
		s.Require().NoError(err)

		txBz2, err := s.encodingConfig.TxConfig.TxEncoder()(tx2)
		s.Require().NoError(err)

		size := int64(len(txBz1)) + int64(len(txBz2))
		gasLimit := uint64(2)
		emptyProposal := proposals.NewProposal(
			log.NewNopLogger(),
			size,
			gasLimit,
		)

		finalProposal, err := lane.PrepareLane(s.ctx, emptyProposal, block.NoOpPrepareLanesHandler())
		s.Require().NoError(err)

		// Ensure the proposal is ordered correctly
		s.Require().Equal(2, len(finalProposal.Txs))
		s.Require().Equal(size, finalProposal.Info.BlockSize)
		s.Require().Equal(gasLimit, finalProposal.Info.GasLimit)
		s.Require().Equal([][]byte{txBz2, txBz1}, finalProposal.Txs)
	})

	s.Run("should include tx that fits in proposal when other does not", func() {
		// Create a basic transaction that should not in the proposal
		tx1, err := testutils.CreateRandomTx(
			s.encodingConfig.TxConfig,
			s.accounts[0],
			0,
			1,
			0,
			2,
			sdk.NewCoin(s.gasTokenDenom, math.NewInt(1)),
		)
		s.Require().NoError(err)

		tx2, err := testutils.CreateRandomTx(
			s.encodingConfig.TxConfig,
			s.accounts[1],
			0,
			10, // This tx is too large to fit in the proposal
			0,
			1,
			sdk.NewCoin(s.gasTokenDenom, math.NewInt(1)),
		)
		s.Require().NoError(err)

		// Create a lane with a max block space of 1 but a proposal that is smaller than the tx
		expectedExecution := map[sdk.Tx]bool{
			tx1: true,
			tx2: true,
		}
		lane := s.initLane(math.LegacyOneDec(), expectedExecution)

		// Insert the transaction into the lane
		s.Require().NoError(lane.Insert(s.ctx.WithPriority(10), tx1))
		s.Require().NoError(lane.Insert(s.ctx.WithPriority(5), tx2))

		txBz1, err := s.encodingConfig.TxConfig.TxEncoder()(tx1)
		s.Require().NoError(err)

		txBz2, err := s.encodingConfig.TxConfig.TxEncoder()(tx2)
		s.Require().NoError(err)

		size := int64(len(txBz1)) + int64(len(txBz2)) - 1
		gasLimit := uint64(3)
		emptyProposal := proposals.NewProposal(
			log.NewNopLogger(),
			size,
			gasLimit,
		)

		finalProposal, err := lane.PrepareLane(s.ctx, emptyProposal, block.NoOpPrepareLanesHandler())
		s.Require().NoError(err)

		// Ensure the proposal is ordered correctly
		s.Require().Equal(1, len(finalProposal.Txs))
		s.Require().Equal(int64(len(txBz1)), finalProposal.Info.BlockSize)
		s.Require().Equal(uint64(2), finalProposal.Info.GasLimit)
		s.Require().Equal([][]byte{txBz1}, finalProposal.Txs)
	})

	s.Run("should include tx that consumes all gas in proposal while other cannot", func() {
		// Create a basic transaction that should not in the proposal
		tx1, err := testutils.CreateRandomTx(
			s.encodingConfig.TxConfig,
			s.accounts[0],
			0,
			1,
			0,
			2,
			sdk.NewCoin(s.gasTokenDenom, math.NewInt(1)),
		)
		s.Require().NoError(err)

		tx2, err := testutils.CreateRandomTx(
			s.encodingConfig.TxConfig,
			s.accounts[1],
			0,
			10, // This tx is too large to fit in the proposal
			0,
			1,
			sdk.NewCoin(s.gasTokenDenom, math.NewInt(1)),
		)
		s.Require().NoError(err)

		// Create a lane with a max block space of 1 but a proposal that is smaller than the tx
		expectedExecution := map[sdk.Tx]bool{
			tx1: true,
			tx2: true,
		}
		lane := s.initLane(math.LegacyOneDec(), expectedExecution)

		// Insert the transaction into the lane
		s.Require().NoError(lane.Insert(s.ctx, tx1))
		s.Require().NoError(lane.Insert(s.ctx, tx2))

		txBz1, err := s.encodingConfig.TxConfig.TxEncoder()(tx1)
		s.Require().NoError(err)

		txBz2, err := s.encodingConfig.TxConfig.TxEncoder()(tx2)
		s.Require().NoError(err)

		size := int64(len(txBz1)) + int64(len(txBz2)) - 1
		gasLimit := uint64(1)
		emptyProposal := proposals.NewProposal(
			log.NewNopLogger(),
			size,
			gasLimit,
		)

		finalProposal, err := lane.PrepareLane(s.ctx, emptyProposal, block.NoOpPrepareLanesHandler())
		s.Require().NoError(err)

		// Ensure the proposal is ordered correctly
		s.Require().Equal(1, len(finalProposal.Txs))
		s.Require().Equal(int64(len(txBz2)), finalProposal.Info.BlockSize)
		s.Require().Equal(uint64(1), finalProposal.Info.GasLimit)
		s.Require().Equal([][]byte{txBz2}, finalProposal.Txs)
	})

	s.Run("should not attempt to include transaction already included in the proposal", func() {
		// Create a basic transaction that should not in the proposal
		tx, err := testutils.CreateRandomTx(
			s.encodingConfig.TxConfig,
			s.accounts[0],
			0,
			1,
			0,
			2,
			sdk.NewCoin(s.gasTokenDenom, math.NewInt(1)),
		)
		s.Require().NoError(err)

		// Create a lane with a max block space of 1 but a proposal that is smaller than the tx
		expectedExecution := map[sdk.Tx]bool{
			tx: true,
		}
		lane := s.initLane(math.LegacyOneDec(), expectedExecution)

		// Insert the transaction into the lane
		s.Require().NoError(lane.Insert(s.ctx, tx))

		txBz, err := s.encodingConfig.TxConfig.TxEncoder()(tx)
		s.Require().NoError(err)

		emptyProposal := proposals.NewProposal(
			log.NewNopLogger(),
			int64(len(txBz))*10,
			1000000,
		)

		mockLane := mocks.NewLane(s.T())

		mockLane.On("Name").Return("test")
		mockLane.On("GetMaxBlockSpace").Return(math.LegacyOneDec())

		txWithInfo, err := lane.GetTxInfo(s.ctx, tx)
		s.Require().NoError(err)

		err = emptyProposal.UpdateProposal(mockLane, []utils.TxWithInfo{txWithInfo})
		s.Require().NoError(err)

		finalProposal, err := lane.PrepareLane(s.ctx, emptyProposal, block.NoOpPrepareLanesHandler())
		s.Require().NoError(err)

		// Ensure the proposal is ordered correctly
		s.Require().Equal(1, len(finalProposal.Txs))
		s.Require().Equal(int64(len(txBz)), finalProposal.Info.BlockSize)
		s.Require().Equal(uint64(2), finalProposal.Info.GasLimit)
		s.Require().Equal([][]byte{txBz}, finalProposal.Txs)
	})

	s.Run("should not attempt to include transaction that matches to a different lane", func() {
		// Create a basic transaction that should not in the proposal
		tx, err := testutils.CreateRandomTx(
			s.encodingConfig.TxConfig,
			s.accounts[0],
			0,
			1,
			0,
			2,
			sdk.NewCoin(s.gasTokenDenom, math.NewInt(1)),
		)
		s.Require().NoError(err)

		mh := func(ctx sdk.Context, tx sdk.Tx) bool {
			return true
		}

		// Create a lane with a max block space of 1 but a proposal that is smaller than the tx
		expectedExecution := map[sdk.Tx]bool{
			tx: true,
		}
		lane := s.initLaneWithMatchHandlers(math.LegacyOneDec(), expectedExecution, []base.MatchHandler{mh})

		// Insert the transaction into the lane
		s.Require().NoError(lane.Insert(s.ctx, tx))

		txBz, err := s.encodingConfig.TxConfig.TxEncoder()(tx)
		s.Require().NoError(err)

		emptyProposal := proposals.NewProposal(
			log.NewNopLogger(),
			int64(len(txBz))*10,
			1000000,
		)

		finalProposal, err := lane.PrepareLane(s.ctx, emptyProposal, block.NoOpPrepareLanesHandler())
		s.Require().NoError(err)
		s.Require().Len(finalProposal.Txs, 0)
	})
}

func (s *BaseTestSuite) TestProcessLane() {
	s.Run("should accept a proposal where transaction fees are not in order bc of sequence numbers", func() {
		tx1, err := testutils.CreateRandomTx(
			s.encodingConfig.TxConfig,
			s.accounts[0],
			0,
			1,
			0,
			1,
			sdk.NewCoin(s.gasTokenDenom, math.NewInt(1)),
		)
		s.Require().NoError(err)

		tx2, err := testutils.CreateRandomTx(
			s.encodingConfig.TxConfig,
			s.accounts[0],
			1,
			0,
			0,
			0,
			sdk.NewCoin(s.gasTokenDenom, math.NewInt(2)),
		)
		s.Require().NoError(err)

		proposal := []sdk.Tx{
			tx1,
			tx2, // This transaction has a higher sequence number and higher fees
		}

		lane := s.initLane(
			math.LegacyOneDec(),
			map[sdk.Tx]bool{
				tx1: true,
				tx2: true,
			},
		)

		txsFromLane, remainingTxs, err := base.NewDefaultProposalHandler(lane).ProcessLaneHandler()(s.ctx, proposal)
		s.Require().NoError(err)
		s.Require().Len(txsFromLane, 2)
		s.Require().Len(remainingTxs, 0)

		emptyProposal := proposals.NewProposal(
			log.NewNopLogger(),
			100000,
			100000,
		)

		finalProposal, err := lane.ProcessLane(s.ctx, emptyProposal, proposal, block.NoOpProcessLanesHandler())
		s.Require().Len(finalProposal.Txs, 2)
		s.Require().NoError(err)

		encodedTxs, err := utils.GetEncodedTxs(s.encodingConfig.TxConfig.TxEncoder(), proposal)
		s.Require().NoError(err)
		s.Require().Equal(encodedTxs, finalProposal.Txs)
	})

	s.Run("should accept a proposal where transaction fees are not in order bc of sequence numbers with other txs", func() {
		tx1, err := testutils.CreateRandomTx(
			s.encodingConfig.TxConfig,
			s.accounts[0],
			0,
			1,
			0,
			1,
			sdk.NewCoin(s.gasTokenDenom, math.NewInt(10)),
		)
		s.Require().NoError(err)

		tx2, err := testutils.CreateRandomTx(
			s.encodingConfig.TxConfig,
			s.accounts[0],
			1,
			0,
			0,
			0,
			sdk.NewCoin(s.gasTokenDenom, math.NewInt(20)),
		)
		s.Require().NoError(err)

		tx3, err := testutils.CreateRandomTx(
			s.encodingConfig.TxConfig,
			s.accounts[1],
			0,
			1,
			0,
			1,
			sdk.NewCoin(s.gasTokenDenom, math.NewInt(3)),
		)
		s.Require().NoError(err)

		proposal := []sdk.Tx{
			tx1,
			tx2, // This transaction has a higher sequence number and higher fees
			tx3,
		}

		lane := s.initLane(
			math.LegacyOneDec(),
			map[sdk.Tx]bool{
				tx1: true,
				tx2: true,
				tx3: true,
			},
		)

		//
		txsFromLane, remainingTxs, err := base.NewDefaultProposalHandler(lane).ProcessLaneHandler()(s.ctx, proposal)
		s.Require().NoError(err)
		s.Require().Len(txsFromLane, 3)
		s.Require().Len(remainingTxs, 0)

		emptyProposal := proposals.NewProposal(
			log.NewNopLogger(),
			100000,
			100000,
		)

		finalProposal, err := lane.ProcessLane(s.ctx, emptyProposal, proposal, block.NoOpProcessLanesHandler())
		s.Require().Len(finalProposal.Txs, 3)
		s.Require().NoError(err)

		encodedTxs, err := utils.GetEncodedTxs(s.encodingConfig.TxConfig.TxEncoder(), proposal)
		s.Require().NoError(err)
		s.Require().Equal(encodedTxs, finalProposal.Txs)
	})

	s.Run("accepts proposal with multiple senders and seq nums", func() {
		tx1, err := testutils.CreateRandomTx(
			s.encodingConfig.TxConfig,
			s.accounts[0],
			0,
			1,
			0,
			1,
			sdk.NewCoin(s.gasTokenDenom, math.NewInt(10)),
		)
		s.Require().NoError(err)

		tx2, err := testutils.CreateRandomTx(
			s.encodingConfig.TxConfig,
			s.accounts[0],
			1,
			0,
			0,
			0,
			sdk.NewCoin(s.gasTokenDenom, math.NewInt(20)),
		)
		s.Require().NoError(err)

		tx3, err := testutils.CreateRandomTx(
			s.encodingConfig.TxConfig,
			s.accounts[1],
			0,
			1,
			0,
			1,
			sdk.NewCoin(s.gasTokenDenom, math.NewInt(9)),
		)
		s.Require().NoError(err)

		tx4, err := testutils.CreateRandomTx(
			s.encodingConfig.TxConfig,
			s.accounts[1],
			1,
			1,
			0,
			1,
			sdk.NewCoin(s.gasTokenDenom, math.NewInt(11)),
		)
		s.Require().NoError(err)

		proposal := []sdk.Tx{
			tx1,
			tx2,
			tx3,
			tx4,
		}

		lane := s.initLane(
			math.LegacyOneDec(),
			map[sdk.Tx]bool{
				tx1: true,
				tx2: true,
				tx3: true,
				tx4: true,
			},
		)

		txsFromLane, remainingTxs, err := base.NewDefaultProposalHandler(lane).ProcessLaneHandler()(s.ctx, proposal)
		s.Require().NoError(err)
		s.Require().Len(txsFromLane, 4)
		s.Require().Len(remainingTxs, 0)

		emptyProposal := proposals.NewProposal(
			log.NewNopLogger(),
			100000,
			100000,
		)

		finalProposal, err := lane.ProcessLane(s.ctx, emptyProposal, proposal, block.NoOpProcessLanesHandler())
		s.Require().Len(finalProposal.Txs, 4)
		s.Require().NoError(err)

		encodedTxs, err := utils.GetEncodedTxs(s.encodingConfig.TxConfig.TxEncoder(), proposal)
		s.Require().NoError(err)
		s.Require().Equal(encodedTxs, finalProposal.Txs)
	})

	s.Run("should accept a proposal with a single valid transaction", func() {
		tx1, err := testutils.CreateRandomTx(
			s.encodingConfig.TxConfig,
			s.accounts[0],
			0,
			1,
			0,
			1,
		)
		s.Require().NoError(err)

		proposal := []sdk.Tx{
			tx1,
		}

		lane := s.initLane(
			math.LegacyOneDec(),
			map[sdk.Tx]bool{
				tx1: true,
			},
		)

		txsFromLane, remainingTxs, err := base.NewDefaultProposalHandler(lane).ProcessLaneHandler()(s.ctx, proposal)
		s.Require().NoError(err)
		s.Require().Len(txsFromLane, 1)
		s.Require().Len(remainingTxs, 0)

		emptyProposal := proposals.NewProposal(
			log.NewNopLogger(),
			100000,
			100000,
		)

		finalProposal, err := lane.ProcessLane(s.ctx, emptyProposal, proposal, block.NoOpProcessLanesHandler())
		s.Require().Len(finalProposal.Txs, 1)
		s.Require().NoError(err)

		encodedTxs, err := utils.GetEncodedTxs(s.encodingConfig.TxConfig.TxEncoder(), proposal)
		s.Require().NoError(err)
		s.Require().Equal(encodedTxs, finalProposal.Txs)
	})

	s.Run("should not accept a proposal with invalid transactions", func() {
		tx1, err := testutils.CreateRandomTx(
			s.encodingConfig.TxConfig,
			s.accounts[0],
			0,
			1,
			0,
			1,
		)
		s.Require().NoError(err)

		proposal := []sdk.Tx{
			tx1,
		}

		lane := s.initLane(
			math.LegacyOneDec(),
			map[sdk.Tx]bool{
				tx1: false,
			},
		)

		txsFromLane, remainingTxs, err := base.NewDefaultProposalHandler(lane).ProcessLaneHandler()(s.ctx, proposal)
		s.Require().Error(err)
		s.Require().Len(txsFromLane, 0)
		s.Require().Len(remainingTxs, 0)

		emptyProposal := proposals.NewProposal(
			log.NewNopLogger(),
			100000,
			100000,
		)

		_, err = lane.ProcessLane(s.ctx, emptyProposal, proposal, block.NoOpProcessLanesHandler())
		s.Require().Error(err)
	})

	s.Run("should not accept a proposal with some invalid transactions", func() {
		tx1, err := testutils.CreateRandomTx(
			s.encodingConfig.TxConfig,
			s.accounts[0],
			0,
			1,
			0,
			1,
		)
		s.Require().NoError(err)

		tx2, err := testutils.CreateRandomTx(
			s.encodingConfig.TxConfig,
			s.accounts[1],
			0,
			1,
			0,
			1,
		)
		s.Require().NoError(err)

		tx3, err := testutils.CreateRandomTx(
			s.encodingConfig.TxConfig,
			s.accounts[2],
			0,
			1,
			0,
			1,
		)
		s.Require().NoError(err)

		proposal := []sdk.Tx{
			tx1,
			tx2,
			tx3,
		}

		lane := s.initLane(
			math.LegacyOneDec(),
			map[sdk.Tx]bool{
				tx1: true,
				tx2: false,
				tx3: true,
			},
		)

		txsFromLane, remainingTxs, err := base.NewDefaultProposalHandler(lane).ProcessLaneHandler()(s.ctx, proposal)
		s.Require().Error(err)
		s.Require().Len(txsFromLane, 0)
		s.Require().Len(remainingTxs, 0)

		emptyProposal := proposals.NewProposal(
			log.NewNopLogger(),
			100000,
			100000,
		)

		_, err = lane.ProcessLane(s.ctx, emptyProposal, proposal, block.NoOpProcessLanesHandler())
		s.Require().Error(err)
	})

	s.Run("should accept proposal with transactions in correct order with same fees", func() {
		tx1, err := testutils.CreateRandomTx(
			s.encodingConfig.TxConfig,
			s.accounts[0],
			0,
			1,
			0,
			1,
			sdk.NewCoin(s.gasTokenDenom, math.NewInt(2)),
		)
		s.Require().NoError(err)

		tx2, err := testutils.CreateRandomTx(
			s.encodingConfig.TxConfig,
			s.accounts[1],
			0,
			1,
			0,
			1,
			sdk.NewCoin(s.gasTokenDenom, math.NewInt(2)),
		)
		s.Require().NoError(err)

		proposal := []sdk.Tx{
			tx1,
			tx2,
		}

		lane := s.initLane(
			math.LegacyOneDec(),
			map[sdk.Tx]bool{
				tx1: true,
				tx2: true,
			},
		)

		txsFromLane, remainingTxs, err := base.NewDefaultProposalHandler(lane).ProcessLaneHandler()(s.ctx, proposal)
		s.Require().NoError(err)
		s.Require().Len(txsFromLane, 2)
		s.Require().Len(remainingTxs, 0)

		emptyProposal := proposals.NewProposal(
			log.NewNopLogger(),
			100000,
			100000,
		)

		finalProposal, err := lane.ProcessLane(s.ctx, emptyProposal, proposal, block.NoOpProcessLanesHandler())
		s.Require().Len(finalProposal.Txs, 2)
		s.Require().NoError(err)

		encodedTxs, err := utils.GetEncodedTxs(s.encodingConfig.TxConfig.TxEncoder(), proposal)
		s.Require().NoError(err)
		s.Require().Equal(encodedTxs, finalProposal.Txs)
	})

	s.Run("should accept a proposal with transactions that are in any order fee wise", func() {
		tx1, err := testutils.CreateRandomTx(
			s.encodingConfig.TxConfig,
			s.accounts[0],
			0,
			1,
			0,
			1,
			sdk.NewCoin(s.gasTokenDenom, math.NewInt(1)),
		)
		s.Require().NoError(err)

		tx2, err := testutils.CreateRandomTx(
			s.encodingConfig.TxConfig,
			s.accounts[1],
			0,
			1,
			0,
			1,
			sdk.NewCoin(s.gasTokenDenom, math.NewInt(2)),
		)
		s.Require().NoError(err)

		proposal := []sdk.Tx{
			tx1,
			tx2,
		}

		lane := s.initLane(
			math.LegacyOneDec(),
			map[sdk.Tx]bool{
				tx1: true,
				tx2: true,
			},
		)

		txsFromLane, remainingTxs, err := base.NewDefaultProposalHandler(lane).ProcessLaneHandler()(s.ctx, proposal)
		s.Require().NoError(err)
		s.Require().Len(txsFromLane, 2)
		s.Require().Len(remainingTxs, 0)

		emptyProposal := proposals.NewProposal(
			log.NewNopLogger(),
			100000,
			100000,
		)

		_, err = lane.ProcessLane(s.ctx, emptyProposal, proposal, block.NoOpProcessLanesHandler())
		s.Require().NoError(err)
	})

	s.Run("should not accept proposal where transactions from lane are not contiguous from the start", func() {
		tx1, err := testutils.CreateRandomTx(
			s.encodingConfig.TxConfig,
			s.accounts[0],
			0,
			2,
			0,
			1,
			sdk.NewCoin(s.gasTokenDenom, math.NewInt(1)),
		)
		s.Require().NoError(err)

		tx2, err := testutils.CreateRandomTx(
			s.encodingConfig.TxConfig,
			s.accounts[1],
			0,
			1,
			0,
			1,
			sdk.NewCoin(s.gasTokenDenom, math.NewInt(2)),
		)
		s.Require().NoError(err)

		// First lane matches this lane the other does not.
		mh := func(ctx sdk.Context, tx sdk.Tx) bool {
			return tx == tx1
		}

		lane := s.initLaneWithMatchHandlers(
			math.LegacyOneDec(),
			map[sdk.Tx]bool{
				tx1: true,
				tx2: true,
			},
			[]base.MatchHandler{mh},
		)

		proposal := []sdk.Tx{
			tx1,
			tx2,
		}

		txsFromLane, remainingTxs, err := base.NewDefaultProposalHandler(lane).ProcessLaneHandler()(s.ctx, proposal)
		s.Require().Error(err)
		s.Require().Len(txsFromLane, 0)
		s.Require().Len(remainingTxs, 0)

		emptyProposal := proposals.NewProposal(
			log.NewNopLogger(),
			100000,
			100000,
		)

		_, err = lane.ProcessLane(s.ctx, emptyProposal, proposal, block.NoOpProcessLanesHandler())
		s.Require().Error(err)
	})

	s.Run("should not accept a proposal that builds too large of a partial block", func() {
		tx1, err := testutils.CreateRandomTx(
			s.encodingConfig.TxConfig,
			s.accounts[0],
			0,
			1,
			0,
			1,
		)
		s.Require().NoError(err)

		proposal := []sdk.Tx{
			tx1,
		}

		lane := s.initLane(
			math.LegacyOneDec(),
			map[sdk.Tx]bool{
				tx1: true,
			},
		)

		txsFromLane, remainingTxs, err := base.NewDefaultProposalHandler(lane).ProcessLaneHandler()(s.ctx, proposal)
		s.Require().NoError(err)
		s.Require().Len(txsFromLane, 1)
		s.Require().Len(remainingTxs, 0)

		// Set the size to be 1 less than the size of the transaction
		maxSize := s.getTxSize(tx1) - 1
		emptyProposal := proposals.NewProposal(
			log.NewNopLogger(),
			maxSize,
			1000000,
		)

		_, err = lane.ProcessLane(s.ctx, emptyProposal, proposal, block.NoOpProcessLanesHandler())
		s.Require().Error(err)
	})

	s.Run("should not accept a proposal that builds a partial block that is too gas consumptive", func() {
		tx1, err := testutils.CreateRandomTx(
			s.encodingConfig.TxConfig,
			s.accounts[0],
			0,
			1,
			0,
			10,
		)
		s.Require().NoError(err)

		proposal := []sdk.Tx{
			tx1,
		}

		lane := s.initLane(
			math.LegacyOneDec(),
			map[sdk.Tx]bool{
				tx1: true,
			},
		)

		txsFromLane, remainingTxs, err := base.NewDefaultProposalHandler(lane).ProcessLaneHandler()(s.ctx, proposal)
		s.Require().NoError(err)
		s.Require().Len(txsFromLane, 1)
		s.Require().Len(remainingTxs, 0)

		maxSize := s.getTxSize(tx1)
		emptyProposal := proposals.NewProposal(
			log.NewNopLogger(),
			maxSize,
			9,
		)

		_, err = lane.ProcessLane(s.ctx, emptyProposal, proposal, block.NoOpProcessLanesHandler())
		s.Require().Error(err)
	})

	s.Run("should not accept a proposal that builds a partial block that is too gas consumptive p2", func() {
		tx1, err := testutils.CreateRandomTx(
			s.encodingConfig.TxConfig,
			s.accounts[0],
			0,
			1,
			0,
			10,
		)
		s.Require().NoError(err)

		tx2, err := testutils.CreateRandomTx(
			s.encodingConfig.TxConfig,
			s.accounts[1],
			0,
			1,
			0,
			10,
		)
		s.Require().NoError(err)

		proposal := []sdk.Tx{
			tx1,
			tx2,
		}

		lane := s.initLane(
			math.LegacyOneDec(),
			map[sdk.Tx]bool{
				tx1: true,
				tx2: true,
			},
		)

		txsFromLane, remainingTxs, err := base.NewDefaultProposalHandler(lane).ProcessLaneHandler()(s.ctx, proposal)
		s.Require().NoError(err)
		s.Require().Len(txsFromLane, 2)
		s.Require().Len(remainingTxs, 0)

		maxSize := s.getTxSize(tx1) + s.getTxSize(tx2)
		emptyProposal := proposals.NewProposal(
			log.NewNopLogger(),
			maxSize,
			19,
		)

		_, err = lane.ProcessLane(s.ctx, emptyProposal, proposal, block.NoOpProcessLanesHandler())
		s.Require().Error(err)
	})

	s.Run("should not accept a proposal that builds a partial block that is too large p2", func() {
		tx1, err := testutils.CreateRandomTx(
			s.encodingConfig.TxConfig,
			s.accounts[0],
			0,
			1,
			0,
			10,
		)
		s.Require().NoError(err)

		tx2, err := testutils.CreateRandomTx(
			s.encodingConfig.TxConfig,
			s.accounts[1],
			0,
			1,
			0,
			10,
		)
		s.Require().NoError(err)

		proposal := []sdk.Tx{
			tx1,
			tx2,
		}

		lane := s.initLane(
			math.LegacyOneDec(),
			map[sdk.Tx]bool{
				tx1: true,
				tx2: true,
			},
		)

		txsFromLane, remainingTxs, err := base.NewDefaultProposalHandler(lane).ProcessLaneHandler()(s.ctx, proposal)
		s.Require().NoError(err)
		s.Require().Len(txsFromLane, 2)
		s.Require().Len(remainingTxs, 0)

		maxSize := s.getTxSize(tx1) + s.getTxSize(tx2) - 1
		emptyProposal := proposals.NewProposal(
			log.NewNopLogger(),
			maxSize,
			20,
		)

		_, err = lane.ProcessLane(s.ctx, emptyProposal, proposal, block.NoOpProcessLanesHandler())
		s.Require().Error(err)
	})

	s.Run("contiguous set of transactions should be accepted with other transactions that do not match", func() {
		tx1, err := testutils.CreateRandomTx(
			s.encodingConfig.TxConfig,
			s.accounts[0],
			1,
			1,
			0,
			1,
		)
		s.Require().NoError(err)

		tx2, err := testutils.CreateRandomTx(
			s.encodingConfig.TxConfig,
			s.accounts[1],
			2,
			1,
			0,
			1,
		)
		s.Require().NoError(err)

		tx3, err := testutils.CreateRandomTx(
			s.encodingConfig.TxConfig,
			s.accounts[2],
			3,
			1,
			0,
			1,
		)
		s.Require().NoError(err)

		tx4, err := testutils.CreateRandomTx(
			s.encodingConfig.TxConfig,
			s.accounts[3],
			4,
			1,
			0,
			1,
		)
		s.Require().NoError(err)

		proposal := []sdk.Tx{
			tx1,
			tx2,
			tx3,
			tx4,
		}

		mh := func(ctx sdk.Context, tx sdk.Tx) bool {
			if tx == tx1 || tx == tx2 {
				return false
			}

			return true
		}

		lane := s.initLaneWithMatchHandlers(
			math.LegacyOneDec(),
			map[sdk.Tx]bool{
				tx1: true,
				tx2: true,
			},
			[]base.MatchHandler{mh},
		)

		txsFromLane, remainingTxs, err := base.NewDefaultProposalHandler(lane).ProcessLaneHandler()(s.ctx, proposal)
		s.Require().NoError(err)
		s.Require().Len(txsFromLane, 2)
		s.Require().Len(remainingTxs, 2)
		s.Require().Equal([]sdk.Tx{tx3, tx4}, remainingTxs)

		emptyProposal := proposals.NewProposal(
			log.NewNopLogger(),
			1000,
			1000,
		)

		finalProposal, err := lane.ProcessLane(s.ctx, emptyProposal, proposal, block.NoOpProcessLanesHandler())
		s.Require().NoError(err)
		s.Require().Len(finalProposal.Txs, 2)

		encodedTxs, err := utils.GetEncodedTxs(s.encodingConfig.TxConfig.TxEncoder(), []sdk.Tx{tx1, tx2})
		s.Require().NoError(err)
		s.Require().Equal(encodedTxs, finalProposal.Txs)
	})

	s.Run("returns no error if transactions belong to a different lane", func() {
		tx1, err := testutils.CreateRandomTx(
			s.encodingConfig.TxConfig,
			s.accounts[0],
			1,
			1,
			0,
			1,
		)
		s.Require().NoError(err)

		tx2, err := testutils.CreateRandomTx(
			s.encodingConfig.TxConfig,
			s.accounts[1],
			2,
			1,
			0,
			1,
		)
		s.Require().NoError(err)

		tx3, err := testutils.CreateRandomTx(
			s.encodingConfig.TxConfig,
			s.accounts[2],
			3,
			1,
			0,
			1,
		)
		s.Require().NoError(err)

		tx4, err := testutils.CreateRandomTx(
			s.encodingConfig.TxConfig,
			s.accounts[3],
			4,
			1,
			0,
			1,
		)
		s.Require().NoError(err)

		proposal := []sdk.Tx{
			tx1,
			tx2,
			tx3,
			tx4,
		}

		mh := func(ctx sdk.Context, tx sdk.Tx) bool {
			return true
		}

		lane := s.initLaneWithMatchHandlers(
			math.LegacyOneDec(),
			map[sdk.Tx]bool{},
			[]base.MatchHandler{mh},
		)

		txsFromLane, remainingTxs, err := base.NewDefaultProposalHandler(lane).ProcessLaneHandler()(s.ctx, proposal)
		s.Require().NoError(err)
		s.Require().Len(txsFromLane, 0)
		s.Require().Len(remainingTxs, 4)
		s.Require().Equal(proposal, remainingTxs)

		emptyProposal := proposals.NewProposal(
			log.NewNopLogger(),
			1000,
			1000,
		)

		finalProposal, err := lane.ProcessLane(s.ctx, emptyProposal, proposal, block.NoOpProcessLanesHandler())
		s.Require().NoError(err)
		s.Require().Len(finalProposal.Txs, 0)
	})

	s.Run("returns an error if transactions are interleaved with other lanes", func() {
		tx1, err := testutils.CreateRandomTx(
			s.encodingConfig.TxConfig,
			s.accounts[0],
			1,
			1,
			0,
			1,
		)
		s.Require().NoError(err)

		tx2, err := testutils.CreateRandomTx(
			s.encodingConfig.TxConfig,
			s.accounts[1],
			2,
			1,
			0,
			1,
		)
		s.Require().NoError(err)

		tx3, err := testutils.CreateRandomTx(
			s.encodingConfig.TxConfig,
			s.accounts[2],
			3,
			1,
			0,
			1,
		)
		s.Require().NoError(err)

		tx4, err := testutils.CreateRandomTx(
			s.encodingConfig.TxConfig,
			s.accounts[3],
			4,
			1,
			0,
			1,
		)
		s.Require().NoError(err)

		proposal := []sdk.Tx{
			tx1,
			tx2,
			tx3,
			tx4,
		}

		mh := func(ctx sdk.Context, tx sdk.Tx) bool {
			if tx == tx1 || tx == tx3 {
				return false
			}

			return true
		}

		lane := s.initLaneWithMatchHandlers(
			math.LegacyOneDec(),
			map[sdk.Tx]bool{
				tx1: true,
				tx2: true,
				tx3: true,
				tx4: true,
			},
			[]base.MatchHandler{mh},
		)

		txsFromLane, remainingTxs, err := base.NewDefaultProposalHandler(lane).ProcessLaneHandler()(s.ctx, proposal)
		s.Require().Error(err)
		s.Require().Len(txsFromLane, 0)
		s.Require().Len(remainingTxs, 0)

		emptyProposal := proposals.NewProposal(
			log.NewNopLogger(),
			1000,
			1000,
		)

		_, err = lane.ProcessLane(s.ctx, emptyProposal, proposal, block.NoOpProcessLanesHandler())
		s.Require().Error(err)
	})
}

func (s *BaseTestSuite) TestPrepareProcessParity() {
	txsToInsert := []sdk.Tx{}
	validationMap := make(map[sdk.Tx]bool)
	numTxsPerAccount := uint64(50)
	accounts := testutils.RandomAccounts(s.random, 50)

	for _, account := range accounts {
		for nonce := uint64(0); nonce < numTxsPerAccount; nonce++ {
			// create a random fee amount
			feeAmount := math.NewInt(int64(rand.Intn(100000)))
			tx, err := testutils.CreateRandomTx(
				s.encodingConfig.TxConfig,
				account,
				nonce,
				1,
				0,
				1,
				sdk.NewCoin(s.gasTokenDenom, feeAmount),
			)
			s.Require().NoError(err)

			txsToInsert = append(txsToInsert, tx)
			validationMap[tx] = true
		}
	}

	// Add the transactions to the lane
	lane := s.initLane(math.LegacyOneDec(), validationMap)
	for _, tx := range txsToInsert {
		s.Require().NoError(lane.Insert(s.ctx, tx))
	}

	// Retrieve the transactions from the lane in the same way the prepare function would.
	retrievedTxs := []sdk.Tx{}
	for iterator := lane.Select(context.Background(), nil); iterator != nil; iterator = iterator.Next() {
		retrievedTxs = append(retrievedTxs, iterator.Tx())
	}
	s.Require().Equal(len(txsToInsert), len(retrievedTxs))

	// Construct a block proposal with the transactions in the mempool
	emptyProposal := proposals.NewProposal(
		log.NewNopLogger(),
		1000000000000000,
		1000000000000000,
	)
	proposal, err := lane.PrepareLane(s.ctx, emptyProposal, block.NoOpPrepareLanesHandler())
	s.Require().NoError(err)
	s.Require().Equal(len(retrievedTxs), len(proposal.Txs))

	// Ensure that the transactions are in the same order
	for i := 0; i < len(retrievedTxs); i++ {
		bz, err := s.encodingConfig.TxConfig.TxEncoder()(retrievedTxs[i])
		s.Require().NoError(err)
		s.Require().Equal(bz, proposal.Txs[i])
	}

	decodedTxs, err := utils.GetDecodedTxs(s.encodingConfig.TxConfig.TxDecoder(), proposal.Txs)
	s.Require().NoError(err)

	// Verify the same proposal with the process lanes handler
	emptyProposal = proposals.NewProposal(
		log.NewNopLogger(),
		1000000000000000,
		1000000000000000,
	)
	proposal, err = lane.ProcessLane(s.ctx, emptyProposal, decodedTxs, block.NoOpProcessLanesHandler())
	s.Require().NoError(err)
	s.Require().Equal(len(txsToInsert), len(proposal.Txs))
	s.T().Logf("proposal num txs: %d", len(proposal.Txs))

	// Ensure that the transactions are in the same order
	for i := 0; i < len(retrievedTxs); i++ {
		bz, err := s.encodingConfig.TxConfig.TxEncoder()(retrievedTxs[i])
		s.Require().NoError(err)
		s.Require().Equal(bz, proposal.Txs[i])
	}
}

func (s *BaseTestSuite) TestIterateMempoolAndProcessProposalParity() {
	txsToInsert := []sdk.Tx{}
	validationMap := make(map[sdk.Tx]bool)
	numTxsPerAccount := uint64(200)
	accounts := testutils.RandomAccounts(s.random, 50)

	for _, account := range accounts {
		for nonce := uint64(0); nonce < numTxsPerAccount; nonce++ {
			// create a random fee amount
			feeAmount := math.NewInt(int64(rand.Intn(100000)))
			tx, err := testutils.CreateRandomTx(
				s.encodingConfig.TxConfig,
				account,
				nonce,
				1,
				0,
				1,
				sdk.NewCoin(s.gasTokenDenom, feeAmount),
			)
			s.Require().NoError(err)

			txsToInsert = append(txsToInsert, tx)
			validationMap[tx] = true
		}
	}

	// Add the transactions to the lane
	lane := s.initLane(math.LegacyOneDec(), validationMap)
	for _, tx := range txsToInsert {
		s.Require().NoError(lane.Insert(s.ctx, tx))
	}

	// Retrieve the transactions from the lane in the same way the prepare function would.
	retrievedTxs := []sdk.Tx{}
	for iterator := lane.Select(context.Background(), nil); iterator != nil; iterator = iterator.Next() {
		retrievedTxs = append(retrievedTxs, iterator.Tx())
	}

	s.Require().Equal(len(txsToInsert), len(retrievedTxs))

	emptyProposal := proposals.NewProposal(
		log.NewNopLogger(),
		1000000000000000,
		1000000000000000,
	)

	proposal, err := lane.ProcessLane(s.ctx, emptyProposal, retrievedTxs, block.NoOpProcessLanesHandler())
	s.Require().NoError(err)
	s.Require().Equal(len(retrievedTxs), len(proposal.Txs))
	s.T().Logf("proposal num txs: %d", len(proposal.Txs))

	// Ensure that the transactions are in the same order
	for i := 0; i < len(retrievedTxs); i++ {
		bz, err := s.encodingConfig.TxConfig.TxEncoder()(retrievedTxs[i])
		s.Require().NoError(err)
		s.Require().Equal(bz, proposal.Txs[i])
	}
}

func (s *BaseTestSuite) initLane(
	maxBlockSpace math.LegacyDec,
	expectedExecution map[sdk.Tx]bool,
) *base.BaseLane {
	config := base.NewLaneConfig(
		log.NewNopLogger(),
		s.encodingConfig.TxConfig.TxEncoder(),
		s.encodingConfig.TxConfig.TxDecoder(),
		s.setUpAnteHandler(expectedExecution),
		signer_extraction.NewDefaultAdapter(),
		maxBlockSpace,
	)

	return defaultlane.NewDefaultLane(config, base.DefaultMatchHandler())
}

func (s *BaseTestSuite) initLaneWithMatchHandlers(
	maxBlockSpace math.LegacyDec,
	expectedExecution map[sdk.Tx]bool,
	matchHandlers []base.MatchHandler,
) *base.BaseLane {
	config := base.NewLaneConfig(
		log.NewNopLogger(),
		s.encodingConfig.TxConfig.TxEncoder(),
		s.encodingConfig.TxConfig.TxDecoder(),
		s.setUpAnteHandler(expectedExecution),
		signer_extraction.NewDefaultAdapter(),
		maxBlockSpace,
	)

	mh := base.NewMatchHandler(base.DefaultMatchHandler(), matchHandlers...)

	return defaultlane.NewDefaultLane(config, mh)
}

func (s *BaseTestSuite) setUpAnteHandler(expectedExecution map[sdk.Tx]bool) sdk.AnteHandler {
	txCache := make(map[string]bool)
	for tx, pass := range expectedExecution {
		bz, err := s.encodingConfig.TxConfig.TxEncoder()(tx)
		s.Require().NoError(err)

		hash := sha256.Sum256(bz)
		hashStr := hex.EncodeToString(hash[:])
		txCache[hashStr] = pass
	}

	anteHandler := func(ctx sdk.Context, tx sdk.Tx, simulate bool) (newCtx sdk.Context, err error) {
		bz, err := s.encodingConfig.TxConfig.TxEncoder()(tx)
		s.Require().NoError(err)

		hash := sha256.Sum256(bz)
		hashStr := hex.EncodeToString(hash[:])

		pass, found := txCache[hashStr]
		if !found {
			return ctx, fmt.Errorf("tx not found")
		}

		if pass {
			return ctx, nil
		}

		return ctx, fmt.Errorf("tx failed")
	}

	return anteHandler
}

func (s *BaseTestSuite) getTxSize(tx sdk.Tx) int64 {
	txBz, err := s.encodingConfig.TxConfig.TxEncoder()(tx)
	s.Require().NoError(err)

	return int64(len(txBz))
}
