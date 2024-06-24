package block_test

import (
	"math/rand"
	"testing"
	"time"

	"cosmossdk.io/log"
	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/suite"

	signer_extraction "github.com/skip-mev/block-sdk/v2/adapters/signer_extraction_adapter"
	"github.com/skip-mev/block-sdk/v2/block"
	"github.com/skip-mev/block-sdk/v2/block/base"
	defaultlane "github.com/skip-mev/block-sdk/v2/lanes/base"
	"github.com/skip-mev/block-sdk/v2/lanes/free"
	"github.com/skip-mev/block-sdk/v2/lanes/mev"
	testutils "github.com/skip-mev/block-sdk/v2/testutils"
	auctiontypes "github.com/skip-mev/block-sdk/v2/x/auction/types"
)

type BlockBusterTestSuite struct {
	suite.Suite
	ctx sdk.Context

	// Define basic tx configuration
	encodingConfig testutils.EncodingConfig

	// Define all of the lanes utilized in the test suite
	mevLane       *mev.MEVLane
	baseLane      *base.BaseLane
	freeLane      *base.BaseLane
	gasTokenDenom string

	lanes   []block.Lane
	mempool *block.LanedMempool

	// account set up
	accounts []testutils.Account
	random   *rand.Rand
	nonces   map[string]uint64
}

func TestBlockBusterTestSuite(t *testing.T) {
	suite.Run(t, new(BlockBusterTestSuite))
}

func (suite *BlockBusterTestSuite) SetupTest() {
	// General config for transactions and randomness for the test suite
	suite.encodingConfig = testutils.CreateTestEncodingConfig()
	suite.random = rand.New(rand.NewSource(time.Now().Unix()))
	key := storetypes.NewKVStoreKey(auctiontypes.StoreKey)
	testCtx := testutil.DefaultContextWithDB(suite.T(), key, storetypes.NewTransientStoreKey("transient_test"))
	suite.ctx = testCtx.Ctx.WithBlockHeight(1)

	// Lanes configuration
	//
	// TOB lane set up
	suite.gasTokenDenom = "stake"
	mevConfig := base.LaneConfig{
		Logger:          log.NewNopLogger(),
		TxEncoder:       suite.encodingConfig.TxConfig.TxEncoder(),
		TxDecoder:       suite.encodingConfig.TxConfig.TxDecoder(),
		SignerExtractor: signer_extraction.NewDefaultAdapter(),
		AnteHandler:     nil,
		MaxBlockSpace:   math.LegacyMustNewDecFromStr("0.3"),
	}
	factory := mev.NewDefaultAuctionFactory(suite.encodingConfig.TxConfig.TxDecoder(), signer_extraction.NewDefaultAdapter())
	suite.mevLane = mev.NewMEVLane(
		mevConfig,
		factory,
		factory.MatchHandler(),
	)

	// Free lane set up
	freeConfig := base.LaneConfig{
		Logger:          log.NewNopLogger(),
		TxEncoder:       suite.encodingConfig.TxConfig.TxEncoder(),
		TxDecoder:       suite.encodingConfig.TxConfig.TxDecoder(),
		SignerExtractor: signer_extraction.NewDefaultAdapter(),
		AnteHandler:     nil,
		MaxBlockSpace:   math.LegacyMustNewDecFromStr("0.3"),
	}
	suite.freeLane = free.NewFreeLane(
		freeConfig,
		base.DefaultTxPriority(),
		free.DefaultMatchHandler(),
	)

	// Base lane set up
	baseConfig := base.LaneConfig{
		Logger:          log.NewNopLogger(),
		TxEncoder:       suite.encodingConfig.TxConfig.TxEncoder(),
		TxDecoder:       suite.encodingConfig.TxConfig.TxDecoder(),
		SignerExtractor: signer_extraction.NewDefaultAdapter(),
		AnteHandler:     nil,
		MaxBlockSpace:   math.LegacyZeroDec(),
	}
	suite.baseLane = defaultlane.NewDefaultLane(
		baseConfig,
		base.DefaultMatchHandler(),
	)

	// Mempool set up
	suite.lanes = []block.Lane{suite.mevLane, suite.freeLane, suite.baseLane}

	var err error
	suite.mempool, err = block.NewLanedMempool(
		log.NewNopLogger(),
		suite.lanes,
	)
	suite.Require().NoError(err)

	// Accounts set up
	suite.accounts = testutils.RandomAccounts(suite.random, 10)
	suite.nonces = make(map[string]uint64)
	for _, acc := range suite.accounts {
		suite.nonces[acc.Address.String()] = 0
	}
}

func (suite *BlockBusterTestSuite) TestNewMempool() {
	defaultConfig := base.LaneConfig{
		Logger:          log.NewNopLogger(),
		TxEncoder:       suite.encodingConfig.TxConfig.TxEncoder(),
		TxDecoder:       suite.encodingConfig.TxConfig.TxDecoder(),
		SignerExtractor: signer_extraction.NewDefaultAdapter(),
		AnteHandler:     nil,
		MaxBlockSpace:   math.LegacyZeroDec(),
	}

	baseConfig := base.LaneConfig{
		Logger:          log.NewNopLogger(),
		TxEncoder:       suite.encodingConfig.TxConfig.TxEncoder(),
		TxDecoder:       suite.encodingConfig.TxConfig.TxDecoder(),
		SignerExtractor: signer_extraction.NewDefaultAdapter(),
		AnteHandler:     nil,
		MaxBlockSpace:   math.LegacyMustNewDecFromStr("0.3"),
	}

	defaultLane := defaultlane.NewDefaultLane(defaultConfig, base.DefaultMatchHandler())
	factory := mev.NewDefaultAuctionFactory(suite.encodingConfig.TxConfig.TxDecoder(), signer_extraction.NewDefaultAdapter())
	mevLane := mev.NewMEVLane(
		baseConfig,
		factory,
		factory.MatchHandler(),
	)
	freeLane := free.NewFreeLane(
		baseConfig,
		base.DefaultTxPriority(),
		free.DefaultMatchHandler(),
	)

	invalidFreeLane := free.NewFreeLane(
		defaultConfig,
		base.DefaultTxPriority(),
		free.DefaultMatchHandler(),
	)

	suite.Run("works with a single lane", func() {
		lanes := []block.Lane{defaultLane}

		_, err := block.NewLanedMempool(
			log.NewNopLogger(),
			lanes,
		)
		suite.Require().NoError(err)
	})

	suite.Run("works mev and default lane", func() {
		lanes := []block.Lane{mevLane, defaultLane}

		_, err := block.NewLanedMempool(
			log.NewNopLogger(),
			lanes,
		)
		suite.Require().NoError(err)
	})

	suite.Run("works mev and default lane in reverse order", func() {
		lanes := []block.Lane{mevLane, defaultLane}

		_, err := block.NewLanedMempool(
			log.NewNopLogger(),
			lanes,
		)
		suite.Require().NoError(err)
	})

	suite.Run("works with mev, free, and default lane", func() {
		lanes := []block.Lane{mevLane, freeLane, defaultLane}

		_, err := block.NewLanedMempool(
			log.NewNopLogger(),
			lanes,
		)
		suite.Require().NoError(err)
	})

	suite.Run("works with mev, default, free lane", func() {
		lanes := []block.Lane{mevLane, defaultLane, freeLane}

		_, err := block.NewLanedMempool(
			log.NewNopLogger(),
			lanes,
		)
		suite.Require().NoError(err)
	})

	suite.Run("works with free, mev, and default lane", func() {
		lanes := []block.Lane{freeLane, mevLane, defaultLane}

		_, err := block.NewLanedMempool(
			log.NewNopLogger(),
			lanes,
		)
		suite.Require().NoError(err)
	})

	suite.Run("works with default, free, mev lanes", func() {
		lanes := []block.Lane{defaultLane, freeLane, mevLane}

		_, err := block.NewLanedMempool(
			log.NewNopLogger(),
			lanes,
		)
		suite.Require().NoError(err)
	})

	suite.Run("default lane not included - invalid total space", func() {
		lanes := []block.Lane{mevLane, freeLane}

		_, err := block.NewLanedMempool(
			log.NewNopLogger(),
			lanes,
		)
		suite.Require().Error(err)
	})

	suite.Run("two lanes with unlimited blockspace", func() {
		lanes := []block.Lane{defaultLane, invalidFreeLane}

		_, err := block.NewLanedMempool(
			log.NewNopLogger(),
			lanes,
		)
		suite.Require().Error(err)
	})

	suite.Run("duplicate lanes", func() {
		lanes := []block.Lane{mevLane, defaultLane, mevLane}

		_, err := block.NewLanedMempool(
			log.NewNopLogger(),
			lanes,
		)
		suite.Require().Error(err)
	})
}

func (suite *BlockBusterTestSuite) TestInsert() {
	cases := []struct {
		name               string
		insertDistribution map[string]uint64
	}{
		{
			"insert 1 mev tx",
			map[string]uint64{
				suite.mevLane.Name(): 1,
			},
		},
		{
			"insert 10 mev txs",
			map[string]uint64{
				suite.mevLane.Name(): 10,
			},
		},
		{
			"insert 1 base tx",
			map[string]uint64{
				suite.baseLane.Name(): 1,
			},
		},
		{
			"insert 10 base txs and 10 mev txs",
			map[string]uint64{
				suite.baseLane.Name(): 10,
				suite.mevLane.Name():  10,
			},
		},
		{
			"insert 100 base txs and 100 mev txs",
			map[string]uint64{
				suite.baseLane.Name(): 100,
				suite.mevLane.Name():  100,
			},
		},
		{
			"insert 100 base txs, 100 mev txs, and 100 free txs",
			map[string]uint64{
				suite.baseLane.Name(): 100,
				suite.mevLane.Name():  100,
				suite.freeLane.Name(): 100,
			},
		},
		{
			"insert 10 free txs",
			map[string]uint64{
				suite.freeLane.Name(): 10,
			},
		},
		{
			"insert 10 free txs and 10 base txs",
			map[string]uint64{
				suite.freeLane.Name(): 10,
				suite.baseLane.Name(): 10,
			},
		},
		{
			"insert 10 mev txs and 10 free txs",
			map[string]uint64{
				suite.mevLane.Name():  10,
				suite.freeLane.Name(): 10,
			},
		},
	}

	for _, tc := range cases {
		suite.Run(tc.name, func() {
			suite.SetupTest() // reset

			// Fill the base lane with numBaseTxs transactions
			suite.fillBaseLane(tc.insertDistribution[suite.baseLane.Name()])

			// Fill the TOB lane with numTobTxs transactions
			suite.fillTOBLane(tc.insertDistribution[suite.mevLane.Name()])

			// Fill the Free lane with numFreeTxs transactions
			suite.fillFreeLane(tc.insertDistribution[suite.freeLane.Name()])

			sum := uint64(0)
			for _, v := range tc.insertDistribution {
				sum += v
			}

			// Validate the mempool
			suite.Require().Equal(int(sum), suite.mempool.CountTx())

			// Validate the lanes
			suite.Require().Equal(tc.insertDistribution[suite.mevLane.Name()], uint64(suite.mevLane.CountTx()))
			suite.Require().Equal(tc.insertDistribution[suite.baseLane.Name()], uint64(suite.baseLane.CountTx()))
			suite.Require().Equal(tc.insertDistribution[suite.freeLane.Name()], uint64(suite.freeLane.CountTx()))

			// Validate the lane counts
			laneCounts := suite.mempool.GetTxDistribution()

			// Ensure that the lane counts are correct
			suite.Require().Equal(tc.insertDistribution[suite.mevLane.Name()], laneCounts[suite.mevLane.Name()])
			suite.Require().Equal(tc.insertDistribution[suite.baseLane.Name()], laneCounts[suite.baseLane.Name()])
			suite.Require().Equal(tc.insertDistribution[suite.freeLane.Name()], laneCounts[suite.freeLane.Name()])
		})
	}
}

func (suite *BlockBusterTestSuite) TestRemove() {
	cases := []struct {
		name       string
		numTobTxs  uint64
		numBaseTxs uint64
	}{
		{
			"insert 1 mev tx",
			1,
			0,
		},
		{
			"insert 10 mev txs",
			10,
			0,
		},
		{
			"insert 1 base tx",
			0,
			1,
		},
		{
			"insert 10 base txs and 10 mev txs",
			10,
			10,
		},
		{
			"insert 100 base txs and 100 mev txs",
			100,
			100,
		},
	}

	for _, tc := range cases {
		suite.Run(tc.name, func() {
			suite.SetupTest() // reset

			// Fill the base lane with numBaseTxs transactions
			suite.fillBaseLane(tc.numBaseTxs)

			// Fill the TOB lane with numTobTxs transactions
			suite.fillTOBLane(tc.numTobTxs)

			// Remove all transactions from the lanes
			mevCount := tc.numTobTxs
			baseCount := tc.numBaseTxs
			for iterator := suite.baseLane.Select(suite.ctx, nil); iterator != nil; {
				tx := iterator.Tx()

				// Remove the transaction from the mempool
				suite.Require().NoError(suite.mempool.Remove(tx))

				// Ensure that the transaction is no longer in the mempool
				suite.Require().Equal(false, suite.mempool.Contains(tx))

				// Ensure the number of transactions in the lane is correct
				baseCount--
				suite.Require().Equal(suite.baseLane.CountTx(), int(baseCount))

				distribution := suite.mempool.GetTxDistribution()
				suite.Require().Equal(distribution[suite.baseLane.Name()], baseCount)

				iterator = suite.baseLane.Select(suite.ctx, nil)
			}

			suite.Require().Equal(0, suite.baseLane.CountTx())
			suite.Require().Equal(int(mevCount), suite.mevLane.CountTx())

			// Remove all transactions from the lanes
			for iterator := suite.mevLane.Select(suite.ctx, nil); iterator != nil; {
				tx := iterator.Tx()

				// Remove the transaction from the mempool
				suite.Require().NoError(suite.mempool.Remove(tx))

				// Ensure that the transaction is no longer in the mempool
				suite.Require().Equal(false, suite.mempool.Contains(tx))

				// Ensure the number of transactions in the lane is correct
				mevCount--
				suite.Require().Equal(suite.mevLane.CountTx(), int(mevCount))

				distribution := suite.mempool.GetTxDistribution()
				suite.Require().Equal(distribution[suite.mevLane.Name()], mevCount)

				iterator = suite.mevLane.Select(suite.ctx, nil)
			}

			suite.Require().Equal(0, suite.mevLane.CountTx())
			suite.Require().Equal(0, suite.baseLane.CountTx())
			suite.Require().Equal(0, suite.mempool.CountTx())

			// Validate the lane counts
			distribution := suite.mempool.GetTxDistribution()

			// Ensure that the lane counts are correct
			suite.Require().Equal(distribution[suite.mevLane.Name()], uint64(0))
			suite.Require().Equal(distribution[suite.baseLane.Name()], uint64(0))
		})
	}
}

// fillBaseLane fills the base lane with numTxs transactions that are randomly created.
func (suite *BlockBusterTestSuite) fillBaseLane(numTxs uint64) {
	for i := uint64(0); i < numTxs; i++ {
		// randomly select an account to create the tx
		randomIndex := suite.random.Intn(len(suite.accounts))
		acc := suite.accounts[randomIndex]

		// create a few random msgs and construct the tx
		nonce := suite.nonces[acc.Address.String()]
		randomMsgs := testutils.CreateRandomMsgs(acc.Address, 3)
		priority := suite.random.Int63n(100) + 1
		tx, err := testutils.CreateTx(suite.encodingConfig.TxConfig, acc, nonce, 1000, randomMsgs, sdk.NewCoin(suite.gasTokenDenom, math.NewInt(priority)))
		suite.Require().NoError(err)

		// insert the tx into the lane and update the account
		suite.nonces[acc.Address.String()]++
		suite.Require().NoError(suite.mempool.Insert(suite.ctx.WithPriority(priority), tx))
	}
}

// fillTOBLane fills the TOB lane with numTxs transactions that are randomly created.
func (suite *BlockBusterTestSuite) fillTOBLane(numTxs uint64) {
	for i := uint64(0); i < numTxs; i++ {
		// randomly select a bidder to create the tx
		randomIndex := suite.random.Intn(len(suite.accounts))
		acc := suite.accounts[randomIndex]

		// create a randomized auction transaction
		nonce := suite.nonces[acc.Address.String()]
		bidAmount := math.NewInt(int64(suite.random.Intn(1000) + 1))
		bid := sdk.NewCoin(suite.gasTokenDenom, bidAmount)
		tx, err := testutils.CreateAuctionTxWithSigners(suite.encodingConfig.TxConfig, acc, bid, nonce, 1000, nil)
		suite.Require().NoError(err)

		// insert the auction tx into the global mempool
		suite.Require().NoError(suite.mempool.Insert(suite.ctx, tx))
		suite.nonces[acc.Address.String()]++
	}
}

// filleFreeLane fills the free lane with numTxs transactions that are randomly created.
func (suite *BlockBusterTestSuite) fillFreeLane(numTxs uint64) {
	for i := uint64(0); i < numTxs; i++ {
		// randomly select an account to create the tx
		randomIndex := suite.random.Intn(len(suite.accounts))
		acc := suite.accounts[randomIndex]

		// create a few random msgs and construct the tx
		nonce := suite.nonces[acc.Address.String()]
		tx, err := testutils.CreateFreeTx(suite.encodingConfig.TxConfig, acc, nonce, 1000, "val1", sdk.NewCoin(suite.gasTokenDenom, math.NewInt(100)), sdk.NewCoin(suite.gasTokenDenom, math.NewInt(100)))
		suite.Require().NoError(err)

		// insert the tx into the lane and update the account
		suite.nonces[acc.Address.String()]++
		suite.Require().NoError(suite.mempool.Insert(suite.ctx, tx))
	}
}
