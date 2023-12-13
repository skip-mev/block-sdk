// Package networksuite provides a base test suite for tests that need a local network instance
package networksuite

import (
	"math/rand"

	tmdb "github.com/cometbft/cometbft-db"
	tmrand "github.com/cometbft/cometbft/libs/rand"
	"github.com/cosmos/cosmos-sdk/baseapp"
	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	pruningtypes "github.com/cosmos/cosmos-sdk/store/pruning/types"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	"github.com/cosmos/gogoproto/proto"
	"github.com/skip-mev/chaintestutil/network"
	"github.com/skip-mev/chaintestutil/sample"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/skip-mev/block-sdk/tests/app"
	auctiontypes "github.com/skip-mev/block-sdk/x/auction/types"
)

var (
	chainID = "chain-" + tmrand.NewRand().Str(6)

	DefaultAppConstructor = func(val network.ValidatorI) servertypes.Application {
		return app.New(
			val.GetCtx().Logger,
			tmdb.NewMemDB(),
			nil,
			true,
			simtestutil.EmptyAppOptions{},
			baseapp.SetPruning(pruningtypes.NewPruningOptionsFromString(val.GetAppConfig().Pruning)),
			baseapp.SetMinGasPrices(val.GetAppConfig().MinGasPrices),
			baseapp.SetChainID(chainID),
		)
	}
)

// NetworkTestSuite is a test suite for tests that initializes a network instance.
type NetworkTestSuite struct {
	suite.Suite

<<<<<<< HEAD
	Network      *network.Network
	AuctionState auctiontypes.GenesisState
=======
	NetworkSuite  *network.TestSuite
	AuctionState  auctiontypes.GenesisState
	BlockSDKState blocksdktypes.GenesisState
>>>>>>> af3bb52 (refactor(tests): use grpc instead of cli for all network testing (#301))
}

// SetupSuite setups the local network with a genesis state.
func (nts *NetworkTestSuite) SetupSuite() {
	var (
		r   = sample.Rand()
		cfg = network.NewConfig(DefaultAppConstructor, app.ModuleBasics, chainID)
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

<<<<<<< HEAD
	nts.Network = network.New(nts.T(), cfg)
=======
	nts.BlockSDKState = populateBlockSDK(r, nts.BlockSDKState)
	updateGenesisConfigState(blocksdktypes.ModuleName, &nts.BlockSDKState)

	nts.NetworkSuite = network.NewSuite(nts.T(), cfg)
>>>>>>> af3bb52 (refactor(tests): use grpc instead of cli for all network testing (#301))
}

func populateAuction(_ *rand.Rand, auctionState auctiontypes.GenesisState) auctiontypes.GenesisState {
	// TODO intercept and populate state randomly if desired
	return auctionState
}
