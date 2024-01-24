package integration_test

import (
	"testing"

	"github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/skip-mev/chaintestutil/encoding"
	"github.com/stretchr/testify/suite"

<<<<<<< HEAD
	testkeeper "github.com/skip-mev/block-sdk/testutils/keeper"
	auctiontypes "github.com/skip-mev/block-sdk/x/auction/types"
=======
	testkeeper "github.com/skip-mev/block-sdk/v2/testutils/keeper"
	auctiontypes "github.com/skip-mev/block-sdk/v2/x/auction/types"
	blocksdktypes "github.com/skip-mev/block-sdk/v2/x/blocksdk/types"
>>>>>>> acff9d0 (chore: Upgrade module path for v2 (#383))
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
	})

	s.ctx, s.TestKeepers, s.TestMsgServers = testkeeper.NewTestSetup(s.T())
}
