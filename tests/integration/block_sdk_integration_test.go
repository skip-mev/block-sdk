package integration_test

import (
	"fmt"
	"testing"

	testutil "github.com/cosmos/cosmos-sdk/types/module/testutil"
	"github.com/strangelove-ventures/interchaintest/v7"
	"github.com/strangelove-ventures/interchaintest/v7/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v7/ibc"
	ictestutil "github.com/strangelove-ventures/interchaintest/v7/testutil"
	"github.com/stretchr/testify/suite"

	"github.com/skip-mev/block-sdk/tests/integration"
	auctiontypes "github.com/skip-mev/block-sdk/x/auction/types"
)

var (
	// config params
	numValidators = 1
	numFullNodes  = 0
	denom         = "stake"

	image = ibc.DockerImage{
		Repository: "block-sdk-integration",
		Version:    "latest",
		UidGid:     "1000:1000",
	}
	encodingConfig = MakeEncodingConfig()
	noHostMount    = false
	gasAdjustment  = float64(2.0)

	genesisKV = []cosmos.GenesisKV{
		{
			Key:   "app_state.auction.params.max_bundle_size",
			Value: 3,
		},
	}

	consensusParams = ictestutil.Toml{
		"timeout_commit": "3500ms",
	}

	// interchain specification
	spec = &interchaintest.ChainSpec{
		ChainName:     "block-sdk",
		Name:          "block-sdk",
		NumValidators: &numValidators,
		NumFullNodes:  &numFullNodes,
		Version:       "latest",
		NoHostMount:   &noHostMount,
		GasAdjustment: &gasAdjustment,
		ChainConfig: ibc.ChainConfig{
			EncodingConfig: encodingConfig,
			Images: []ibc.DockerImage{
				image,
			},
			Type:                   "cosmos",
			Name:                   "block-sdk",
			Denom:                  denom,
			ChainID:                "chain-id-0",
			Bin:                    "testappd",
			Bech32Prefix:           "cosmos",
			CoinType:               "118",
			GasAdjustment:          gasAdjustment,
			GasPrices:              fmt.Sprintf("0%s", denom),
			TrustingPeriod:         "48h",
			NoHostMount:            noHostMount,
			UsingNewGenesisCommand: true,
			ModifyGenesis:          cosmos.ModifyGenesis(genesisKV),
			ConfigFileOverrides:    map[string]any{"config/config.toml": ictestutil.Toml{"consensus": consensusParams}},
		},
	}
)

func MakeEncodingConfig() *testutil.TestEncodingConfig {
	cfg := cosmos.DefaultEncoding()

	// register auction types
	auctiontypes.RegisterInterfaces(cfg.InterfaceRegistry)

	return &cfg
}

func TestIntegrationTestSuite(t *testing.T) {
	suite.Run(t, integration.NewIntegrationTestSuiteFromSpec(spec))
}
