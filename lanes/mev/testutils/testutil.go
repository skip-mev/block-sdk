package testutils

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/rand"
	"time"

	"cosmossdk.io/log"
	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/suite"

	signer_extraction "github.com/skip-mev/block-sdk/v2/adapters/signer_extraction_adapter"
	"github.com/skip-mev/block-sdk/v2/block/base"
	"github.com/skip-mev/block-sdk/v2/lanes/mev"
	testutils "github.com/skip-mev/block-sdk/v2/testutils"
)

type MEVLaneTestSuiteBase struct {
	suite.Suite

	EncCfg        testutils.EncodingConfig
	Config        mev.Factory
	Ctx           sdk.Context
	Accounts      []testutils.Account
	GasTokenDenom string
}

func (s *MEVLaneTestSuiteBase) SetupTest() {
	// Init encoding config
	s.EncCfg = testutils.CreateTestEncodingConfig()
	s.Config = mev.NewDefaultAuctionFactory(s.EncCfg.TxConfig.TxDecoder(), signer_extraction.NewDefaultAdapter())
	testCtx := testutil.DefaultContextWithDB(s.T(), storetypes.NewKVStoreKey("test"), storetypes.NewTransientStoreKey("transient_test"))
	s.Ctx = testCtx.Ctx.WithExecMode(sdk.ExecModePrepareProposal)
	s.Ctx = s.Ctx.WithBlockHeight(1)

	// Init accounts
	random := rand.New(rand.NewSource(time.Now().Unix()))
	s.Accounts = testutils.RandomAccounts(random, 10)
	s.GasTokenDenom = "stake"
}

func (s *MEVLaneTestSuiteBase) InitLane(
	maxBlockSpace math.LegacyDec,
	expectedExecution map[sdk.Tx]bool,
) *mev.MEVLane {
	config := base.NewLaneConfig(
		log.NewNopLogger(),
		s.EncCfg.TxConfig.TxEncoder(),
		s.EncCfg.TxConfig.TxDecoder(),
		s.SetUpAnteHandler(expectedExecution),
		signer_extraction.NewDefaultAdapter(),
		maxBlockSpace,
	)

	factory := mev.NewDefaultAuctionFactory(s.EncCfg.TxConfig.TxDecoder(), signer_extraction.NewDefaultAdapter())
	return mev.NewMEVLane(config, factory, factory.MatchHandler())
}

func (s *MEVLaneTestSuiteBase) SetUpAnteHandler(expectedExecution map[sdk.Tx]bool) sdk.AnteHandler {
	txCache := make(map[string]bool)
	for tx, pass := range expectedExecution {
		bz, err := s.EncCfg.TxConfig.TxEncoder()(tx)
		s.Require().NoError(err)

		hash := sha256.Sum256(bz)
		hashStr := hex.EncodeToString(hash[:])
		txCache[hashStr] = pass
	}

	anteHandler := func(ctx sdk.Context, tx sdk.Tx, simulate bool) (newCtx sdk.Context, err error) {
		bz, err := s.EncCfg.TxConfig.TxEncoder()(tx)
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
