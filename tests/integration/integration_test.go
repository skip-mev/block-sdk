package integration_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/skip-mev/chaintestutil/encoding"
	"github.com/stretchr/testify/suite"

<<<<<<< HEAD
	"github.com/skip-mev/block-sdk/tests/app"
	testkeeper "github.com/skip-mev/block-sdk/testutils/keeper"
=======
	testkeeper "github.com/skip-mev/block-sdk/v2/testutils/keeper"
	auctiontypes "github.com/skip-mev/block-sdk/v2/x/auction/types"
>>>>>>> 97e5731 (rm (#530))
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
<<<<<<< HEAD
	s.encCfg = encoding.MakeTestEncodingConfig(app.ModuleBasics.RegisterInterfaces)
=======
	s.encCfg = encoding.MakeTestEncodingConfig(func(registry types.InterfaceRegistry) {
		auctiontypes.RegisterInterfaces(registry)
	})
>>>>>>> 97e5731 (rm (#530))

	s.ctx, s.TestKeepers, s.TestMsgServers = testkeeper.NewTestSetup(s.T())
}
