package integration_test

import (
	"fmt"
	"testing"

	testutil "github.com/cosmos/cosmos-sdk/types/module/testutil"
	"github.com/skip-mev/pob/tests/integration"
	buildertypes "github.com/skip-mev/pob/x/builder/types"
	"github.com/strangelove-ventures/interchaintest/v7"
	"github.com/strangelove-ventures/interchaintest/v7/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v7/ibc"
	"github.com/stretchr/testify/suite"
)

var (
	// config params
	numValidators = 4
	numFullNodes  = 0
	denom         = "stake"

	image = ibc.DockerImage{
		Repository: "pob-integration",
		Version:    "latest",
		UidGid:     "1000:1000",
	}
	encodingConfig = MakeEncodingConfig()
	noHostMount    = false
	gasAdjustment  = float64(2.0)

	genesisKV = []cosmos.GenesisKV{
		{
			Key:   "app_state.builder.params.max_bundle_size",
			Value: 3,
		},
	}

	// interchain specification
	spec = &interchaintest.ChainSpec{
		ChainName:     "pob",
		Name:          "pob",
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
			Name:                   "pob",
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
		},
	}
)

func MakeEncodingConfig() *testutil.TestEncodingConfig {
	cfg := cosmos.DefaultEncoding()

	// register builder types
	buildertypes.RegisterInterfaces(cfg.InterfaceRegistry)

	return &cfg
}

func TestIntegrationTestSuite(t *testing.T) {
	suite.Run(t, integration.NewPOBIntegrationTestSuiteFromSpec(spec))
}
