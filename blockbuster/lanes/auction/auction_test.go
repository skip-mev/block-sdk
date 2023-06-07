package auction_test

import (
	"math/rand"
	"testing"
	"time"

	"github.com/cometbft/cometbft/libs/log"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/skip-mev/pob/blockbuster/lanes/auction"
	testutils "github.com/skip-mev/pob/testutils"
	"github.com/stretchr/testify/suite"
)

type IntegrationTestSuite struct {
	suite.Suite

	encCfg   testutils.EncodingConfig
	config   auction.Factory
	mempool  auction.Mempool
	ctx      sdk.Context
	random   *rand.Rand
	accounts []testutils.Account
	nonces   map[string]uint64
}

func TestMempoolTestSuite(t *testing.T) {
	suite.Run(t, new(IntegrationTestSuite))
}

func (suite *IntegrationTestSuite) SetupTest() {
	// Mempool setup
	suite.encCfg = testutils.CreateTestEncodingConfig()
	suite.config = auction.NewDefaultAuctionFactory(suite.encCfg.TxConfig.TxDecoder())
	suite.mempool = auction.NewMempool(suite.encCfg.TxConfig.TxEncoder(), 0, suite.config)
	suite.ctx = sdk.NewContext(nil, cmtproto.Header{}, false, log.NewNopLogger())

	// Init accounts
	suite.random = rand.New(rand.NewSource(time.Now().Unix()))
	suite.accounts = testutils.RandomAccounts(suite.random, 10)

	suite.nonces = make(map[string]uint64)
	for _, acc := range suite.accounts {
		suite.nonces[acc.Address.String()] = 0
	}
}
