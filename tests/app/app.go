package app

import (
	"io"
	"os"
	"path/filepath"

	"cosmossdk.io/log"
	dbm "github.com/cosmos/cosmos-db"

	"cosmossdk.io/depinject"
	storetypes "cosmossdk.io/store/types"
	circuitkeeper "cosmossdk.io/x/circuit/keeper"
	upgradekeeper "cosmossdk.io/x/upgrade/keeper"

	feegrantkeeper "cosmossdk.io/x/feegrant/keeper"
	cometabci "github.com/cometbft/cometbft/abci/types"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/server"
	"github.com/cosmos/cosmos-sdk/server/api"
	"github.com/cosmos/cosmos-sdk/server/config"
	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/cosmos/cosmos-sdk/x/auth/ante"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	authzkeeper "github.com/cosmos/cosmos-sdk/x/authz/keeper"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	consensuskeeper "github.com/cosmos/cosmos-sdk/x/consensus/keeper"
	crisiskeeper "github.com/cosmos/cosmos-sdk/x/crisis/keeper"
	distrkeeper "github.com/cosmos/cosmos-sdk/x/distribution/keeper"
	govkeeper "github.com/cosmos/cosmos-sdk/x/gov/keeper"
	groupkeeper "github.com/cosmos/cosmos-sdk/x/group/keeper"
	mintkeeper "github.com/cosmos/cosmos-sdk/x/mint/keeper"
	paramskeeper "github.com/cosmos/cosmos-sdk/x/params/keeper"
	paramstypes "github.com/cosmos/cosmos-sdk/x/params/types"
	slashingkeeper "github.com/cosmos/cosmos-sdk/x/slashing/keeper"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"

	"github.com/skip-mev/block-sdk/v2/abci"
	"github.com/skip-mev/block-sdk/v2/abci/checktx"
	"github.com/skip-mev/block-sdk/v2/block"
	"github.com/skip-mev/block-sdk/v2/block/base"
	service "github.com/skip-mev/block-sdk/v2/block/service"
	"github.com/skip-mev/block-sdk/v2/block/utils"
	auctionkeeper "github.com/skip-mev/block-sdk/v2/x/auction/keeper"
)

const (
	ChainID = "chain-id-0"
)

var (
	BondDenom = sdk.DefaultBondDenom

	// DefaultNodeHome default home directories for the application daemon
	DefaultNodeHome string
)

var (
	_ runtime.AppI            = (*TestApp)(nil)
	_ servertypes.Application = (*TestApp)(nil)
)

type TestApp struct {
	*runtime.App
	legacyAmino       *codec.LegacyAmino
	appCodec          codec.Codec
	txConfig          client.TxConfig
	interfaceRegistry codectypes.InterfaceRegistry

	// keepers
	AccountKeeper         authkeeper.AccountKeeper
	BankKeeper            bankkeeper.Keeper
	StakingKeeper         *stakingkeeper.Keeper
	SlashingKeeper        slashingkeeper.Keeper
	MintKeeper            mintkeeper.Keeper
	DistrKeeper           distrkeeper.Keeper
	GovKeeper             *govkeeper.Keeper
	CrisisKeeper          *crisiskeeper.Keeper
	UpgradeKeeper         *upgradekeeper.Keeper
	ParamsKeeper          paramskeeper.Keeper
	AuthzKeeper           authzkeeper.Keeper
	GroupKeeper           groupkeeper.Keeper
	ConsensusParamsKeeper consensuskeeper.Keeper
	CircuitBreakerKeeper  circuitkeeper.Keeper
	auctionkeeper         auctionkeeper.Keeper
	FeeGrantKeeper        feegrantkeeper.Keeper

	// custom checkTx handler
	checkTxHandler checktx.CheckTx
}

func init() {
	userHomeDir, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}

	DefaultNodeHome = filepath.Join(userHomeDir, ".testapp")
}

func New(
	logger log.Logger,
	db dbm.DB,
	traceStore io.Writer,
	loadLatest bool,
	appOpts servertypes.AppOptions,
	baseAppOptions ...func(*baseapp.BaseApp),
) *TestApp {
	var (
		app        = &TestApp{}
		appBuilder *runtime.AppBuilder

		// merge the AppConfig and other configuration in one config
		appConfig = depinject.Configs(
			AppConfig,
			depinject.Supply(
				// supply the application options
				appOpts,

				logger,

				// ADVANCED CONFIGURATION

				//
				// AUTH
				//
				// For providing a custom function required in auth to generate custom account types
				// add it below. By default the auth module uses simulation.RandomGenesisAccounts.
				//
				// authtypes.RandomGenesisAccountsFn(simulation.RandomGenesisAccounts),

				// For providing a custom a base account type add it below.
				// By default the auth module uses authtypes.ProtoBaseAccount().
				//
				// func() authtypes.AccountI { return authtypes.ProtoBaseAccount() },

				//
				// MINT
				//

				// For providing a custom inflation function for x/mint add here your
				// custom function that implements the minttypes.InflationCalculationFn
				// interface.
			),
		)
	)

	if err := depinject.Inject(appConfig,
		&appBuilder,
		&app.appCodec,
		&app.legacyAmino,
		&app.txConfig,
		&app.interfaceRegistry,
		&app.AccountKeeper,
		&app.BankKeeper,
		&app.StakingKeeper,
		&app.SlashingKeeper,
		&app.MintKeeper,
		&app.DistrKeeper,
		&app.GovKeeper,
		&app.CrisisKeeper,
		&app.UpgradeKeeper,
		&app.ParamsKeeper,
		&app.AuthzKeeper,
		&app.GroupKeeper,
		&app.auctionkeeper,
		&app.ConsensusParamsKeeper,
		&app.FeeGrantKeeper,
		&app.CircuitBreakerKeeper,
	); err != nil {
		panic(err)
	}

	// Below we could construct and set an application specific mempool and
	// ABCI 1.0 PrepareProposal and ProcessProposal handlers. These defaults are
	// already set in the SDK's BaseApp, this shows an example of how to override
	// them.
	//
	// Example:
	//
	// app.App = appBuilder.Build(...)
	// nonceMempool := mempool.NewSenderNonceMempool()
	// abciPropHandler := NewDefaultProposalHandler(nonceMempool, app.App.BaseApp)
	//
	// app.App.BaseApp.SetMempool(nonceMempool)
	// app.App.BaseApp.SetPrepareProposal(abciPropHandler.PrepareProposalHandler())
	// app.App.BaseApp.SetProcessProposal(abciPropHandler.ProcessProposalHandler())
	//
	// Alternatively, you can construct BaseApp options, append those to
	// baseAppOptions and pass them to the appBuilder.
	//
	// Example:
	//
	// prepareOpt = func(app *baseapp.BaseApp) {
	// 	abciPropHandler := baseapp.NewDefaultProposalHandler(nonceMempool, app)
	// 	app.SetPrepareProposal(abciPropHandler.PrepareProposalHandler())
	// }
	// baseAppOptions = append(baseAppOptions, prepareOpt)

	app.App = appBuilder.Build(db, traceStore, baseAppOptions...)

	// ---------------------------------------------------------------------------- //
	// ------------------------- Begin Custom Code -------------------------------- //
	// ---------------------------------------------------------------------------- //
	// STEP 1-3: Create the Block SDK lanes.
	mevLane, freeLane, defaultLane := CreateLanes(app)

	// STEP 4: Construct a mempool based off the lanes. Note that the order of the lanes
	// matters. Blocks are constructed from the top lane to the bottom lane. The top lane
	// is the first lane in the array and the bottom lane is the last lane in the array.
	mempool, err := block.NewLanedMempool(
		app.Logger(),
		[]block.Lane{mevLane, freeLane, defaultLane},
	)
	if err != nil {
		panic(err)
	}

	// The application's mempool is now powered by the Block SDK!
	app.App.SetMempool(mempool)

	// STEP 5: Create a global ante handler that will be called on each transaction when
	// proposals are being built and verified. Note that this step must be done before
	// setting the ante handler on the lanes.
	handlerOptions := ante.HandlerOptions{
		AccountKeeper:   app.AccountKeeper,
		BankKeeper:      app.BankKeeper,
		FeegrantKeeper:  app.FeeGrantKeeper,
		SigGasConsumer:  ante.DefaultSigVerificationGasConsumer,
		SignModeHandler: app.txConfig.SignModeHandler(),
	}
	options := BSDKHandlerOptions{
		BaseOptions:   handlerOptions,
		auctionkeeper: app.auctionkeeper,
		TxDecoder:     app.txConfig.TxDecoder(),
		TxEncoder:     app.txConfig.TxEncoder(),
		FreeLane:      freeLane,
		MEVLane:       mevLane,
	}
	anteHandler := NewBSDKAnteHandler(options)
	app.App.SetAnteHandler(anteHandler)

	// Set the ante handler on the lanes.
	opt := []base.LaneOption{
		base.WithAnteHandler(anteHandler),
	}
	mevLane.WithOptions(
		opt...,
	)
	freeLane.WithOptions(
		opt...,
	)
	defaultLane.WithOptions(
		opt...,
	)

	// Step 6: Create the proposal handler and set it on the app. Now the application
	// will build and verify proposals using the Block SDK!
	//
	// NOTE: It is recommended to use the default proposal handler by constructing
	// using the NewDefaultProposalHandler function. This will use the correct prepare logic
	// for the lanes, but the process logic will be a no-op. To read more about the default
	// proposal handler, see the documentation in readme.md in this directory.
	//
	// If you want to customize the prepare and process logic, you can construct the proposal
	// handler using the New function and setting the useProcess flag to true.
	proposalHandler := abci.New( // use NewDefaultProposalHandler instead for default behavior (RECOMMENDED)
		app.Logger(),
		app.TxConfig().TxDecoder(),
		app.TxConfig().TxEncoder(),
		mempool,
		true,
	)
	app.App.SetPrepareProposal(proposalHandler.PrepareProposalHandler())
	app.App.SetProcessProposal(proposalHandler.ProcessProposalHandler())

	cacheDecoder, err := utils.NewDefaultCacheTxDecoder(app.txConfig.TxDecoder())
	if err != nil {
		panic(err)
	}

	// Step 7: Set the custom CheckTx handler on BaseApp. This is only required if you
	// use the MEV lane.
	mevCheckTx := checktx.NewMEVCheckTxHandler(
		app.App,
		cacheDecoder.TxDecoder(),
		mevLane,
		anteHandler,
		app.App.CheckTx,
	)
	checkTxHandler := checktx.NewMempoolParityCheckTx(
		app.Logger(),
		mempool,
		cacheDecoder.TxDecoder(),
		mevCheckTx.CheckTx(),
	)

	app.SetCheckTx(checkTxHandler.CheckTx())

	// ---------------------------------------------------------------------------- //
	// ------------------------- End Custom Code ---------------------------------- //
	// ---------------------------------------------------------------------------- //

	/****  Module Options ****/

	app.ModuleManager.RegisterInvariants(app.CrisisKeeper)

	// RegisterUpgradeHandlers is used for registering any on-chain upgrades.
	// app.RegisterUpgradeHandlers()

	// add test gRPC service for testing gRPC queries in isolation
	// testdata_pulsar.RegisterQueryServer(app.GRPCQueryRouter(), testdata_pulsar.QueryImpl{})

	// A custom InitChainer can be set if extra pre-init-genesis logic is required.
	// By default, when using app wiring enabled module, this is not required.
	// For instance, the upgrade module will set automatically the module version map in its init genesis thanks to app wiring.
	// However, when registering a module manually (i.e. that does not support app wiring), the module version map
	// must be set manually as follow. The upgrade module will de-duplicate the module version map.
	//
	// app.SetInitChainer(func(ctx sdk.Context, req abci.RequestInitChain) abci.ResponseInitChain {
	// 	app.UpgradeKeeper.SetModuleVersionMap(ctx, app.ModuleManager.GetVersionMap())
	// 	return app.App.InitChainer(ctx, req)
	// })

	if err := app.Load(loadLatest); err != nil {
		panic(err)
	}

	return app
}

// CheckTx will check the transaction with the provided checkTxHandler. We override the default
// handler so that we can verify bid transactions before they are inserted into the mempool.
// With the POB CheckTx, we can verify the bid transaction and all of the bundled transactions
// before inserting the bid transaction into the mempool.
func (app *TestApp) CheckTx(req *cometabci.RequestCheckTx) (*cometabci.ResponseCheckTx, error) {
	return app.checkTxHandler(req)
}

// SetCheckTx sets the checkTxHandler for the app.
func (app *TestApp) SetCheckTx(handler checktx.CheckTx) {
	app.checkTxHandler = handler
}

// Name returns the name of the App
func (app *TestApp) Name() string { return app.BaseApp.Name() }

// LegacyAmino returns SimApp's amino codec.
//
// NOTE: This is solely to be used for testing purposes as it may be desirable
// for modules to register their own custom testing types.
func (app *TestApp) LegacyAmino() *codec.LegacyAmino {
	return app.legacyAmino
}

// AppCodec returns SimApp's app codec.
//
// NOTE: This is solely to be used for testing purposes as it may be desirable
// for modules to register their own custom testing types.
func (app *TestApp) AppCodec() codec.Codec {
	return app.appCodec
}

// InterfaceRegistry returns SimApp's InterfaceRegistry
func (app *TestApp) InterfaceRegistry() codectypes.InterfaceRegistry {
	return app.interfaceRegistry
}

// TxConfig returns SimApp's TxConfig
func (app *TestApp) TxConfig() client.TxConfig {
	return app.txConfig
}

// GetKey returns the KVStoreKey for the provided store key.
//
// NOTE: This is solely to be used for testing purposes.
func (app *TestApp) GetKey(storeKey string) *storetypes.KVStoreKey {
	sk := app.UnsafeFindStoreKey(storeKey)
	kvStoreKey, ok := sk.(*storetypes.KVStoreKey)
	if !ok {
		return nil
	}
	return kvStoreKey
}

// GetSubspace returns a param subspace for a given module name.
//
// NOTE: This is solely to be used for testing purposes.
func (app *TestApp) GetSubspace(moduleName string) paramstypes.Subspace {
	subspace, _ := app.ParamsKeeper.GetSubspace(moduleName)
	return subspace
}

// SimulationManager implements the SimulationApp interface
func (app *TestApp) SimulationManager() *module.SimulationManager {
	return nil
}

// RegisterAPIRoutes registers all application module routes with the provided
// API server.
func (app *TestApp) RegisterAPIRoutes(apiSvr *api.Server, apiConfig config.APIConfig) {
	// Register the base app API routes.
	app.App.RegisterAPIRoutes(apiSvr, apiConfig)

	// Register the Block SDK mempool API routes.
	service.RegisterGRPCGatewayRoutes(apiSvr.ClientCtx, apiSvr.GRPCGatewayRouter)

	// register swagger API in app.go so that other applications can override easily
	if err := server.RegisterSwaggerAPI(apiSvr.ClientCtx, apiSvr.Router, apiConfig.Swagger); err != nil {
		panic(err)
	}
}

// RegisterTxService implements the Application.RegisterTxService method.
func (app *TestApp) RegisterTxService(clientCtx client.Context) {
	// Register the base app transaction service.
	app.App.RegisterTxService(clientCtx)

	// Register the Block SDK mempool transaction service.
	mempool, ok := app.App.Mempool().(block.Mempool)
	if !ok {
		panic("mempool is not a block.Mempool")
	}
	service.RegisterMempoolService(app.GRPCQueryRouter(), mempool)
}

// GetMaccPerms returns a copy of the module account permissions
//
// NOTE: This is solely to be used for testing purposes.
func GetMaccPerms() map[string][]string {
	dup := make(map[string][]string)
	for _, perms := range moduleAccPerms {
		dup[perms.Account] = perms.Permissions
	}

	return dup
}

// BlockedAddresses returns all the app's blocked account addresses.
func BlockedAddresses() map[string]bool {
	result := make(map[string]bool)

	if len(blockAccAddrs) > 0 {
		for _, addr := range blockAccAddrs {
			result[addr] = true
		}
	} else {
		for addr := range GetMaccPerms() {
			result[addr] = true
		}
	}

	return result
}
