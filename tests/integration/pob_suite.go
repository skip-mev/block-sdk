package integration

import (
	"context"

	interchaintest "github.com/strangelove-ventures/interchaintest/v7"
	"github.com/strangelove-ventures/interchaintest/v7/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v7/ibc"
	"github.com/stretchr/testify/suite"
)

const (
	initBalance = 10000000000
)

// POBIntegrationTestSuite runs the POB integration test-suite against a given interchaintest specification
type POBIntegrationTestSuite struct {
	suite.Suite
	// spec
	spec *interchaintest.ChainSpec
	// chain
	chain ibc.Chain
	// interchain
	ic *interchaintest.Interchain
	// users
	user1, user2, user3 cosmos.User
}

func NewPOBIntegrationTestSuiteFromSpec(spec *interchaintest.ChainSpec) *POBIntegrationTestSuite {
	return &POBIntegrationTestSuite{
		spec: spec,
	}
}

func (s *POBIntegrationTestSuite) SetupSuite() {
	// build the chain
	s.T().Log("building chain with spec", s.spec)
	chain := ChainBuilderFromChainSpec(s.T(), s.spec)

	// build the interchain
	s.T().Log("building interchain")
	ctx := context.Background()
	s.ic = BuildPOBInterchain(s.T(), ctx, chain)

	// get the users
	s.user1 = interchaintest.GetAndFundTestUsers(s.T(), ctx, s.T().Name(), initBalance, chain)[0]
	s.user2 = interchaintest.GetAndFundTestUsers(s.T(), ctx, s.T().Name(), initBalance, chain)[0]
	s.user3 = interchaintest.GetAndFundTestUsers(s.T(), ctx, s.T().Name(), initBalance, chain)[0]
}

func (s *POBIntegrationTestSuite) TearDownSuite() {
	// close the interchain
	s.ic.Close()
}
