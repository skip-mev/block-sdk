package base_test

import (
	"math/rand"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/skip-mev/block-sdk/v2/testutils"
)

func (s *BaseTestSuite) TestGetTxInfo() {
	accounts := testutils.RandomAccounts(rand.New(rand.NewSource(1)), 3)
	lane := s.initLane(math.LegacyOneDec(), nil)

	s.Run("can retrieve information for a default tx", func() {
		signer := accounts[0]
		nonce := uint64(1)
		fee := sdk.NewCoins(sdk.NewCoin(s.gasTokenDenom, math.NewInt(100)))
		gasLimit := uint64(100)

		tx, err := testutils.CreateRandomTx(
			s.encodingConfig.TxConfig,
			signer,
			nonce,
			1,
			0,
			gasLimit,
			fee...,
		)
		s.Require().NoError(err)

		txInfo, err := lane.GetTxInfo(s.ctx, tx)
		s.Require().NoError(err)
		s.Require().NotEmpty(txInfo.Hash)

		// Verify the signers
		s.Require().Len(txInfo.Signers, 1)
		s.Require().Equal(signer.Address.String(), txInfo.Signers[0].Signer.String())
		s.Require().Equal(nonce, txInfo.Signers[0].Sequence)

		// Verify the gas limit
		s.Require().Equal(gasLimit, txInfo.GasLimit)

		// Verify the bytes
		txBz, err := s.encodingConfig.TxConfig.TxEncoder()(tx)
		s.Require().NoError(err)
		s.Require().Equal(txBz, txInfo.TxBytes)

		// Verify the size
		s.Require().Equal(int64(len(txBz)), txInfo.Size)
	})

	s.Run("can retrieve information with different fees", func() {
		signer := accounts[1]
		nonce := uint64(10)
		fee := sdk.NewCoins(sdk.NewCoin(s.gasTokenDenom, math.NewInt(20000)))
		gasLimit := uint64(10000000)

		tx, err := testutils.CreateRandomTx(
			s.encodingConfig.TxConfig,
			signer,
			nonce,
			10,
			0,
			gasLimit,
			fee...,
		)
		s.Require().NoError(err)

		txInfo, err := lane.GetTxInfo(s.ctx, tx)
		s.Require().NoError(err)
		s.Require().NotEmpty(txInfo.Hash)

		// Verify the signers
		s.Require().Len(txInfo.Signers, 1)
		s.Require().Equal(signer.Address.String(), txInfo.Signers[0].Signer.String())
		s.Require().Equal(nonce, txInfo.Signers[0].Sequence)

		// Verify the bytes
		txBz, err := s.encodingConfig.TxConfig.TxEncoder()(tx)
		s.Require().NoError(err)
		s.Require().Equal(txBz, txInfo.TxBytes)
	})
}
