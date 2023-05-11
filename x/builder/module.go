package builder

import (
	"context"
	"encoding/json"
	"fmt"

	"cosmossdk.io/core/appmodule"
	"cosmossdk.io/depinject"
	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/gorilla/mux"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	modulev1 "github.com/skip-mev/pob/api/pob/builder/module/v1"
	"github.com/skip-mev/pob/x/builder/client/cli"
	"github.com/skip-mev/pob/x/builder/keeper"
	"github.com/skip-mev/pob/x/builder/types"
	"github.com/spf13/cobra"
)

var (
	_ module.AppModule      = AppModule{}
	_ module.AppModuleBasic = AppModuleBasic{}
)

// ConsensusVersion defines the current x/builder module consensus version.
const ConsensusVersion = 1

// AppModuleBasic defines the basic application module used by the builder module.
type AppModuleBasic struct {
	cdc codec.Codec
}

// Name returns the builder module's name.
func (AppModuleBasic) Name() string {
	return types.ModuleName
}

// RegisterLegacyAminoCodec registers the builder module's types on the given LegacyAmino codec.
func (AppModuleBasic) RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	types.RegisterLegacyAminoCodec(cdc)
}

// RegisterInterfaces registers the builder module's interface types.
func (AppModuleBasic) RegisterInterfaces(registry cdctypes.InterfaceRegistry) {
	types.RegisterInterfaces(registry)
}

// DefaultGenesis returns default genesis state as raw bytes for the builder module.
func (AppModuleBasic) DefaultGenesis(cdc codec.JSONCodec) json.RawMessage {
	return cdc.MustMarshalJSON(types.DefaultGenesisState())
}

// ValidateGenesis performs genesis state validation for the builder module.
func (AppModuleBasic) ValidateGenesis(cdc codec.JSONCodec, _ client.TxEncodingConfig, bz json.RawMessage) error {
	var genState types.GenesisState
	if err := cdc.UnmarshalJSON(bz, &genState); err != nil {
		return fmt.Errorf("failed to unmarshal %s genesis state: %w", types.ModuleName, err)
	}

	return genState.Validate()
}

// RegisterGRPCGatewayRoutes registers the gRPC Gateway routes for the builder module.
func (AppModuleBasic) RegisterGRPCGatewayRoutes(clientCtx client.Context, mux *runtime.ServeMux) {
	if err := types.RegisterQueryHandlerClient(context.Background(), mux, types.NewQueryClient(clientCtx)); err != nil {
		panic(err)
	}
}

// GetTxCmd returns the root tx command for the builder module.
func (AppModuleBasic) GetTxCmd() *cobra.Command {
	return cli.NewTxCmd()
}

// GetQueryCmd returns the root query command for the builder module.
func (AppModuleBasic) GetQueryCmd() *cobra.Command {
	return cli.GetQueryCmd()
}

type AppModule struct {
	AppModuleBasic

	keeper keeper.Keeper
}

var _ appmodule.AppModule = AppModule{}

// NewAppModule creates a new AppModule object.
func NewAppModule(cdc codec.Codec, keeper keeper.Keeper) AppModule {
	return AppModule{
		AppModuleBasic: AppModuleBasic{cdc: cdc},
		keeper:         keeper,
	}
}

// IsAppModule implements the appmodule.AppModule interface.
func (am AppModule) IsAppModule() {}

// IsOnePerModuleType implements the depinject.OnePerModuleType interface.
func (am AppModule) IsOnePerModuleType() {}

// ConsensusVersion implements AppModule/ConsensusVersion.
func (AppModule) ConsensusVersion() uint64 { return ConsensusVersion }

// RegisterServices registers a the gRPC Query and Msg services for the x/builder
// module.
func (am AppModule) RegisterServices(cfc module.Configurator) {
	types.RegisterMsgServer(cfc.MsgServer(), keeper.NewMsgServerImpl(am.keeper))
	types.RegisterQueryServer(cfc.QueryServer(), keeper.NewQueryServer(am.keeper))
}

func (a AppModuleBasic) RegisterRESTRoutes(_ client.Context, _ *mux.Router) {}

// RegisterInvariants registers the invariants of the module. If an invariant
// deviates from its predicted value, the InvariantRegistry triggers appropriate
// logic (most often the chain will be halted).
func (am AppModule) RegisterInvariants(_ sdk.InvariantRegistry) {}

// InitGenesis performs the module's genesis initialization for the builder
// module. It returns no validator updates.
func (am AppModule) InitGenesis(ctx sdk.Context, cdc codec.JSONCodec, gs json.RawMessage) []abci.ValidatorUpdate {
	var genState types.GenesisState
	cdc.MustUnmarshalJSON(gs, &genState)

	am.keeper.InitGenesis(ctx, genState)
	return []abci.ValidatorUpdate{}
}

// ExportGenesis returns the builder module's exported genesis state as raw
// JSON bytes.
func (am AppModule) ExportGenesis(ctx sdk.Context, cdc codec.JSONCodec) json.RawMessage {
	genState := am.keeper.ExportGenesis(ctx)
	return cdc.MustMarshalJSON(genState)
}

func init() {
	appmodule.Register(
		&modulev1.Module{},
		appmodule.Provide(ProvideModule),
	)
}

type Inputs struct {
	depinject.In

	Config *modulev1.Module
	Cdc    codec.Codec
	Key    *storetypes.KVStoreKey

	AccountKeeper      types.AccountKeeper
	BankKeeper         types.BankKeeper
	DistributionKeeper types.DistributionKeeper
	StakingKeeper      types.StakingKeeper
}

type Outputs struct {
	depinject.Out

	BuilderKeeper keeper.Keeper
	Module        appmodule.AppModule
}

func ProvideModule(in Inputs) Outputs {
	// default to governance authority if not provided
	authority := authtypes.NewModuleAddress(govtypes.ModuleName)
	if in.Config.Authority != "" {
		authority = authtypes.NewModuleAddressOrBech32Address(in.Config.Authority)
	}

	builderKeeper := keeper.NewKeeper(
		in.Cdc,
		in.Key,
		in.AccountKeeper,
		in.BankKeeper,
		in.DistributionKeeper,
		in.StakingKeeper,
		authority.String(),
	)

	m := NewAppModule(in.Cdc, builderKeeper)

	return Outputs{BuilderKeeper: builderKeeper, Module: m}
}
