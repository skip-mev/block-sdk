// Package networksuite provides a base test suite for tests that need a local network instance
package networksuite

import (
	"math/rand"
	"os"

	"cosmossdk.io/log"
	"cosmossdk.io/math"
	"github.com/cosmos/gogoproto/proto"
	"github.com/skip-mev/chaintestutil/network"
	"github.com/skip-mev/chaintestutil/sample"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	pruningtypes "cosmossdk.io/store/pruning/types"
	cmtrand "github.com/cometbft/cometbft/libs/rand"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/baseapp"
	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	"github.com/skip-mev/chaintestutil/account"

	"github.com/skip-mev/block-sdk/v2/tests/app"
	auctiontypes "github.com/skip-mev/block-sdk/v2/x/auction/types"
)

var (
	chainID = "chain-" + cmtrand.NewRand().Str(6)

	genBalance = sdk.NewCoin(sdk.DefaultBondDenom, math.NewInt(1000000000000000000))
)

// NetworkTestSuite is a test suite for query tests that initializes a network instance.
type NetworkTestSuite struct {
	suite.Suite

	NetworkSuite *network.TestSuite
	AuctionState auctiontypes.GenesisState
	AuthState    authtypes.GenesisState
	BankState    banktypes.GenesisState
	Accounts     []*account.Account
}

// SetupSuite setups the local network with a genesis state.
func (nts *NetworkTestSuite) SetupSuite() {
	var (
		r       = sample.Rand()
		cfg     = network.NewConfig(app.AppConfig)
		appCons = func(val network.ValidatorI) servertypes.Application {
			return app.New(
				log.NewLogger(os.Stdout),
				dbm.NewMemDB(),
				nil,
				true,
				simtestutil.NewAppOptionsWithFlagHome(val.GetCtx().Config.RootDir),
				baseapp.SetPruning(pruningtypes.NewPruningOptionsFromString(val.GetAppConfig().Pruning)),
				baseapp.SetMinGasPrices(val.GetAppConfig().MinGasPrices),
				baseapp.SetChainID(cfg.ChainID),
			)
		}
	)
	cfg.AppConstructor = appCons
	cfg.ChainID = chainID

	updateGenesisConfigState := func(moduleName string, moduleState proto.Message) {
		buf, err := cfg.Codec.MarshalJSON(moduleState)
		require.NoError(nts.T(), err)
		cfg.GenesisState[moduleName] = buf
	}

	// initialize genesis
	require.NoError(nts.T(), cfg.Codec.UnmarshalJSON(cfg.GenesisState[auctiontypes.ModuleName], &nts.AuctionState))
	nts.AuctionState = populateAuction(r, nts.AuctionState)
	updateGenesisConfigState(auctiontypes.ModuleName, &nts.AuctionState)

	// add genesis accounts
	nts.Accounts = []*account.Account{
		account.NewAccount(),
	}

	require.NoError(nts.T(), cfg.Codec.UnmarshalJSON(cfg.GenesisState[authtypes.ModuleName], &nts.AuthState))
	require.NoError(nts.T(), cfg.Codec.UnmarshalJSON(cfg.GenesisState[banktypes.ModuleName], &nts.BankState))

	addGenesisAccounts(&nts.AuthState, &nts.BankState, nts.Accounts)

	// update genesis
	updateGenesisConfigState(authtypes.ModuleName, &nts.AuthState)
	updateGenesisConfigState(banktypes.ModuleName, &nts.BankState)

	nts.NetworkSuite = network.NewSuite(nts.T(), cfg)
}

// addGenesisAccount adds a genesis account to the auth / bank genesis state.
func addGenesisAccounts(authGenState *authtypes.GenesisState, bankGenState *banktypes.GenesisState, accs []*account.Account) {
	balances := make([]banktypes.Balance, len(accs))
	accounts := make(authtypes.GenesisAccounts, len(accs))

	// create accounts / update bank state w/ account + gen balance
	for i, acc := range accs {
		// base account
		bacc := authtypes.NewBaseAccount(acc.Address(), acc.PubKey(), 0, 0)

		accounts[i] = bacc
		balances[i] = banktypes.Balance{
			Address: acc.Address().String(),
			Coins:   sdk.NewCoins(genBalance),
		}
	}

	// update auth state w/ accounts
	var err error
	authGenState.Accounts, err = authtypes.PackAccounts(accounts)
	if err != nil {
		panic(err)
	}

	// update bank state w/ balances
	bankGenState.Balances = balances
}

func populateAuction(_ *rand.Rand, auctionState auctiontypes.GenesisState) auctiontypes.GenesisState {
	// TODO intercept and populate state randomly if desired
	return auctionState
}
