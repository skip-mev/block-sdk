package petri_e2e

import (
	"fmt"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	"github.com/cosmos/cosmos-sdk/crypto/hd"
	"github.com/cosmos/cosmos-sdk/types/module/testutil"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/skip-mev/petri/chain"
	"github.com/skip-mev/petri/node"
	"github.com/skip-mev/petri/provider"
	petritypes "github.com/skip-mev/petri/types"
	"github.com/stretchr/testify/suite"
)

import (
	"testing"

	auctiontypes "github.com/skip-mev/block-sdk/x/auction/types"
)

var (
	// config params
	numValidators = 4
	numFullNodes  = 0
	denom         = "stake"

	image = provider.ImageDefinition{
        Image: "block-sdk-e2e:latest",
		UID:   "0",
		GID:   "0",
	}
	encodingConfig = MakeEncodingConfig()
	noHostMount    = false
	gasAdjustment  = 2.0

	genesisKV = []chain.GenesisKV{
		{
			Key:   "app_state.auction.params.max_bundle_size",
			Value: 3,
		},
		{
			Key:   "consensus_params.block.max_gas",
			Value: "100000000000000000",
		},
		//{
		//	Key: "app_state.blocksdk.lanes",
		//	Value: []blocksdkmoduletypes.Lane{
		//		{
		//			Id:            mev.LaneName,
		//			MaxBlockSpace: math.LegacyMustNewDecFromStr("0.2"),
		//			Order:         0,
		//		},
		//		{
		//			Id:            free.LaneName,
		//			MaxBlockSpace: math.LegacyMustNewDecFromStr("0.2"),
		//			Order:         1,
		//		},
		//		{
		//			Id:            base.LaneName,
		//			MaxBlockSpace: math.LegacyMustNewDecFromStr("0.6"),
		//			Order:         2,
		//		},
		//	},
		//},
	}

	// interchain specification
	spec = petritypes.ChainConfig{
		ChainId:              "block-sdk-0",
		NumValidators:        numValidators,
		NumNodes:             numFullNodes,
		EncodingConfig:       *encodingConfig,
		Image:                image,
		Denom:                denom,
		Decimals:             6,
		BinaryName:           "testappd",
		HomeDir:              "/home/testappd",
		Bech32Prefix:         "cosmos",
		CoinType:             "118",
		HDPath:               "m/44'/118'/0'/0/0",
		GasAdjustment:        gasAdjustment,
		GasPrices:            fmt.Sprintf("0%s", denom),
		ModifyGenesis:        chain.ModifyGenesis(genesisKV),
		UseGenesisSubCommand: true,
		NodeCreator:          node.CreateNode,
	}
)

func MakeEncodingConfig() *testutil.TestEncodingConfig {
	cfg := testutil.MakeTestEncodingConfig()

	cryptocodec.RegisterInterfaces(cfg.InterfaceRegistry)
	authtypes.RegisterInterfaces(cfg.InterfaceRegistry)
	banktypes.RegisterInterfaces(cfg.InterfaceRegistry)

	// register auction types
	auctiontypes.RegisterInterfaces(cfg.InterfaceRegistry)

	return &cfg
}

func TestE2ETestSuite(t *testing.T) {
	bip44Path, err := hd.NewParamsFromPath(spec.HDPath)
	if err != nil {
		panic(err)
	}

	spec.WalletConfig = petritypes.WalletConfig{
		DerivationFn:     hd.Secp256k1.Derive(),
		GenerationFn:     hd.Secp256k1.Generate(),
		SigningAlgorithm: string(hd.Secp256k1Type),
		Bech32Prefix:     spec.Bech32Prefix,
		HDPath:           bip44Path,
	}
	suite.Run(t, NewE2ETestSuiteFromSpec(spec))
}
