package base_test

import (
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	signer_extraction "github.com/skip-mev/block-sdk/v2/adapters/signer_extraction_adapter"
	"github.com/skip-mev/block-sdk/v2/block/base"
	testutils "github.com/skip-mev/block-sdk/v2/testutils"
)

func (s *BaseTestSuite) TestCompareTxPriority() {
	lane := s.initLane(math.LegacyOneDec(), nil)

	s.Run("should return -1 when signers are the same but the first tx has a higher sequence", func() {
		tx1, err := testutils.CreateRandomTx(
			s.encodingConfig.TxConfig,
			s.accounts[0],
			1,
			0,
			0,
			0,
			sdk.NewCoin(s.gasTokenDenom, math.NewInt(100)),
		)
		s.Require().NoError(err)

		tx2, err := testutils.CreateRandomTx(
			s.encodingConfig.TxConfig,
			s.accounts[0],
			0,
			0,
			0,
			0,
			sdk.NewCoin(s.gasTokenDenom, math.NewInt(100)),
		)
		s.Require().NoError(err)

		cmp, err := lane.Compare(sdk.Context{}, tx1, tx2)
		s.Require().NoError(err)
		s.Require().Equal(-1, cmp)
	})

	s.Run("should return 1 when signers are the same but the second tx has a higher sequence", func() {
		tx1, err := testutils.CreateRandomTx(
			s.encodingConfig.TxConfig,
			s.accounts[0],
			0,
			0,
			0,
			0,
			sdk.NewCoin(s.gasTokenDenom, math.NewInt(100)),
		)
		s.Require().NoError(err)

		tx2, err := testutils.CreateRandomTx(
			s.encodingConfig.TxConfig,
			s.accounts[0],
			1,
			0,
			0,
			0,
			sdk.NewCoin(s.gasTokenDenom, math.NewInt(100)),
		)
		s.Require().NoError(err)

		cmp, err := lane.Compare(sdk.Context{}, tx1, tx2)
		s.Require().NoError(err)
		s.Require().Equal(1, cmp)
	})

	s.Run("should return 0 when signers are the same and the sequence is the same", func() {
		tx1, err := testutils.CreateRandomTx(
			s.encodingConfig.TxConfig,
			s.accounts[0],
			1,
			0,
			0,
			0,
			sdk.NewCoin(s.gasTokenDenom, math.NewInt(100)),
		)
		s.Require().NoError(err)

		tx2, err := testutils.CreateRandomTx(
			s.encodingConfig.TxConfig,
			s.accounts[0],
			1,
			0,
			0,
			0,
			sdk.NewCoin(s.gasTokenDenom, math.NewInt(100)),
		)
		s.Require().NoError(err)

		_, err = lane.Compare(sdk.Context{}, tx1, tx2)
		s.Require().Error(err)
	})

	s.Run("should return 0 when the first tx has a higher fee", func() {
		tx1, err := testutils.CreateRandomTx(
			s.encodingConfig.TxConfig,
			s.accounts[0],
			0,
			0,
			0,
			0,
			sdk.NewCoin(s.gasTokenDenom, math.NewInt(200)),
		)
		s.Require().NoError(err)

		tx2, err := testutils.CreateRandomTx(
			s.encodingConfig.TxConfig,
			s.accounts[1],
			0,
			0,
			0,
			0,
			sdk.NewCoin(s.gasTokenDenom, math.NewInt(100)),
		)
		s.Require().NoError(err)

		cmp, err := lane.Compare(sdk.Context{}, tx1, tx2)
		s.Require().NoError(err)
		s.Require().Equal(0, cmp)
	})
}

func (s *BaseTestSuite) TestInsert() {
	mempool := base.NewMempool(base.DefaultTxPriority(), signer_extraction.NewDefaultAdapter(), 3)

	s.Run("should be able to insert a transaction", func() {
		tx, err := testutils.CreateRandomTx(
			s.encodingConfig.TxConfig,
			s.accounts[0],
			0,
			0,
			0,
			0,
			sdk.NewCoin(s.gasTokenDenom, math.NewInt(100)),
		)
		s.Require().NoError(err)

		err = mempool.Insert(sdk.Context{}, tx)
		s.Require().NoError(err)
		s.Require().True(mempool.Contains(tx))
	})

	s.Run("cannot insert more transactions than the max", func() {
		for i := 0; i < 3; i++ {
			tx, err := testutils.CreateRandomTx(
				s.encodingConfig.TxConfig,
				s.accounts[0],
				uint64(i),
				0,
				0,
				0,
				sdk.NewCoin(s.gasTokenDenom, math.NewInt(int64(100*i))),
			)
			s.Require().NoError(err)

			err = mempool.Insert(sdk.Context{}, tx)
			s.Require().NoError(err)
			s.Require().True(mempool.Contains(tx))
		}

		tx, err := testutils.CreateRandomTx(
			s.encodingConfig.TxConfig,
			s.accounts[0],
			10,
			0,
			0,
			0,
			sdk.NewCoin(s.gasTokenDenom, math.NewInt(100)),
		)
		s.Require().NoError(err)

		err = mempool.Insert(sdk.Context{}, tx)
		s.Require().Error(err)
		s.Require().False(mempool.Contains(tx))
	})
}

func (s *BaseTestSuite) TestRemove() {
	mempool := base.NewMempool(base.DefaultTxPriority(), signer_extraction.NewDefaultAdapter(), 3)

	s.Run("should be able to remove a transaction", func() {
		tx, err := testutils.CreateRandomTx(
			s.encodingConfig.TxConfig,
			s.accounts[0],
			0,
			0,
			0,
			0,
			sdk.NewCoin(s.gasTokenDenom, math.NewInt(100)),
		)
		s.Require().NoError(err)

		err = mempool.Insert(sdk.Context{}, tx)
		s.Require().NoError(err)
		s.Require().True(mempool.Contains(tx))

		mempool.Remove(tx)
		s.Require().False(mempool.Contains(tx))
	})

	s.Run("should not error when removing a transaction that does not exist", func() {
		tx, err := testutils.CreateRandomTx(
			s.encodingConfig.TxConfig,
			s.accounts[0],
			0,
			0,
			0,
			0,
			sdk.NewCoin(s.gasTokenDenom, math.NewInt(100)),
		)
		s.Require().NoError(err)

		mempool.Remove(tx)
	})
}

func (s *BaseTestSuite) TestSelect() {
	s.Run("should be able to select transactions in the correct order", func() {
		mempool := base.NewMempool(base.DefaultTxPriority(), signer_extraction.NewDefaultAdapter(), 3)

		tx1, err := testutils.CreateRandomTx(
			s.encodingConfig.TxConfig,
			s.accounts[1],
			0,
			0,
			0,
			0,
			sdk.NewCoin(s.gasTokenDenom, math.NewInt(100)),
		)
		s.Require().NoError(err)

		tx2, err := testutils.CreateRandomTx(
			s.encodingConfig.TxConfig,
			s.accounts[1],
			1,
			0,
			0,
			0,
			sdk.NewCoin(s.gasTokenDenom, math.NewInt(200)),
		)
		s.Require().NoError(err)

		// Insert the transactions into the mempool
		s.Require().NoError(mempool.Insert(sdk.Context{}, tx1))
		s.Require().NoError(mempool.Insert(sdk.Context{}, tx2))
		s.Require().Equal(2, mempool.CountTx())

		// Check that the transactions are in the correct order
		iterator := mempool.Select(sdk.Context{}, nil)
		s.Require().NotNil(iterator)
		s.Require().Equal(tx1, iterator.Tx())

		// Check the second transaction
		iterator = iterator.Next()
		s.Require().NotNil(iterator)
		s.Require().Equal(tx2, iterator.Tx())
	})

	s.Run("should be able to select a single transaction", func() {
		mempool := base.NewMempool(base.DefaultTxPriority(), signer_extraction.NewDefaultAdapter(), 3)

		tx1, err := testutils.CreateRandomTx(
			s.encodingConfig.TxConfig,
			s.accounts[0],
			0,
			0,
			0,
			0,
			sdk.NewCoin(s.gasTokenDenom, math.NewInt(100)),
		)
		s.Require().NoError(err)

		// Insert the transactions into the mempool
		s.Require().NoError(mempool.Insert(sdk.Context{}, tx1))
		s.Require().Equal(1, mempool.CountTx())

		// Check that the transactions are in the correct order
		iterator := mempool.Select(sdk.Context{}, nil)
		s.Require().NotNil(iterator)
		s.Require().Equal(tx1, iterator.Tx())

		iterator = iterator.Next()
		s.Require().Nil(iterator)
	})
}
