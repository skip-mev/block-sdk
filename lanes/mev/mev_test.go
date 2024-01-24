package mev_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/suite"

	"github.com/skip-mev/block-sdk/v2/block/utils"
	"github.com/skip-mev/block-sdk/v2/lanes/mev/testutils"
)

type MEVTestSuite struct {
	testutils.MEVLaneTestSuiteBase
}

func TestMEVTestSuite(t *testing.T) {
	suite.Run(t, new(MEVTestSuite))
}

func (s *MEVTestSuite) getTxSize(tx sdk.Tx) int64 {
	txBz, err := s.EncCfg.TxConfig.TxEncoder()(tx)
	s.Require().NoError(err)

	return int64(len(txBz))
}

func (s *MEVTestSuite) compare(first, second []sdk.Tx) {
	firstBytes, err := utils.GetEncodedTxs(s.EncCfg.TxConfig.TxEncoder(), first)
	s.Require().NoError(err)

	secondBytes, err := utils.GetEncodedTxs(s.EncCfg.TxConfig.TxEncoder(), second)
	s.Require().NoError(err)

	s.Require().Equal(firstBytes, secondBytes)
}
