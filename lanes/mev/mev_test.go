package mev_test

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/rand"
	"os"
	"testing"
	"time"

	"cosmossdk.io/math"
	"github.com/cometbft/cometbft/libs/log"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/suite"

	signer_extraction "github.com/skip-mev/block-sdk/adapters/signer_extraction_adapter"
	"github.com/skip-mev/block-sdk/block/base"
	"github.com/skip-mev/block-sdk/block/utils"
	"github.com/skip-mev/block-sdk/lanes/mev"
	testutils "github.com/skip-mev/block-sdk/testutils"
)

type MEVTestSuite struct {
	suite.Suite

	encCfg        testutils.EncodingConfig
	config        mev.Factory
	ctx           sdk.Context
	accounts      []testutils.Account
	gasTokenDenom string
}

func TestMEVTestSuite(t *testing.T) {
	suite.Run(t, new(MEVTestSuite))
}

func (s *MEVTestSuite) SetupTest() {
	// Init encoding config
	s.encCfg = testutils.CreateTestEncodingConfig()
	s.config = mev.NewDefaultAuctionFactory(s.encCfg.TxConfig.TxDecoder(), signer_extraction.NewDefaultAdapter())
	testCtx := testutil.DefaultContextWithDB(s.T(), storetypes.NewKVStoreKey("test"), storetypes.NewTransientStoreKey("transient_test"))
	s.ctx = s.ctx.WithBlockHeight(1)
	s.ctx = testCtx.Ctx.WithIsCheckTx(true)

	// Init accounts
	random := rand.New(rand.NewSource(time.Now().Unix()))
	s.accounts = testutils.RandomAccounts(random, 10)
	s.gasTokenDenom = "stake"
}

func (s *MEVTestSuite) initLane(
	maxBlockSpace math.LegacyDec,
	expectedExecution map[sdk.Tx]bool,
) *mev.MEVLane {
	config := base.NewLaneConfig(
		log.NewTMLogger(os.Stdout),
		s.encCfg.TxConfig.TxEncoder(),
		s.encCfg.TxConfig.TxDecoder(),
		s.setUpAnteHandler(expectedExecution),
		signer_extraction.NewDefaultAdapter(),
		maxBlockSpace,
	)

	factory := mev.NewDefaultAuctionFactory(s.encCfg.TxConfig.TxDecoder(), signer_extraction.NewDefaultAdapter())
	return mev.NewMEVLane(config, factory, factory.MatchHandler())
}

func (s *MEVTestSuite) setUpAnteHandler(expectedExecution map[sdk.Tx]bool) sdk.AnteHandler {
	txCache := make(map[string]bool)
	for tx, pass := range expectedExecution {
		bz, err := s.encCfg.TxConfig.TxEncoder()(tx)
		s.Require().NoError(err)

		hash := sha256.Sum256(bz)
		hashStr := hex.EncodeToString(hash[:])
		txCache[hashStr] = pass
	}

	anteHandler := func(ctx sdk.Context, tx sdk.Tx, simulate bool) (newCtx sdk.Context, err error) {
		bz, err := s.encCfg.TxConfig.TxEncoder()(tx)
		s.Require().NoError(err)

		hash := sha256.Sum256(bz)
		hashStr := hex.EncodeToString(hash[:])

		pass, found := txCache[hashStr]
		if !found {
			return ctx, fmt.Errorf("tx not found")
		}

		if pass {
			return ctx, nil
		}

		return ctx, fmt.Errorf("tx failed")
	}

	return anteHandler
}

func (s *MEVTestSuite) getTxSize(tx sdk.Tx) int64 {
	txBz, err := s.encCfg.TxConfig.TxEncoder()(tx)
	s.Require().NoError(err)

	return int64(len(txBz))
}

func (s *MEVTestSuite) compare(first, second []sdk.Tx) {
	firstBytes, err := utils.GetEncodedTxs(s.encCfg.TxConfig.TxEncoder(), first)
	s.Require().NoError(err)

	secondBytes, err := utils.GetEncodedTxs(s.encCfg.TxConfig.TxEncoder(), second)
	s.Require().NoError(err)

	s.Require().Equal(firstBytes, secondBytes)
}
