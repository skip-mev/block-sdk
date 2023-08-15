package base_test

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"cosmossdk.io/log"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/skip-mev/pob/blockbuster"
	"github.com/skip-mev/pob/blockbuster/lanes/base"
	"github.com/skip-mev/pob/blockbuster/utils/mocks"
	testutils "github.com/skip-mev/pob/testutils"
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
			sdk.NewCoin(s.gasTokenDenom, math.NewInt(1)),
		)
		s.Require().NoError(err)

		// Create a lane with a max block space of 1 but a proposal that is smaller than the tx
		expectedExecution := map[sdk.Tx]bool{
			tx: true,
		}
		lane := s.initLane(math.LegacyMustNewDecFromStr("1"), expectedExecution)

		// Insert the transaction into the lane
		s.Require().NoError(lane.Insert(sdk.Context{}, tx))

		txBz, err := s.encodingConfig.TxConfig.TxEncoder()(tx)
		s.Require().NoError(err)

		// Create a proposal
		maxTxBytes := int64(len(txBz) - 1)
		proposal, err := lane.PrepareLane(sdk.Context{}, blockbuster.NewProposal(maxTxBytes), maxTxBytes, blockbuster.NoOpPrepareLanesHandler())
		s.Require().NoError(err)

		// Ensure the proposal is empty
		s.Require().Equal(0, proposal.GetNumTxs())
		s.Require().Equal(int64(0), proposal.GetTotalTxBytes())
	})

	s.Run("should not build a proposal when box space configured to lane is too small", func() {
		// Create a basic transaction that should not in the proposal
		tx, err := testutils.CreateRandomTx(
			s.encodingConfig.TxConfig,
			s.accounts[0],
			0,
			1,
			0,
			sdk.NewCoin(s.gasTokenDenom, math.NewInt(1)),
		)
		s.Require().NoError(err)

		// Create a lane with a max block space of 1 but a proposal that is smaller than the tx
		expectedExecution := map[sdk.Tx]bool{
			tx: true,
		}
		lane := s.initLane(math.LegacyMustNewDecFromStr("0.000001"), expectedExecution)

		// Insert the transaction into the lane
		s.Require().NoError(lane.Insert(sdk.Context{}, tx))

		txBz, err := s.encodingConfig.TxConfig.TxEncoder()(tx)
		s.Require().NoError(err)

		// Create a proposal
		maxTxBytes := int64(len(txBz))
		proposal, err := lane.PrepareLane(sdk.Context{}, blockbuster.NewProposal(maxTxBytes), maxTxBytes, blockbuster.NoOpPrepareLanesHandler())
		s.Require().Error(err)

		// Ensure the proposal is empty
		s.Require().Equal(0, proposal.GetNumTxs())
		s.Require().Equal(int64(0), proposal.GetTotalTxBytes())
	})

	s.Run("should be able to build a proposal with a tx that just fits in", func() {
		// Create a basic transaction that should not in the proposal
		tx, err := testutils.CreateRandomTx(
			s.encodingConfig.TxConfig,
			s.accounts[0],
			0,
			1,
			0,
			sdk.NewCoin(s.gasTokenDenom, math.NewInt(1)),
		)
		s.Require().NoError(err)

		// Create a lane with a max block space of 1 but a proposal that is smaller than the tx
		expectedExecution := map[sdk.Tx]bool{
			tx: true,
		}
		lane := s.initLane(math.LegacyMustNewDecFromStr("1"), expectedExecution)

		// Insert the transaction into the lane
		s.Require().NoError(lane.Insert(sdk.Context{}, tx))

		txBz, err := s.encodingConfig.TxConfig.TxEncoder()(tx)
		s.Require().NoError(err)

		// Create a proposal
		maxTxBytes := int64(len(txBz))
		proposal, err := lane.PrepareLane(sdk.Context{}, blockbuster.NewProposal(maxTxBytes), maxTxBytes, blockbuster.NoOpPrepareLanesHandler())
		s.Require().NoError(err)

		// Ensure the proposal is not empty and contains the transaction
		s.Require().Equal(1, proposal.GetNumTxs())
		s.Require().Equal(maxTxBytes, proposal.GetTotalTxBytes())
		s.Require().Equal(txBz, proposal.GetTxs()[0])
	})

	s.Run("should not build a proposal with a that fails verify tx", func() {
		// Create a basic transaction that should not in the proposal
		tx, err := testutils.CreateRandomTx(
			s.encodingConfig.TxConfig,
			s.accounts[0],
			0,
			1,
			0,
			sdk.NewCoin(s.gasTokenDenom, math.NewInt(1)),
		)
		s.Require().NoError(err)

		// Create a lane with a max block space of 1 but a proposal that is smaller than the tx
		expectedExecution := map[sdk.Tx]bool{
			tx: false,
		}
		lane := s.initLane(math.LegacyMustNewDecFromStr("1"), expectedExecution)

		// Insert the transaction into the lane
		s.Require().NoError(lane.Insert(sdk.Context{}, tx))

		// Create a proposal
		txBz, err := s.encodingConfig.TxConfig.TxEncoder()(tx)
		s.Require().NoError(err)

		maxTxBytes := int64(len(txBz))
		proposal, err := lane.PrepareLane(sdk.Context{}, blockbuster.NewProposal(maxTxBytes), maxTxBytes, blockbuster.NoOpPrepareLanesHandler())
		s.Require().NoError(err)

		// Ensure the proposal is empty
		s.Require().Equal(0, proposal.GetNumTxs())
		s.Require().Equal(int64(0), proposal.GetTotalTxBytes())

		// Ensure the transaction is removed from the lane
		s.Require().False(lane.Contains(tx))
		s.Require().Equal(0, lane.CountTx())
	})

	s.Run("should order transactions correctly in the proposal", func() {
		// Create a basic transaction that should not in the proposal
		tx1, err := testutils.CreateRandomTx(
			s.encodingConfig.TxConfig,
			s.accounts[0],
			0,
			1,
			0,
			sdk.NewCoin(s.gasTokenDenom, math.NewInt(2)),
		)
		s.Require().NoError(err)

		tx2, err := testutils.CreateRandomTx(
			s.encodingConfig.TxConfig,
			s.accounts[1],
			0,
			1,
			0,
			sdk.NewCoin(s.gasTokenDenom, math.NewInt(1)),
		)
		s.Require().NoError(err)

		// Create a lane with a max block space of 1 but a proposal that is smaller than the tx
		expectedExecution := map[sdk.Tx]bool{
			tx1: true,
			tx2: true,
		}
		lane := s.initLane(math.LegacyMustNewDecFromStr("1"), expectedExecution)

		// Insert the transaction into the lane
		s.Require().NoError(lane.Insert(sdk.Context{}, tx1))
		s.Require().NoError(lane.Insert(sdk.Context{}, tx2))

		txBz1, err := s.encodingConfig.TxConfig.TxEncoder()(tx1)
		s.Require().NoError(err)

		txBz2, err := s.encodingConfig.TxConfig.TxEncoder()(tx2)
		s.Require().NoError(err)

		maxTxBytes := int64(len(txBz1)) + int64(len(txBz2))
		proposal, err := lane.PrepareLane(sdk.Context{}, blockbuster.NewProposal(maxTxBytes), maxTxBytes, blockbuster.NoOpPrepareLanesHandler())
		s.Require().NoError(err)

		// Ensure the proposal is ordered correctly
		s.Require().Equal(2, proposal.GetNumTxs())
		s.Require().Equal(maxTxBytes, proposal.GetTotalTxBytes())
		s.Require().Equal([][]byte{txBz1, txBz2}, proposal.GetTxs())
	})

	s.Run("should order transactions correctly in the proposal (with different insertion)", func() {
		// Create a basic transaction that should not in the proposal
		tx1, err := testutils.CreateRandomTx(
			s.encodingConfig.TxConfig,
			s.accounts[0],
			0,
			1,
			0,
			sdk.NewCoin(s.gasTokenDenom, math.NewInt(1)),
		)
		s.Require().NoError(err)

		tx2, err := testutils.CreateRandomTx(
			s.encodingConfig.TxConfig,
			s.accounts[1],
			0,
			1,
			0,
			sdk.NewCoin(s.gasTokenDenom, math.NewInt(2)),
		)
		s.Require().NoError(err)

		// Create a lane with a max block space of 1 but a proposal that is smaller than the tx
		expectedExecution := map[sdk.Tx]bool{
			tx1: true,
			tx2: true,
		}
		lane := s.initLane(math.LegacyMustNewDecFromStr("1"), expectedExecution)

		// Insert the transaction into the lane
		s.Require().NoError(lane.Insert(sdk.Context{}, tx1))
		s.Require().NoError(lane.Insert(sdk.Context{}, tx2))

		txBz1, err := s.encodingConfig.TxConfig.TxEncoder()(tx1)
		s.Require().NoError(err)

		txBz2, err := s.encodingConfig.TxConfig.TxEncoder()(tx2)
		s.Require().NoError(err)

		maxTxBytes := int64(len(txBz1)) + int64(len(txBz2))
		proposal, err := lane.PrepareLane(sdk.Context{}, blockbuster.NewProposal(maxTxBytes), maxTxBytes, blockbuster.NoOpPrepareLanesHandler())
		s.Require().NoError(err)

		// Ensure the proposal is ordered correctly
		s.Require().Equal(2, proposal.GetNumTxs())
		s.Require().Equal(maxTxBytes, proposal.GetTotalTxBytes())
		s.Require().Equal([][]byte{txBz2, txBz1}, proposal.GetTxs())
	})

	s.Run("should include tx that fits in proposal when other does not", func() {
		// Create a basic transaction that should not in the proposal
		tx1, err := testutils.CreateRandomTx(
			s.encodingConfig.TxConfig,
			s.accounts[0],
			0,
			1,
			0,
			sdk.NewCoin(s.gasTokenDenom, math.NewInt(1)),
		)
		s.Require().NoError(err)

		tx2, err := testutils.CreateRandomTx(
			s.encodingConfig.TxConfig,
			s.accounts[1],
			0,
			10, // This tx is too large to fit in the proposal
			0,
			sdk.NewCoin(s.gasTokenDenom, math.NewInt(1)),
		)
		s.Require().NoError(err)

		// Create a lane with a max block space of 1 but a proposal that is smaller than the tx
		expectedExecution := map[sdk.Tx]bool{
			tx1: true,
			tx2: true,
		}
		lane := s.initLane(math.LegacyMustNewDecFromStr("1"), expectedExecution)

		// Insert the transaction into the lane
		s.Require().NoError(lane.Insert(sdk.Context{}.WithPriority(10), tx1))
		s.Require().NoError(lane.Insert(sdk.Context{}.WithPriority(5), tx2))

		txBz1, err := s.encodingConfig.TxConfig.TxEncoder()(tx1)
		s.Require().NoError(err)

		txBz2, err := s.encodingConfig.TxConfig.TxEncoder()(tx2)
		s.Require().NoError(err)

		maxTxBytes := int64(len(txBz1)) + int64(len(txBz2)) - 1
		proposal, err := lane.PrepareLane(sdk.Context{}, blockbuster.NewProposal(maxTxBytes), maxTxBytes, blockbuster.NoOpPrepareLanesHandler())
		s.Require().NoError(err)

		// Ensure the proposal is ordered correctly
		s.Require().Equal(1, proposal.GetNumTxs())
		s.Require().Equal(int64(len(txBz1)), proposal.GetTotalTxBytes())
		s.Require().Equal([][]byte{txBz1}, proposal.GetTxs())
	})
}

func (s *BaseTestSuite) TestProcessLane() {
	s.Run("should accept a proposal with valid transactions", func() {
		tx1, err := testutils.CreateRandomTx(
			s.encodingConfig.TxConfig,
			s.accounts[0],
			0,
			1,
			0,
		)
		s.Require().NoError(err)

		proposal := []sdk.Tx{
			tx1,
		}

		lane := s.initLane(math.LegacyMustNewDecFromStr("1"), map[sdk.Tx]bool{
			tx1: true,
		})

		_, err = lane.ProcessLane(sdk.Context{}, proposal, blockbuster.NoOpProcessLanesHandler())
		s.Require().NoError(err)
	})

	s.Run("should not accept a proposal with invalid transactions", func() {
		tx1, err := testutils.CreateRandomTx(
			s.encodingConfig.TxConfig,
			s.accounts[0],
			0,
			1,
			0,
		)
		s.Require().NoError(err)

		proposal := []sdk.Tx{
			tx1,
		}

		lane := s.initLane(math.LegacyMustNewDecFromStr("1"), map[sdk.Tx]bool{
			tx1: false,
		})

		_, err = lane.ProcessLane(sdk.Context{}, proposal, blockbuster.NoOpProcessLanesHandler())
		s.Require().Error(err)
	})

	s.Run("should not accept a proposal with some invalid transactions", func() {
		tx1, err := testutils.CreateRandomTx(
			s.encodingConfig.TxConfig,
			s.accounts[0],
			0,
			1,
			0,
		)
		s.Require().NoError(err)

		tx2, err := testutils.CreateRandomTx(
			s.encodingConfig.TxConfig,
			s.accounts[1],
			0,
			1,
			0,
		)
		s.Require().NoError(err)

		tx3, err := testutils.CreateRandomTx(
			s.encodingConfig.TxConfig,
			s.accounts[2],
			0,
			1,
			0,
		)
		s.Require().NoError(err)

		proposal := []sdk.Tx{
			tx1,
			tx2,
			tx3,
		}

		lane := s.initLane(math.LegacyMustNewDecFromStr("1"), map[sdk.Tx]bool{
			tx1: true,
			tx2: false,
			tx3: true,
		})

		_, err = lane.ProcessLane(sdk.Context{}, proposal, blockbuster.NoOpProcessLanesHandler())
		s.Require().Error(err)
	})
}

func (s *BaseTestSuite) TestCheckOrder() {
	s.Run("should accept proposal with transactions in correct order", func() {
		tx1, err := testutils.CreateRandomTx(
			s.encodingConfig.TxConfig,
			s.accounts[0],
			0,
			1,
			0,
			sdk.NewCoin(s.gasTokenDenom, math.NewInt(2)),
		)
		s.Require().NoError(err)

		tx2, err := testutils.CreateRandomTx(
			s.encodingConfig.TxConfig,
			s.accounts[1],
			0,
			1,
			0,
			sdk.NewCoin(s.gasTokenDenom, math.NewInt(1)),
		)
		s.Require().NoError(err)

		proposal := []sdk.Tx{
			tx1,
			tx2,
		}

		lane := s.initLane(math.LegacyMustNewDecFromStr("1"), map[sdk.Tx]bool{
			tx1: true,
			tx2: true,
		})
		s.Require().NoError(lane.CheckOrder(sdk.Context{}, proposal))
	})

	s.Run("should not accept a proposal with transactions that are not in the correct order", func() {
		tx1, err := testutils.CreateRandomTx(
			s.encodingConfig.TxConfig,
			s.accounts[0],
			0,
			1,
			0,
			sdk.NewCoin(s.gasTokenDenom, math.NewInt(1)),
		)
		s.Require().NoError(err)

		tx2, err := testutils.CreateRandomTx(
			s.encodingConfig.TxConfig,
			s.accounts[1],
			0,
			1,
			0,
			sdk.NewCoin(s.gasTokenDenom, math.NewInt(2)),
		)
		s.Require().NoError(err)

		proposal := []sdk.Tx{
			tx1,
			tx2,
		}

		lane := s.initLane(math.LegacyMustNewDecFromStr("1"), map[sdk.Tx]bool{
			tx1: true,
			tx2: true,
		})
		s.Require().Error(lane.CheckOrder(sdk.Context{}, proposal))
	})

	s.Run("should not accept a proposal where transactions are out of order relative to other lanes", func() {
		tx1, err := testutils.CreateRandomTx(
			s.encodingConfig.TxConfig,
			s.accounts[0],
			0,
			2,
			0,
			sdk.NewCoin(s.gasTokenDenom, math.NewInt(1)),
		)
		s.Require().NoError(err)

		tx2, err := testutils.CreateRandomTx(
			s.encodingConfig.TxConfig,
			s.accounts[1],
			0,
			1,
			0,
			sdk.NewCoin(s.gasTokenDenom, math.NewInt(2)),
		)
		s.Require().NoError(err)

		mocklane := mocks.NewLane(s.T())
		mocklane.On("Match", sdk.Context{}, tx1).Return(true)
		mocklane.On("Match", sdk.Context{}, tx2).Return(false)

		lane := s.initLane(math.LegacyMustNewDecFromStr("1"), nil)
		lane.SetIgnoreList([]blockbuster.Lane{mocklane})

		proposal := []sdk.Tx{
			tx1,
			tx2,
		}

		s.Require().Error(lane.CheckOrder(sdk.Context{}, proposal))
	})
}

func (s *BaseTestSuite) initLane(
	maxBlockSpace math.LegacyDec,
	expectedExecution map[sdk.Tx]bool,
) *base.DefaultLane {
	config := blockbuster.NewBaseLaneConfig(
		log.NewTestLogger(s.T()),
		s.encodingConfig.TxConfig.TxEncoder(),
		s.encodingConfig.TxConfig.TxDecoder(),
		s.setUpAnteHandler(expectedExecution),
		maxBlockSpace,
	)

	return base.NewDefaultLane(config)
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
