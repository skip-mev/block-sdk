package blockbuster_test

import (
	"math/rand"
	"testing"
	"time"

	"cosmossdk.io/log"
	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/skip-mev/pob/blockbuster"
	"github.com/skip-mev/pob/blockbuster/lanes/auction"
	"github.com/skip-mev/pob/blockbuster/lanes/base"
	"github.com/skip-mev/pob/blockbuster/lanes/free"
	testutils "github.com/skip-mev/pob/testutils"
	buildertypes "github.com/skip-mev/pob/x/builder/types"
	"github.com/stretchr/testify/suite"
)

type BlockBusterTestSuite struct {
	suite.Suite
	ctx sdk.Context

	// Define basic tx configuration
	encodingConfig testutils.EncodingConfig

	// Define all of the lanes utilized in the test suite
	tobLane  *auction.TOBLane
	baseLane *base.DefaultLane
	freeLane *free.Lane

	lanes   []blockbuster.Lane
	mempool blockbuster.Mempool

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
	key := storetypes.NewKVStoreKey(buildertypes.StoreKey)
	testCtx := testutil.DefaultContextWithDB(suite.T(), key, storetypes.NewTransientStoreKey("transient_test"))
	suite.ctx = testCtx.Ctx.WithBlockHeight(1)

	// Lanes configuration
	//
	// TOB lane set up
	config := blockbuster.BaseLaneConfig{
		Logger:        log.NewNopLogger(),
		TxEncoder:     suite.encodingConfig.TxConfig.TxEncoder(),
		TxDecoder:     suite.encodingConfig.TxConfig.TxDecoder(),
		AnteHandler:   nil,
		MaxBlockSpace: math.LegacyZeroDec(),
	}

	// Top of block lane set up
	suite.tobLane = auction.NewTOBLane(
		config,
		0, // No bound on the number of transactions in the lane
		auction.NewDefaultAuctionFactory(suite.encodingConfig.TxConfig.TxDecoder()),
	)

	// Free lane set up
	suite.freeLane = free.NewFreeLane(
		config,
		free.NewDefaultFreeFactory(suite.encodingConfig.TxConfig.TxDecoder()),
	)

	// Base lane set up
	suite.baseLane = base.NewDefaultLane(
		config,
	)

	// Mempool set up
	suite.lanes = []blockbuster.Lane{suite.tobLane, suite.freeLane, suite.baseLane}
	suite.mempool = blockbuster.NewMempool(suite.lanes...)

	// Accounts set up
	suite.accounts = testutils.RandomAccounts(suite.random, 10)
	suite.nonces = make(map[string]uint64)
	for _, acc := range suite.accounts {
		suite.nonces[acc.Address.String()] = 0
	}
}

func (suite *BlockBusterTestSuite) TestInsert() {
	cases := []struct {
		name               string
		insertDistribution map[string]int
	}{
		{
			"insert 1 tob tx",
			map[string]int{
				suite.tobLane.Name(): 1,
			},
		},
		{
			"insert 10 tob txs",
			map[string]int{
				suite.tobLane.Name(): 10,
			},
		},
		{
			"insert 1 base tx",
			map[string]int{
				suite.baseLane.Name(): 1,
			},
		},
		{
			"insert 10 base txs and 10 tob txs",
			map[string]int{
				suite.baseLane.Name(): 10,
				suite.tobLane.Name():  10,
			},
		},
		{
			"insert 100 base txs and 100 tob txs",
			map[string]int{
				suite.baseLane.Name(): 100,
				suite.tobLane.Name():  100,
			},
		},
		{
			"insert 100 base txs, 100 tob txs, and 100 free txs",
			map[string]int{
				suite.baseLane.Name(): 100,
				suite.tobLane.Name():  100,
				suite.freeLane.Name(): 100,
			},
		},
		{
			"insert 10 free txs",
			map[string]int{
				suite.freeLane.Name(): 10,
			},
		},
		{
			"insert 10 free txs and 10 base txs",
			map[string]int{
				suite.freeLane.Name(): 10,
				suite.baseLane.Name(): 10,
			},
		},
		{
			"insert 10 tob txs and 10 free txs",
			map[string]int{
				suite.tobLane.Name():  10,
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
			suite.fillTOBLane(tc.insertDistribution[suite.tobLane.Name()])

			// Fill the Free lane with numFreeTxs transactions
			suite.fillFreeLane(tc.insertDistribution[suite.freeLane.Name()])

			sum := 0
			for _, v := range tc.insertDistribution {
				sum += v
			}

			// Validate the mempool
			suite.Require().Equal(suite.mempool.CountTx(), sum)

			// Validate the lanes
			suite.Require().Equal(suite.baseLane.CountTx(), tc.insertDistribution[suite.baseLane.Name()])
			suite.Require().Equal(suite.tobLane.CountTx(), tc.insertDistribution[suite.tobLane.Name()])
			suite.Require().Equal(suite.freeLane.CountTx(), tc.insertDistribution[suite.freeLane.Name()])

			// Validate the lane counts
			laneCounts := suite.mempool.GetTxDistribution()

			// Ensure that the lane counts are correct
			suite.Require().Equal(laneCounts[suite.tobLane.Name()], tc.insertDistribution[suite.tobLane.Name()])
			suite.Require().Equal(laneCounts[suite.baseLane.Name()], tc.insertDistribution[suite.baseLane.Name()])
			suite.Require().Equal(laneCounts[suite.freeLane.Name()], tc.insertDistribution[suite.freeLane.Name()])
		})
	}
}

func (suite *BlockBusterTestSuite) TestRemove() {
	cases := []struct {
		name       string
		numTobTxs  int
		numBaseTxs int
	}{
		{
			"insert 1 tob tx",
			1,
			0,
		},
		{
			"insert 10 tob txs",
			10,
			0,
		},
		{
			"insert 1 base tx",
			0,
			1,
		},
		{
			"insert 10 base txs and 10 tob txs",
			10,
			10,
		},
		{
			"insert 100 base txs and 100 tob txs",
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
			tobCount := tc.numTobTxs
			baseCount := tc.numBaseTxs
			for iterator := suite.baseLane.Select(suite.ctx, nil); iterator != nil; {
				tx := iterator.Tx()

				// Remove the transaction from the mempool
				suite.Require().NoError(suite.mempool.Remove(tx))

				// Ensure that the transaction is no longer in the mempool
				contains, err := suite.mempool.Contains(tx)
				suite.Require().NoError(err)
				suite.Require().False(contains)

				// Ensure the number of transactions in the lane is correct
				baseCount--
				suite.Require().Equal(suite.baseLane.CountTx(), baseCount)

				distribution := suite.mempool.GetTxDistribution()
				suite.Require().Equal(distribution[suite.baseLane.Name()], baseCount)

				iterator = suite.baseLane.Select(suite.ctx, nil)
			}

			suite.Require().Equal(0, suite.baseLane.CountTx())
			suite.Require().Equal(tobCount, suite.tobLane.CountTx())

			// Remove all transactions from the lanes
			for iterator := suite.tobLane.Select(suite.ctx, nil); iterator != nil; {
				tx := iterator.Tx()

				// Remove the transaction from the mempool
				suite.Require().NoError(suite.mempool.Remove(tx))

				// Ensure that the transaction is no longer in the mempool
				contains, err := suite.mempool.Contains(tx)
				suite.Require().NoError(err)
				suite.Require().False(contains)

				// Ensure the number of transactions in the lane is correct
				tobCount--
				suite.Require().Equal(suite.tobLane.CountTx(), tobCount)

				distribution := suite.mempool.GetTxDistribution()
				suite.Require().Equal(distribution[suite.tobLane.Name()], tobCount)

				iterator = suite.tobLane.Select(suite.ctx, nil)
			}

			suite.Require().Equal(0, suite.tobLane.CountTx())
			suite.Require().Equal(0, suite.baseLane.CountTx())
			suite.Require().Equal(0, suite.mempool.CountTx())

			// Validate the lane counts
			distribution := suite.mempool.GetTxDistribution()

			// Ensure that the lane counts are correct
			suite.Require().Equal(distribution[suite.tobLane.Name()], 0)
			suite.Require().Equal(distribution[suite.baseLane.Name()], 0)
		})
	}
}

// fillBaseLane fills the base lane with numTxs transactions that are randomly created.
func (suite *BlockBusterTestSuite) fillBaseLane(numTxs int) {
	for i := 0; i < numTxs; i++ {
		// randomly select an account to create the tx
		randomIndex := suite.random.Intn(len(suite.accounts))
		acc := suite.accounts[randomIndex]

		// create a few random msgs and construct the tx
		nonce := suite.nonces[acc.Address.String()]
		randomMsgs := testutils.CreateRandomMsgs(acc.Address, 3)
		tx, err := testutils.CreateTx(suite.encodingConfig.TxConfig, acc, nonce, 1000, randomMsgs)
		suite.Require().NoError(err)

		// insert the tx into the lane and update the account
		suite.nonces[acc.Address.String()]++
		priority := suite.random.Int63n(100) + 1
		suite.Require().NoError(suite.mempool.Insert(suite.ctx.WithPriority(priority), tx))
	}
}

// fillTOBLane fills the TOB lane with numTxs transactions that are randomly created.
func (suite *BlockBusterTestSuite) fillTOBLane(numTxs int) {
	for i := 0; i < numTxs; i++ {
		// randomly select a bidder to create the tx
		randomIndex := suite.random.Intn(len(suite.accounts))
		acc := suite.accounts[randomIndex]

		// create a randomized auction transaction
		nonce := suite.nonces[acc.Address.String()]
		bidAmount := math.NewInt(int64(suite.random.Intn(1000) + 1))
		bid := sdk.NewCoin("stake", bidAmount)
		tx, err := testutils.CreateAuctionTxWithSigners(suite.encodingConfig.TxConfig, acc, bid, nonce, 1000, nil)
		suite.Require().NoError(err)

		// insert the auction tx into the global mempool
		suite.Require().NoError(suite.mempool.Insert(suite.ctx, tx))
		suite.nonces[acc.Address.String()]++
	}
}

// filleFreeLane fills the free lane with numTxs transactions that are randomly created.
func (suite *BlockBusterTestSuite) fillFreeLane(numTxs int) {
	for i := 0; i < numTxs; i++ {
		// randomly select an account to create the tx
		randomIndex := suite.random.Intn(len(suite.accounts))
		acc := suite.accounts[randomIndex]

		// create a few random msgs and construct the tx
		nonce := suite.nonces[acc.Address.String()]
		tx, err := testutils.CreateFreeTx(suite.encodingConfig.TxConfig, acc, nonce, 1000, "val1", sdk.NewCoin("stake", math.NewInt(100)))
		suite.Require().NoError(err)

		// insert the tx into the lane and update the account
		suite.nonces[acc.Address.String()]++
		suite.Require().NoError(suite.mempool.Insert(suite.ctx, tx))
	}
}
