package e2e

import (
	"fmt"
	"os"

	dbm "github.com/cometbft/cometbft-db"
	"github.com/cometbft/cometbft/libs/log"
	cometrand "github.com/cometbft/cometbft/libs/rand"
	"github.com/cosmos/cosmos-sdk/codec"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	"github.com/skip-mev/pob/tests/app"
	"github.com/skip-mev/pob/tests/app/params"
)

const (
	keyringPassphrase = "testpassphrase"
	keyringAppName    = "testnet"
)

var (
	encodingConfig params.EncodingConfig
	cdc            codec.Codec
)

func init() {
	testApp := app.New(log.NewNopLogger(), dbm.NewMemDB(), nil, true, simtestutil.NewAppOptionsWithFlagHome(app.DefaultNodeHome))
	encodingConfig = params.EncodingConfig{
		InterfaceRegistry: testApp.InterfaceRegistry(),
		Codec:             testApp.AppCodec(),
		TxConfig:          testApp.TxConfig(),
		Amino:             testApp.LegacyAmino(),
	}
	cdc = encodingConfig.Codec
}

type chain struct {
	dataDir    string
	id         string
	validators []*validator
}

func newChain() (*chain, error) {
	tmpDir, err := os.MkdirTemp("", "pob-e2e-testnet-")
	if err != nil {
		return nil, err
	}

	return &chain{
		id:      "chain-" + cometrand.NewRand().Str(6),
		dataDir: tmpDir,
	}, nil
}

func (c *chain) configDir() string {
	return fmt.Sprintf("%s/%s", c.dataDir, c.id)
}

func (c *chain) createAndInitValidators(count int) error {
	for i := 0; i < count; i++ {
		node := c.createValidator(i)

		// generate genesis files
		if err := node.init(); err != nil {
			return err
		}

		c.validators = append(c.validators, node)

		// create keys
		if err := node.createKey("val"); err != nil {
			return err
		}
		if err := node.createNodeKey(); err != nil {
			return err
		}
		if err := node.createConsensusKey(); err != nil {
			return err
		}
	}

	return nil
}

func (c *chain) createValidator(index int) *validator {
	return &validator{
		chain:   c,
		index:   index,
		moniker: "testapp",
	}
}
