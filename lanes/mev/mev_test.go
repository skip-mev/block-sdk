package mev_test

import (
	"math/rand"
	"testing"
	"time"

	"cosmossdk.io/log"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/suite"

	"github.com/skip-mev/block-sdk/lanes/mev"
	testutils "github.com/skip-mev/block-sdk/testutils"
)

type MEVTestSuite struct {
	suite.Suite

	encCfg   testutils.EncodingConfig
	config   mev.Factory
	ctx      sdk.Context
	random   *rand.Rand
	accounts []testutils.Account
	nonces   map[string]uint64
}

func TestMempoolTestSuite(t *testing.T) {
	suite.Run(t, new(MEVTestSuite))
}

func (suite *MEVTestSuite) SetupTest() {
	// Mempool setup
	suite.encCfg = testutils.CreateTestEncodingConfig()
	suite.config = mev.NewDefaultAuctionFactory(suite.encCfg.TxConfig.TxDecoder())
	suite.ctx = sdk.NewContext(nil, cmtproto.Header{}, false, log.NewTestLogger(suite.T()))

	// Init accounts
	suite.random = rand.New(rand.NewSource(time.Now().Unix()))
	suite.accounts = testutils.RandomAccounts(suite.random, 10)

	suite.nonces = make(map[string]uint64)
	for _, acc := range suite.accounts {
		suite.nonces[acc.Address.String()] = 0
	}
}
