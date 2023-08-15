package base_test

import (
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/skip-mev/pob/blockbuster"
	testutils "github.com/skip-mev/pob/testutils"
)

func (s *BaseTestSuite) TestGetTxPriority() {
	txPriority := blockbuster.DefaultTxPriority()

	s.Run("should be able to get the priority off a normal transaction with fees", func() {
		tx, err := testutils.CreateRandomTx(
			s.encodingConfig.TxConfig,
			s.accounts[0],
			0,
			0,
			0,
			sdk.NewCoin(s.gasTokenDenom, math.NewInt(100)),
		)
		s.Require().NoError(err)

		priority := txPriority.GetTxPriority(sdk.Context{}, tx)
		s.Require().Equal(sdk.NewCoin(s.gasTokenDenom, math.NewInt(100)).String(), priority)
	})

	s.Run("should not get a priority when the transaction does not have a fee", func() {
		tx, err := testutils.CreateRandomTx(
			s.encodingConfig.TxConfig,
			s.accounts[0],
			0,
			0,
			0,
		)
		s.Require().NoError(err)

		priority := txPriority.GetTxPriority(sdk.Context{}, tx)
		s.Require().Equal("", priority)
	})

	s.Run("should get a priority when the gas token is different", func() {
		tx, err := testutils.CreateRandomTx(
			s.encodingConfig.TxConfig,
			s.accounts[0],
			0,
			0,
			0,
			sdk.NewCoin("random", math.NewInt(100)),
		)
		s.Require().NoError(err)

		priority := txPriority.GetTxPriority(sdk.Context{}, tx)
		s.Require().Equal(sdk.NewCoin("random", math.NewInt(100)).String(), priority)
	})
}

func (s *BaseTestSuite) TestCompareTxPriority() {
	txPriority := blockbuster.DefaultTxPriority()

	s.Run("should return 0 when both priorities are nil", func() {
		a := sdk.NewCoin(s.gasTokenDenom, math.NewInt(0)).String()
		b := sdk.NewCoin(s.gasTokenDenom, math.NewInt(0)).String()
		s.Require().Equal(0, txPriority.Compare(a, b))
	})

	s.Run("should return 1 when the first priority is greater", func() {
		a := sdk.NewCoin(s.gasTokenDenom, math.NewInt(100)).String()
		b := sdk.NewCoin(s.gasTokenDenom, math.NewInt(1)).String()
		s.Require().Equal(1, txPriority.Compare(a, b))
	})

	s.Run("should return -1 when the second priority is greater", func() {
		a := sdk.NewCoin(s.gasTokenDenom, math.NewInt(1)).String()
		b := sdk.NewCoin(s.gasTokenDenom, math.NewInt(100)).String()
		s.Require().Equal(-1, txPriority.Compare(a, b))
	})

	s.Run("should return 0 when both priorities are equal", func() {
		a := sdk.NewCoin(s.gasTokenDenom, math.NewInt(100)).String()
		b := sdk.NewCoin(s.gasTokenDenom, math.NewInt(100)).String()
		s.Require().Equal(0, txPriority.Compare(a, b))
	})
}

func (s *BaseTestSuite) TestInsert() {
	mempool := blockbuster.NewConstructorMempool[string](blockbuster.DefaultTxPriority(), s.encodingConfig.TxConfig.TxEncoder(), 3)

	s.Run("should be able to insert a transaction", func() {
		tx, err := testutils.CreateRandomTx(
			s.encodingConfig.TxConfig,
			s.accounts[0],
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
			sdk.NewCoin(s.gasTokenDenom, math.NewInt(100)),
		)
		s.Require().NoError(err)

		err = mempool.Insert(sdk.Context{}, tx)
		s.Require().Error(err)
		s.Require().False(mempool.Contains(tx))
	})
}

func (s *BaseTestSuite) TestRemove() {
	mempool := blockbuster.NewConstructorMempool[string](blockbuster.DefaultTxPriority(), s.encodingConfig.TxConfig.TxEncoder(), 3)

	s.Run("should be able to remove a transaction", func() {
		tx, err := testutils.CreateRandomTx(
			s.encodingConfig.TxConfig,
			s.accounts[0],
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
			sdk.NewCoin(s.gasTokenDenom, math.NewInt(100)),
		)
		s.Require().NoError(err)

		mempool.Remove(tx)
	})
}

func (s *BaseTestSuite) TestSelect() {
	s.Run("should be able to select transactions in the correct order", func() {
		mempool := blockbuster.NewConstructorMempool[string](blockbuster.DefaultTxPriority(), s.encodingConfig.TxConfig.TxEncoder(), 3)

		tx1, err := testutils.CreateRandomTx(
			s.encodingConfig.TxConfig,
			s.accounts[0],
			0,
			0,
			0,
			sdk.NewCoin(s.gasTokenDenom, math.NewInt(100)),
		)
		s.Require().NoError(err)

		tx2, err := testutils.CreateRandomTx(
			s.encodingConfig.TxConfig,
			s.accounts[1],
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
		s.Require().Equal(tx2, iterator.Tx())

		// Check the second transaction
		iterator = iterator.Next()
		s.Require().NotNil(iterator)
		s.Require().Equal(tx1, iterator.Tx())
	})

	s.Run("should be able to select a single transaction", func() {
		mempool := blockbuster.NewConstructorMempool[string](blockbuster.DefaultTxPriority(), s.encodingConfig.TxConfig.TxEncoder(), 3)

		tx1, err := testutils.CreateRandomTx(
			s.encodingConfig.TxConfig,
			s.accounts[0],
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
