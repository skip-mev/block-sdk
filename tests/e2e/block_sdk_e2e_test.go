package e2e_test

import (
	"fmt"
	"testing"

	testutil "github.com/cosmos/cosmos-sdk/types/module/testutil"
	"github.com/strangelove-ventures/interchaintest/v7"
	"github.com/strangelove-ventures/interchaintest/v7/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v7/ibc"
	ictestutil "github.com/strangelove-ventures/interchaintest/v7/testutil"
	"github.com/stretchr/testify/suite"

<<<<<<< HEAD:tests/integration/block_sdk_integration_test.go
	"github.com/skip-mev/block-sdk/tests/integration"

=======
	"github.com/skip-mev/block-sdk/lanes/base"
	"github.com/skip-mev/block-sdk/lanes/free"
	"github.com/skip-mev/block-sdk/lanes/mev"
	"github.com/skip-mev/block-sdk/tests/e2e"
>>>>>>> 7e279c5 (chore: rename `integration` to `e2e` (#291)):tests/e2e/block_sdk_e2e_test.go
	auctiontypes "github.com/skip-mev/block-sdk/x/auction/types"
)

var (
	// config params
	numValidators = 4
	numFullNodes  = 0
	denom         = "stake"

	image = ibc.DockerImage{
		Repository: "block-sdk-e2e",
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
		"timeout_commit": "5000ms",
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

func TestE2ETestSuite(t *testing.T) {
	suite.Run(t, e2e.NewE2ETestSuiteFromSpec(spec))
}
