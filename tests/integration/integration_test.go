package integration_test

import (
	"testing"

	"github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/skip-mev/chaintestutil/encoding"
	"github.com/stretchr/testify/suite"

	testkeeper "github.com/skip-mev/block-sdk/testutils/keeper"
	auctiontypes "github.com/skip-mev/block-sdk/x/auction/types"
	blocksdktypes "github.com/skip-mev/block-sdk/x/blocksdk/types"
)

type IntegrationTestSuite struct {
	suite.Suite
	testkeeper.TestKeepers
	testkeeper.TestMsgServers

	encCfg encoding.TestEncodingConfig
	ctx    sdk.Context
}

func TestIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(IntegrationTestSuite))
}

func (s *IntegrationTestSuite) SetupTest() {
	s.encCfg = encoding.MakeTestEncodingConfig(func(registry types.InterfaceRegistry) {
		auctiontypes.RegisterInterfaces(registry)
		blocksdktypes.RegisterInterfaces(registry)
	})

	s.ctx, s.TestKeepers, s.TestMsgServers = testkeeper.NewTestSetup(s.T())
}
