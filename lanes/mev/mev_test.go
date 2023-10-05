package mev_test

import (
	"math/rand"
	"testing"
	"time"

	"github.com/cometbft/cometbft/libs/log"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/suite"

	signer_extraction "github.com/skip-mev/block-sdk/adapters/signer_extraction_adapter"
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

<<<<<<< HEAD
func (suite *MEVTestSuite) SetupTest() {
	// Mempool setup
	suite.encCfg = testutils.CreateTestEncodingConfig()
	suite.config = mev.NewDefaultAuctionFactory(suite.encCfg.TxConfig.TxDecoder(), signer_extraction.NewDefaultAdapter())
	suite.ctx = sdk.NewContext(nil, cmtproto.Header{}, false, log.NewNopLogger())
=======
func (s *MEVTestSuite) SetupTest() {
	// Init encoding config
	s.encCfg = testutils.CreateTestEncodingConfig()
	s.config = mev.NewDefaultAuctionFactory(s.encCfg.TxConfig.TxDecoder(), signer_extraction.NewDefaultAdapter())
	testCtx := testutil.DefaultContextWithDB(s.T(), storetypes.NewKVStoreKey("test"), storetypes.NewTransientStoreKey("transient_test"))
	s.ctx = testCtx.Ctx.WithExecMode(sdk.ExecModePrepareProposal)
	s.ctx = s.ctx.WithBlockHeight(1)
>>>>>>> cbc0483 (chore(verifytx): Updating VerifyTx to Cache between Transactions (#137))

	// Init accounts
	suite.random = rand.New(rand.NewSource(time.Now().Unix()))
	suite.accounts = testutils.RandomAccounts(suite.random, 10)

	suite.nonces = make(map[string]uint64)
	for _, acc := range suite.accounts {
		suite.nonces[acc.Address.String()] = 0
	}
}
