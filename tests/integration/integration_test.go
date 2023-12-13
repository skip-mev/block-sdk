package integration_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/skip-mev/chaintestutil/encoding"
	"github.com/stretchr/testify/suite"

	"github.com/skip-mev/block-sdk/tests/app"
	testkeeper "github.com/skip-mev/block-sdk/testutils/keeper"
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
	s.encCfg = encoding.MakeTestEncodingConfig(app.ModuleBasics.RegisterInterfaces)

	s.ctx, s.TestKeepers, s.TestMsgServers = testkeeper.NewTestSetup(s.T())
}
