// Package networksuite provides a base test suite for tests that need a local network instance
package networksuite

import (
	"math/rand"

	"github.com/cosmos/gogoproto/proto"
	"github.com/skip-mev/chaintestutil/network"
	"github.com/skip-mev/chaintestutil/sample"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/skip-mev/block-sdk/tests/app"
	auctiontypes "github.com/skip-mev/block-sdk/x/auction/types"
)

// NetworkTestSuite is a test suite for tests that initializes a network instance.
type NetworkTestSuite struct {
	suite.Suite

	NetworkSuite *network.TestSuite
	AuctionState auctiontypes.GenesisState
}

// SetupSuite setups the local network with a genesis state.
func (nts *NetworkTestSuite) SetupSuite() {
	var (
		r   = sample.Rand()
		cfg = network.NewConfig(app.AppConfig)
	)

	updateGenesisConfigState := func(moduleName string, moduleState proto.Message) {
		buf, err := cfg.Codec.MarshalJSON(moduleState)
		require.NoError(nts.T(), err)
		cfg.GenesisState[moduleName] = buf
	}

	// initialize genesis
	require.NoError(nts.T(), cfg.Codec.UnmarshalJSON(cfg.GenesisState[auctiontypes.ModuleName], &nts.AuctionState))
	nts.AuctionState = populateAuction(r, nts.AuctionState)
	updateGenesisConfigState(auctiontypes.ModuleName, &nts.AuctionState)

	nts.NetworkSuite = network.NewSuite(nts.T(), cfg)
}

func populateAuction(_ *rand.Rand, auctionState auctiontypes.GenesisState) auctiontypes.GenesisState {
	// TODO intercept and populate state randomly if desired
	return auctionState
}
