package abci_test

import (
	"math/rand"
	"os"
	"testing"

	"cosmossdk.io/math"
	cometabci "github.com/cometbft/cometbft/abci/types"
	"github.com/cometbft/cometbft/libs/log"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/skip-mev/block-sdk/abci"
	"github.com/skip-mev/block-sdk/block"
	"github.com/skip-mev/block-sdk/block/proposals"
	testutils "github.com/skip-mev/block-sdk/testutils"
	"github.com/stretchr/testify/suite"
)

type ProposalsTestSuite struct {
	suite.Suite
	ctx sdk.Context
	key *storetypes.KVStoreKey

	encodingConfig testutils.EncodingConfig
	random         *rand.Rand
	accounts       []testutils.Account
	gasTokenDenom  string
}

func TestProposalsTestSuite(t *testing.T) {
	suite.Run(t, new(ProposalsTestSuite))
}

func (s *ProposalsTestSuite) SetupTest() {
	// Set up basic TX encoding config.
	s.encodingConfig = testutils.CreateTestEncodingConfig()

	// Create a few random accounts
	s.random = rand.New(rand.NewSource(1))
	s.accounts = testutils.RandomAccounts(s.random, 5)
	s.gasTokenDenom = "stake"

	s.key = storetypes.NewKVStoreKey("test")
	testCtx := testutil.DefaultContextWithDB(s.T(), s.key, storetypes.NewTransientStoreKey("transient_test"))
	s.ctx = testCtx.Ctx.WithIsCheckTx(true)
	s.ctx = s.ctx.WithBlockHeight(1)
}

func (s *ProposalsTestSuite) SetupSubTest() {
	s.setBlockParams(1000000000000, 1000000000000)
}

func (s *ProposalsTestSuite) TestPrepareProposal() {
	s.Run("can prepare a proposal with no transactions", func() {
		// Set up the default lane with no transactions
		defaultLane := s.setUpStandardLane(math.LegacyMustNewDecFromStr("1"), nil)

		proposalHandler := s.setUpProposalHandlers([]block.Lane{defaultLane}).PrepareProposalHandler()

		resp := proposalHandler(s.ctx, cometabci.RequestPrepareProposal{Height: 2})
		s.Require().NotNil(resp)
		s.Require().Equal(1, len(resp.Txs))

		info := s.getProposalInfo(resp.Txs[0])
		s.Require().NotNil(info)
		s.Require().Equal(0, len(info.TxsByLane))

		maxBlockSize, maxGasLimit := proposals.GetBlockLimits(s.ctx)
		s.Require().Equal(maxBlockSize, info.MaxBlockSize)
		s.Require().Equal(maxGasLimit, info.MaxGasLimit)

		s.Require().LessOrEqual(info.BlockSize, info.MaxBlockSize)
		s.Require().LessOrEqual(info.GasLimit, info.MaxGasLimit)
	})

	s.Run("can build a proposal with a single tx from the lane", func() {
		// Create a random transaction that will be inserted into the default lane
		tx, err := testutils.CreateRandomTx(
			s.encodingConfig.TxConfig,
			s.accounts[0],
			0,
			1,
			0,
			1,
			sdk.NewCoin(s.gasTokenDenom, math.NewInt(1000000)),
		)
		s.Require().NoError(err)

		// Set up the default lane
		defaultLane := s.setUpStandardLane(math.LegacyMustNewDecFromStr("1"), map[sdk.Tx]bool{tx: true})
		s.Require().NoError(defaultLane.Insert(sdk.Context{}, tx))

		proposalHandler := s.setUpProposalHandlers([]block.Lane{defaultLane}).PrepareProposalHandler()
		resp := proposalHandler(s.ctx, cometabci.RequestPrepareProposal{Height: 2})
		s.Require().NotNil(resp)

		proposal := s.getTxBytes(tx)
		s.Require().Equal(2, len(resp.Txs))
		s.Require().Equal(proposal, resp.Txs[1:])

		info := s.getProposalInfo(resp.Txs[0])
		s.Require().NotNil(info)
		s.Require().Equal(1, len(info.TxsByLane))
		s.Require().Equal(uint64(1), info.TxsByLane[defaultLane.Name()])

		maxBlockSize, maxGasLimit := proposals.GetBlockLimits(s.ctx)
		s.Require().Equal(maxBlockSize, info.MaxBlockSize)
		s.Require().Equal(maxGasLimit, info.MaxGasLimit)

		s.Require().LessOrEqual(info.BlockSize, info.MaxBlockSize)
		s.Require().LessOrEqual(info.GasLimit, info.MaxGasLimit)
	})

	s.Run("can build a proposal with multiple txs from the lane", func() {
		// Create a random transaction that will be inserted into the default lane
		tx1, err := testutils.CreateRandomTx(
			s.encodingConfig.TxConfig,
			s.accounts[0],
			0,
			1,
			0,
			1,
			sdk.NewCoin(s.gasTokenDenom, math.NewInt(1000000)),
		)
		s.Require().NoError(err)

		// Create a second random transaction that will be inserted into the default lane
		tx2, err := testutils.CreateRandomTx(
			s.encodingConfig.TxConfig,
			s.accounts[1],
			1,
			1,
			0,
			1,
			sdk.NewCoin(s.gasTokenDenom, math.NewInt(200000000)),
		)
		s.Require().NoError(err)

		// Set up the default lane with both transactions passing
		defaultLane := s.setUpStandardLane(math.LegacyMustNewDecFromStr("1"), map[sdk.Tx]bool{tx1: true, tx2: true})
		s.Require().NoError(defaultLane.Insert(sdk.Context{}, tx1))
		s.Require().NoError(defaultLane.Insert(sdk.Context{}, tx2))

		proposalHandler := s.setUpProposalHandlers([]block.Lane{defaultLane}).PrepareProposalHandler()
		resp := proposalHandler(s.ctx, cometabci.RequestPrepareProposal{Height: 2})
		s.Require().NotNil(resp)

		proposal := s.getTxBytes(tx2, tx1)
		s.Require().Equal(3, len(resp.Txs))
		s.Require().Equal(proposal, resp.Txs[1:])

		info := s.getProposalInfo(resp.Txs[0])
		s.Require().NotNil(info)
		s.Require().Equal(1, len(info.TxsByLane))
		s.Require().Equal(uint64(2), info.TxsByLane[defaultLane.Name()])

		maxBlockSize, maxGasLimit := proposals.GetBlockLimits(s.ctx)
		s.Require().Equal(maxBlockSize, info.MaxBlockSize)
		s.Require().Equal(maxGasLimit, info.MaxGasLimit)

		s.Require().LessOrEqual(info.BlockSize, info.MaxBlockSize)
		s.Require().LessOrEqual(info.GasLimit, info.MaxGasLimit)
	})

	s.Run("can build a proposal with single tx with other that fails", func() {
		// Create a random transaction that will be inserted into the default lane
		tx1, err := testutils.CreateRandomTx(
			s.encodingConfig.TxConfig,
			s.accounts[0],
			0,
			1,
			0,
			1,
			sdk.NewCoin(s.gasTokenDenom, math.NewInt(1000000)),
		)
		s.Require().NoError(err)

		// Create a second random transaction that will be inserted into the default lane
		tx2, err := testutils.CreateRandomTx(
			s.encodingConfig.TxConfig,
			s.accounts[1],
			1,
			1,
			0,
			1,
			sdk.NewCoin(s.gasTokenDenom, math.NewInt(200000000)),
		)
		s.Require().NoError(err)

		// Set up the default lane with both transactions passing
		defaultLane := s.setUpStandardLane(math.LegacyMustNewDecFromStr("1"), map[sdk.Tx]bool{tx1: true, tx2: false})
		s.Require().NoError(defaultLane.Insert(sdk.Context{}, tx1))
		s.Require().NoError(defaultLane.Insert(sdk.Context{}, tx2))

		proposalHandler := s.setUpProposalHandlers([]block.Lane{defaultLane}).PrepareProposalHandler()
		resp := proposalHandler(s.ctx, cometabci.RequestPrepareProposal{Height: 2})
		s.Require().NotNil(resp)

		proposal := s.getTxBytes(tx1)
		s.Require().Equal(2, len(resp.Txs))
		s.Require().Equal(proposal, resp.Txs[1:])

		info := s.getProposalInfo(resp.Txs[0])
		s.Require().NotNil(info)
		s.Require().Equal(1, len(info.TxsByLane))
		s.Require().Equal(uint64(1), info.TxsByLane[defaultLane.Name()])

		maxBlockSize, maxGasLimit := proposals.GetBlockLimits(s.ctx)
		s.Require().Equal(maxBlockSize, info.MaxBlockSize)
		s.Require().Equal(maxGasLimit, info.MaxGasLimit)

		s.Require().LessOrEqual(info.BlockSize, info.MaxBlockSize)
		s.Require().LessOrEqual(info.GasLimit, info.MaxGasLimit)
	})

	s.Run("can build a proposal an empty proposal with multiple lanes", func() {
		mevLane := s.setUpTOBLane(math.LegacyMustNewDecFromStr("0.5"), nil)
		defaultLane := s.setUpStandardLane(math.LegacyMustNewDecFromStr("0.5"), nil)

		proposalHandler := s.setUpProposalHandlers([]block.Lane{mevLane, defaultLane}).PrepareProposalHandler()

		resp := proposalHandler(s.ctx, cometabci.RequestPrepareProposal{Height: 2})
		s.Require().NotNil(resp)

		s.Require().Equal(1, len(resp.Txs))

		info := s.getProposalInfo(resp.Txs[0])
		s.Require().NotNil(info)
		s.Require().Equal(0, len(info.TxsByLane))

		maxBlockSize, maxGasLimit := proposals.GetBlockLimits(s.ctx)
		s.Require().Equal(maxBlockSize, info.MaxBlockSize)
		s.Require().Equal(maxGasLimit, info.MaxGasLimit)

		s.Require().LessOrEqual(info.BlockSize, info.MaxBlockSize)
		s.Require().LessOrEqual(info.GasLimit, info.MaxGasLimit)
	})

	s.Run("can build a proposal with transactions from a single lane given multiple lanes", func() {
		// Create a bid tx that includes a single bundled tx
		tx, bundleTxs, err := testutils.CreateAuctionTx(
			s.encodingConfig.TxConfig,
			s.accounts[0],
			sdk.NewCoin(s.gasTokenDenom, math.NewInt(1000000)),
			0,
			0,
			s.accounts[0:1],
			100,
		)
		s.Require().NoError(err)

		// Set up the TOB lane with the bid tx and the bundled tx
		mevLane := s.setUpTOBLane(math.LegacyMustNewDecFromStr("0.5"), map[sdk.Tx]bool{
			tx:           true,
			bundleTxs[0]: true,
		})
		s.Require().NoError(mevLane.Insert(sdk.Context{}, tx))

		defaultLane := s.setUpStandardLane(math.LegacyMustNewDecFromStr("0.5"), nil)

		proposalHandler := s.setUpProposalHandlers([]block.Lane{mevLane, defaultLane}).PrepareProposalHandler()

		resp := proposalHandler(s.ctx, cometabci.RequestPrepareProposal{Height: 2})
		s.Require().NotNil(resp)

		proposal := s.getTxBytes(tx, bundleTxs[0])
		s.Require().Equal(3, len(resp.Txs))
		s.Require().Equal(proposal, resp.Txs[1:])

		info := s.getProposalInfo(resp.Txs[0])
		s.Require().NotNil(info)
		s.Require().Equal(1, len(info.TxsByLane))
		s.Require().Equal(uint64(2), info.TxsByLane[mevLane.Name()])

		maxBlockSize, maxGasLimit := proposals.GetBlockLimits(s.ctx)
		s.Require().Equal(maxBlockSize, info.MaxBlockSize)
		s.Require().Equal(maxGasLimit, info.MaxGasLimit)

		s.Require().LessOrEqual(info.BlockSize, info.MaxBlockSize)
		s.Require().LessOrEqual(info.GasLimit, info.MaxGasLimit)
	})

	s.Run("can ignore txs that are already included in a proposal", func() {
		// Create a bid tx that includes a single bundled tx
		tx, bundleTxs, err := testutils.CreateAuctionTx(
			s.encodingConfig.TxConfig,
			s.accounts[0],
			sdk.NewCoin(s.gasTokenDenom, math.NewInt(1000000)),
			0,
			0,
			s.accounts[0:1],
			100,
		)
		s.Require().NoError(err)

		// Set up the TOB lane with the bid tx and the bundled tx
		mevLane := s.setUpTOBLane(math.LegacyMustNewDecFromStr("0.5"), map[sdk.Tx]bool{
			tx:           true,
			bundleTxs[0]: true,
		})
		s.Require().NoError(mevLane.Insert(sdk.Context{}, tx))

		// Set up the default lane with the bid tx and the bundled tx
		defaultLane := s.setUpStandardLane(math.LegacyMustNewDecFromStr("0.5"), map[sdk.Tx]bool{
			tx:           true,
			bundleTxs[0]: true,
		})
		s.Require().NoError(defaultLane.Insert(sdk.Context{}, tx))
		s.Require().NoError(defaultLane.Insert(sdk.Context{}, bundleTxs[0]))

		proposalHandler := s.setUpProposalHandlers([]block.Lane{mevLane, defaultLane}).PrepareProposalHandler()

		resp := proposalHandler(s.ctx, cometabci.RequestPrepareProposal{Height: 2})
		s.Require().NotNil(resp)

		proposal := s.getTxBytes(tx, bundleTxs[0])
		s.Require().Equal(3, len(resp.Txs))
		s.Require().Equal(proposal, resp.Txs[1:])

		info := s.getProposalInfo(resp.Txs[0])
		s.Require().NotNil(info)
		s.Require().Equal(1, len(info.TxsByLane))
		s.Require().Equal(uint64(2), info.TxsByLane[mevLane.Name()])

		maxBlockSize, maxGasLimit := proposals.GetBlockLimits(s.ctx)
		s.Require().Equal(maxBlockSize, info.MaxBlockSize)
		s.Require().Equal(maxGasLimit, info.MaxGasLimit)

		s.Require().LessOrEqual(info.BlockSize, info.MaxBlockSize)
		s.Require().LessOrEqual(info.GasLimit, info.MaxGasLimit)
	})

	s.Run("can build a proposal where first lane has failing tx and second lane has a valid tx", func() {
		// Create a bid tx that includes a single bundled tx
		tx, bundleTxs, err := testutils.CreateAuctionTx(
			s.encodingConfig.TxConfig,
			s.accounts[0],
			sdk.NewCoin(s.gasTokenDenom, math.NewInt(1000000)),
			0,
			0,
			s.accounts[0:1],
			100,
		)
		s.Require().NoError(err)

		// Set up the TOB lane with the bid tx and the bundled tx
		mevLane := s.setUpTOBLane(math.LegacyMustNewDecFromStr("0.5"), map[sdk.Tx]bool{
			tx:           false,
			bundleTxs[0]: true,
		})
		s.Require().NoError(mevLane.Insert(sdk.Context{}, tx))

		// Set up the default lane with the bid tx and the bundled tx
		defaultLane := s.setUpStandardLane(math.LegacyMustNewDecFromStr("0"), map[sdk.Tx]bool{
			// Even though this passes it should not include it in the proposal because it is in the ignore list
			tx:           true,
			bundleTxs[0]: true,
		})
		s.Require().NoError(defaultLane.Insert(sdk.Context{}, tx))
		s.Require().NoError(defaultLane.Insert(sdk.Context{}, bundleTxs[0]))

		proposalHandler := s.setUpProposalHandlers([]block.Lane{mevLane, defaultLane}).PrepareProposalHandler()

		resp := proposalHandler(s.ctx, cometabci.RequestPrepareProposal{Height: 2})
		s.Require().NotNil(resp)

		proposal := s.getTxBytes(bundleTxs[0])
		s.Require().Equal(2, len(resp.Txs))
		s.Require().Equal(proposal, resp.Txs[1:])

		info := s.getProposalInfo(resp.Txs[0])
		s.Require().NotNil(info)
		s.Require().Equal(1, len(info.TxsByLane))
		s.Require().Equal(uint64(1), info.TxsByLane[defaultLane.Name()])

		maxBlockSize, maxGasLimit := proposals.GetBlockLimits(s.ctx)
		s.Require().Equal(maxBlockSize, info.MaxBlockSize)
		s.Require().Equal(maxGasLimit, info.MaxGasLimit)

		s.Require().LessOrEqual(info.BlockSize, info.MaxBlockSize)
		s.Require().LessOrEqual(info.GasLimit, info.MaxGasLimit)
	})

	s.Run("can build a proposal where first lane cannot fit txs but second lane can", func() {
		// Create a bid tx that includes a single bundled tx
		tx, bundleTxs, err := testutils.CreateAuctionTx(
			s.encodingConfig.TxConfig,
			s.accounts[0],
			sdk.NewCoin(s.gasTokenDenom, math.NewInt(1000000)),
			0,
			0,
			s.accounts[0:1],
			100,
		)
		s.Require().NoError(err)

		// Set up the TOB lane with the bid tx and the bundled tx
		mevLane := s.setUpTOBLane(math.LegacyMustNewDecFromStr("0.5"), map[sdk.Tx]bool{
			tx:           true,
			bundleTxs[0]: true,
		})
		s.Require().NoError(mevLane.Insert(sdk.Context{}, tx))

		// Set up the default lane with the bid tx and the bundled tx
		defaultLane := s.setUpStandardLane(math.LegacyMustNewDecFromStr("0.0"), map[sdk.Tx]bool{
			// Even though this passes it should not include it in the proposal because it is in the ignore list
			tx:           true,
			bundleTxs[0]: true,
		})
		s.Require().NoError(defaultLane.Insert(sdk.Context{}, tx))
		s.Require().NoError(defaultLane.Insert(sdk.Context{}, bundleTxs[0]))

		proposalHandler := s.setUpProposalHandlers([]block.Lane{mevLane, defaultLane}).PrepareProposalHandler()
		proposal := s.getTxBytes(tx, bundleTxs[0])
		size := int64(len(proposal[0]) - 1)

		s.setBlockParams(10000000, size)
		resp := proposalHandler(s.ctx, cometabci.RequestPrepareProposal{Height: 2})
		s.Require().NotNil(resp)

		s.Require().Equal(2, len(resp.Txs))
		s.Require().Equal(proposal[1:], resp.Txs[1:])

		info := s.getProposalInfo(resp.Txs[0])
		s.Require().NotNil(info)
		s.Require().Equal(1, len(info.TxsByLane))
		s.Require().Equal(uint64(1), info.TxsByLane[defaultLane.Name()])

		maxBlockSize, maxGasLimit := proposals.GetBlockLimits(s.ctx)
		s.Require().Equal(maxBlockSize, info.MaxBlockSize)
		s.Require().Equal(maxGasLimit, info.MaxGasLimit)

		s.Require().LessOrEqual(info.BlockSize, info.MaxBlockSize)
		s.Require().LessOrEqual(info.GasLimit, info.MaxGasLimit)
	})

	s.Run("can build a proposal with single tx from middle lane", func() {
		mevLane := s.setUpTOBLane(math.LegacyMustNewDecFromStr("0.25"), nil)

		freeTx, err := testutils.CreateFreeTx(
			s.encodingConfig.TxConfig,
			s.accounts[0],
			0,
			0,
			"test",
			sdk.NewCoin(s.gasTokenDenom, math.NewInt(1000000)),
			sdk.NewCoin(s.gasTokenDenom, math.NewInt(1000000)),
		)
		s.Require().NoError(err)

		defaultLane := s.setUpStandardLane(math.LegacyMustNewDecFromStr("0.0"), map[sdk.Tx]bool{
			freeTx: true,
		})
		s.Require().NoError(defaultLane.Insert(sdk.Context{}, freeTx))

		freeLane := s.setUpFreeLane(math.LegacyMustNewDecFromStr("0.25"), map[sdk.Tx]bool{
			freeTx: true,
		})
		s.Require().NoError(freeLane.Insert(sdk.Context{}, freeTx))

		proposalHandler := s.setUpProposalHandlers([]block.Lane{mevLane, freeLane, defaultLane}).PrepareProposalHandler()

		proposal := s.getTxBytes(freeTx)

		resp := proposalHandler(s.ctx, cometabci.RequestPrepareProposal{Height: 2})
		s.Require().NotNil(resp)

		s.Require().Equal(2, len(resp.Txs))
		s.Require().Equal(proposal, resp.Txs[1:])

		info := s.getProposalInfo(resp.Txs[0])
		s.Require().NotNil(info)
		s.Require().Equal(1, len(info.TxsByLane))
		s.Require().Equal(uint64(1), info.TxsByLane[freeLane.Name()])

		maxBlockSize, maxGasLimit := proposals.GetBlockLimits(s.ctx)
		s.Require().Equal(maxBlockSize, info.MaxBlockSize)
		s.Require().Equal(maxGasLimit, info.MaxGasLimit)

		s.Require().LessOrEqual(info.BlockSize, info.MaxBlockSize)
		s.Require().LessOrEqual(info.GasLimit, info.MaxGasLimit)
	})

	s.Run("transaction from every lane", func() {
		// Create a bid tx that includes a single bundled tx
		tx, bundleTxs, err := testutils.CreateAuctionTx(
			s.encodingConfig.TxConfig,
			s.accounts[0],
			sdk.NewCoin(s.gasTokenDenom, math.NewInt(1000000)),
			0,
			0,
			s.accounts[0:4],
			100,
		)
		s.Require().NoError(err)

		// Create a normal tx
		normalTx, err := testutils.CreateRandomTx(
			s.encodingConfig.TxConfig,
			s.accounts[1],
			0,
			0,
			0,
			1,
			sdk.NewCoin(s.gasTokenDenom, math.NewInt(1000000)),
		)
		s.Require().NoError(err)

		// Create a free tx
		freeTx, err := testutils.CreateFreeTx(
			s.encodingConfig.TxConfig,
			s.accounts[2],
			0,
			0,
			"test",
			sdk.NewCoin(s.gasTokenDenom, math.NewInt(1000000)),
			sdk.NewCoin(s.gasTokenDenom, math.NewInt(1000000)),
		)
		s.Require().NoError(err)

		mevLane := s.setUpTOBLane(math.LegacyMustNewDecFromStr("0.25"), map[sdk.Tx]bool{
			tx:           true,
			bundleTxs[0]: true,
			bundleTxs[1]: true,
			bundleTxs[2]: true,
			bundleTxs[3]: true,
		})
		mevLane.Insert(sdk.Context{}, tx)

		defaultLane := s.setUpStandardLane(math.LegacyMustNewDecFromStr("0.0"), map[sdk.Tx]bool{
			normalTx: true,
		})
		defaultLane.Insert(sdk.Context{}, normalTx)

		freeLane := s.setUpFreeLane(math.LegacyMustNewDecFromStr("0.25"), map[sdk.Tx]bool{
			freeTx: true,
		})
		freeLane.Insert(sdk.Context{}, freeTx)

		proposalHandler := s.setUpProposalHandlers([]block.Lane{mevLane, freeLane, defaultLane}).PrepareProposalHandler()
		proposal := s.getTxBytes(tx, bundleTxs[0], bundleTxs[1], bundleTxs[2], bundleTxs[3], freeTx, normalTx)

		resp := proposalHandler(s.ctx, cometabci.RequestPrepareProposal{Height: 2})
		s.Require().NotNil(resp)

		s.Require().Equal(8, len(resp.Txs))
		s.Require().Equal(proposal, resp.Txs[1:])

		info := s.getProposalInfo(resp.Txs[0])
		s.Require().NotNil(info)
		s.Require().Equal(3, len(info.TxsByLane))
		s.Require().Equal(uint64(1), info.TxsByLane[freeLane.Name()])
		s.Require().Equal(uint64(5), info.TxsByLane[mevLane.Name()])
		s.Require().Equal(uint64(1), info.TxsByLane[defaultLane.Name()])

		maxBlockSize, maxGasLimit := proposals.GetBlockLimits(s.ctx)
		s.Require().Equal(maxBlockSize, info.MaxBlockSize)
		s.Require().Equal(maxGasLimit, info.MaxGasLimit)

		s.Require().LessOrEqual(info.BlockSize, info.MaxBlockSize)
		s.Require().LessOrEqual(info.GasLimit, info.MaxGasLimit)
	})

	s.Run("can build a proposal where first lane does not have enough gas but second lane does", func() {
		// set up the gas block limit for the proposal
		s.setBlockParams(100, 1000000000)

		// Create a bid tx that includes a single bundled tx
		tx, bundleTxs, err := testutils.CreateAuctionTx(
			s.encodingConfig.TxConfig,
			s.accounts[0],
			sdk.NewCoin(s.gasTokenDenom, math.NewInt(1000000)),
			0,
			0,
			s.accounts[0:1],
			51,
		)
		s.Require().NoError(err)

		// Set up the TOB lane with the bid tx and the bundled tx
		mevLane := s.setUpTOBLane(math.LegacyMustNewDecFromStr("0.5"), map[sdk.Tx]bool{
			tx:           true,
			bundleTxs[0]: true,
		})
		s.Require().NoError(mevLane.Insert(sdk.Context{}, tx))

		// create a random tx to be included in the default lane
		normalTx, err := testutils.CreateRandomTx(
			s.encodingConfig.TxConfig,
			s.accounts[1],
			0,
			0,
			0,
			100, // This should consume all of the block limit
			sdk.NewCoin(s.gasTokenDenom, math.NewInt(1000000)),
		)
		s.Require().NoError(err)

		defaultLane := s.setUpStandardLane(math.LegacyMustNewDecFromStr("0.0"), map[sdk.Tx]bool{
			normalTx: true,
		})
		s.Require().NoError(defaultLane.Insert(sdk.Context{}, normalTx))

		proposalHandler := s.setUpProposalHandlers([]block.Lane{mevLane, defaultLane}).PrepareProposalHandler()
		proposal := s.getTxBytes(tx, bundleTxs[0], normalTx)

		// Should be theoretically sufficient to fit the bid tx and the bundled tx + normal tx
		resp := proposalHandler(s.ctx, cometabci.RequestPrepareProposal{Height: 2})
		s.Require().NotNil(resp)

		s.Require().Equal(2, len(resp.Txs))
		s.Require().Equal(proposal[2:], resp.Txs[1:])

		info := s.getProposalInfo(resp.Txs[0])
		s.Require().NotNil(info)
		s.Require().Equal(1, len(info.TxsByLane))
		s.Require().Equal(uint64(1), info.TxsByLane[defaultLane.Name()])

		maxBlockSize, maxGasLimit := proposals.GetBlockLimits(s.ctx)
		s.Require().Equal(maxBlockSize, info.MaxBlockSize)
		s.Require().Equal(maxGasLimit, info.MaxGasLimit)

		s.Require().LessOrEqual(info.BlockSize, info.MaxBlockSize)
		s.Require().LessOrEqual(info.GasLimit, info.MaxGasLimit)
	})
}

func (s *ProposalsTestSuite) TestPrepareProposalEdgeCases() {
	s.Run("can build a proposal if a lane panics first", func() {
		panicLane := s.setUpPanicLane(math.LegacyMustNewDecFromStr("0.25"))

		tx, err := testutils.CreateRandomTx(
			s.encodingConfig.TxConfig,
			s.accounts[0],
			0,
			0,
			0,
			1,
			sdk.NewCoin(s.gasTokenDenom, math.NewInt(1000000)),
		)
		s.Require().NoError(err)

		defaultLane := s.setUpStandardLane(math.LegacyMustNewDecFromStr("0.0"), map[sdk.Tx]bool{
			tx: true,
		})
		s.Require().NoError(defaultLane.Insert(sdk.Context{}, tx))

		mempool := block.NewLanedMempool(log.NewTMLogger(os.Stdout), false, panicLane, defaultLane)

		proposalHandler := abci.NewProposalHandler(
			log.NewTMLogger(os.Stdout),
			s.encodingConfig.TxConfig.TxDecoder(),
			s.encodingConfig.TxConfig.TxEncoder(),
			mempool,
		).PrepareProposalHandler()

		resp := proposalHandler(s.ctx, cometabci.RequestPrepareProposal{Height: 2})
		s.Require().NotNil(resp)

		proposal := s.getTxBytes(tx)
		s.Require().Equal(2, len(resp.Txs))
		s.Require().Equal(proposal, resp.Txs[1:])

		info := s.getProposalInfo(resp.Txs[0])
		s.Require().NotNil(info)
		s.Require().Equal(1, len(info.TxsByLane))
		s.Require().Equal(uint64(1), info.TxsByLane[defaultLane.Name()])

		maxBlockSize, maxGasLimit := proposals.GetBlockLimits(s.ctx)
		s.Require().Equal(maxBlockSize, info.MaxBlockSize)
		s.Require().Equal(maxGasLimit, info.MaxGasLimit)

		s.Require().LessOrEqual(info.BlockSize, info.MaxBlockSize)
		s.Require().LessOrEqual(info.GasLimit, info.MaxGasLimit)
	})

	s.Run("can build a proposal if second lane panics", func() {
		panicLane := s.setUpPanicLane(math.LegacyMustNewDecFromStr("0.25"))

		tx, err := testutils.CreateRandomTx(
			s.encodingConfig.TxConfig,
			s.accounts[0],
			0,
			0,
			0,
			1,
			sdk.NewCoin(s.gasTokenDenom, math.NewInt(1000000)),
		)
		s.Require().NoError(err)

		defaultLane := s.setUpStandardLane(math.LegacyMustNewDecFromStr("0.0"), map[sdk.Tx]bool{
			tx: true,
		})
		s.Require().NoError(defaultLane.Insert(sdk.Context{}, tx))

		mempool := block.NewLanedMempool(log.NewTMLogger(os.Stdout), false, defaultLane, panicLane)

		proposalHandler := abci.NewProposalHandler(
			log.NewTMLogger(os.Stdout),
			s.encodingConfig.TxConfig.TxDecoder(),
			s.encodingConfig.TxConfig.TxEncoder(),
			mempool,
		).PrepareProposalHandler()

		resp := proposalHandler(s.ctx, cometabci.RequestPrepareProposal{Height: 2})
		s.Require().NotNil(resp)

		proposal := s.getTxBytes(tx)
		s.Require().Equal(2, len(resp.Txs))
		s.Require().Equal(proposal, resp.Txs[1:])

		info := s.getProposalInfo(resp.Txs[0])
		s.Require().NotNil(info)
		s.Require().Equal(1, len(info.TxsByLane))
		s.Require().Equal(uint64(1), info.TxsByLane[defaultLane.Name()])

		maxBlockSize, maxGasLimit := proposals.GetBlockLimits(s.ctx)
		s.Require().Equal(maxBlockSize, info.MaxBlockSize)
		s.Require().Equal(maxGasLimit, info.MaxGasLimit)

		s.Require().LessOrEqual(info.BlockSize, info.MaxBlockSize)
		s.Require().LessOrEqual(info.GasLimit, info.MaxGasLimit)
	})

	s.Run("can build a proposal if multiple consecutive lanes panic", func() {
		panicLane := s.setUpPanicLane(math.LegacyMustNewDecFromStr("0.25"))
		panicLane2 := s.setUpPanicLane(math.LegacyMustNewDecFromStr("0.25"))

		tx, err := testutils.CreateRandomTx(
			s.encodingConfig.TxConfig,
			s.accounts[0],
			0,
			0,
			0,
			1,
			sdk.NewCoin(s.gasTokenDenom, math.NewInt(1000000)),
		)
		s.Require().NoError(err)

		defaultLane := s.setUpStandardLane(math.LegacyMustNewDecFromStr("0.0"), map[sdk.Tx]bool{
			tx: true,
		})
		s.Require().NoError(defaultLane.Insert(sdk.Context{}, tx))

		mempool := block.NewLanedMempool(log.NewTMLogger(os.Stdout), false, panicLane, panicLane2, defaultLane)

		proposalHandler := abci.NewProposalHandler(
			log.NewTMLogger(os.Stdout),
			s.encodingConfig.TxConfig.TxDecoder(),
			s.encodingConfig.TxConfig.TxEncoder(),
			mempool,
		).PrepareProposalHandler()

		resp := proposalHandler(s.ctx, cometabci.RequestPrepareProposal{Height: 2})
		s.Require().NotNil(resp)

		proposal := s.getTxBytes(tx)
		s.Require().Equal(2, len(resp.Txs))
		s.Require().Equal(proposal, resp.Txs[1:])

		info := s.getProposalInfo(resp.Txs[0])
		s.Require().NotNil(info)
		s.Require().Equal(1, len(info.TxsByLane))
		s.Require().Equal(uint64(1), info.TxsByLane[defaultLane.Name()])

		maxBlockSize, maxGasLimit := proposals.GetBlockLimits(s.ctx)
		s.Require().Equal(maxBlockSize, info.MaxBlockSize)
		s.Require().Equal(maxGasLimit, info.MaxGasLimit)

		s.Require().LessOrEqual(info.BlockSize, info.MaxBlockSize)
		s.Require().LessOrEqual(info.GasLimit, info.MaxGasLimit)
	})

	s.Run("can build a proposal if the last few lanes panic", func() {
		panicLane := s.setUpPanicLane(math.LegacyMustNewDecFromStr("0.25"))
		panicLane2 := s.setUpPanicLane(math.LegacyMustNewDecFromStr("0.25"))

		tx, err := testutils.CreateRandomTx(
			s.encodingConfig.TxConfig,
			s.accounts[0],
			0,
			0,
			0,
			1,
			sdk.NewCoin(s.gasTokenDenom, math.NewInt(1000000)),
		)
		s.Require().NoError(err)

		defaultLane := s.setUpStandardLane(math.LegacyMustNewDecFromStr("0.0"), map[sdk.Tx]bool{
			tx: true,
		})
		s.Require().NoError(defaultLane.Insert(sdk.Context{}, tx))

		mempool := block.NewLanedMempool(log.NewTMLogger(os.Stdout), false, defaultLane, panicLane, panicLane2)

		proposalHandler := abci.NewProposalHandler(
			log.NewTMLogger(os.Stdout),
			s.encodingConfig.TxConfig.TxDecoder(),
			s.encodingConfig.TxConfig.TxEncoder(),
			mempool,
		).PrepareProposalHandler()

		resp := proposalHandler(s.ctx, cometabci.RequestPrepareProposal{Height: 2})
		s.Require().NotNil(resp)

		proposal := s.getTxBytes(tx)
		s.Require().Equal(2, len(resp.Txs))
		s.Require().Equal(proposal, resp.Txs[1:])

		info := s.getProposalInfo(resp.Txs[0])
		s.Require().NotNil(info)
		s.Require().Equal(1, len(info.TxsByLane))
		s.Require().Equal(uint64(1), info.TxsByLane[defaultLane.Name()])

		maxBlockSize, maxGasLimit := proposals.GetBlockLimits(s.ctx)
		s.Require().Equal(maxBlockSize, info.MaxBlockSize)
		s.Require().Equal(maxGasLimit, info.MaxGasLimit)

		s.Require().LessOrEqual(info.BlockSize, info.MaxBlockSize)
		s.Require().LessOrEqual(info.GasLimit, info.MaxGasLimit)
	})
}

func (s *ProposalsTestSuite) TestProcessProposal() {
	s.Run("can process a valid empty proposal", func() {
		mevLane := s.setUpTOBLane(math.LegacyMustNewDecFromStr("0.25"), map[sdk.Tx]bool{})
		freeLane := s.setUpFreeLane(math.LegacyMustNewDecFromStr("0.25"), map[sdk.Tx]bool{})
		defaultLane := s.setUpStandardLane(math.LegacyMustNewDecFromStr("0.0"), map[sdk.Tx]bool{})

		proposalHandler := s.setUpProposalHandlers([]block.Lane{mevLane, freeLane, defaultLane}).ProcessProposalHandler()

		info := s.createProposalInfoBytes(
			0,
			0,
			0,
			0,
			nil,
		)
		proposal := [][]byte{info}

		resp := proposalHandler(s.ctx, cometabci.RequestProcessProposal{Txs: proposal, Height: 2})
		s.Require().NotNil(resp)
		s.Require().Equal(cometabci.ResponseProcessProposal{Status: cometabci.ResponseProcessProposal_ACCEPT}, resp)
	})

	s.Run("can process a valid proposal with a single tx", func() {
		// Create a random transaction that will be inserted into the default lane
		tx, err := testutils.CreateRandomTx(
			s.encodingConfig.TxConfig,
			s.accounts[0],
			0,
			0,
			0,
			1,
			sdk.NewCoin(s.gasTokenDenom, math.NewInt(1000000)),
		)
		s.Require().NoError(err)

		// Mev lane
		mevLane := s.setUpTOBLane(math.LegacyMustNewDecFromStr("0.25"), map[sdk.Tx]bool{})
		defaultLane := s.setUpStandardLane(math.LegacyMustNewDecFromStr("0.0"), map[sdk.Tx]bool{
			tx: true,
		})

		proposal := s.createProposal(map[string]uint64{defaultLane.Name(): 1}, tx)

		proposalHandler := s.setUpProposalHandlers([]block.Lane{mevLane, defaultLane}).ProcessProposalHandler()
		resp := proposalHandler(s.ctx, cometabci.RequestProcessProposal{Txs: proposal, Height: 2})
		s.Require().NotNil(resp)
		s.Require().Equal(cometabci.ResponseProcessProposal{Status: cometabci.ResponseProcessProposal_ACCEPT}, resp)
	})

	s.Run("can process a valid proposal with txs from multiple lanes", func() {
		// Create a random transaction that will be inserted into the default lane
		tx, err := testutils.CreateRandomTx(
			s.encodingConfig.TxConfig,
			s.accounts[0],
			0,
			0,
			0,
			1,
			sdk.NewCoin(s.gasTokenDenom, math.NewInt(1000000)),
		)
		s.Require().NoError(err)

		// create a bid tx that will be inserted into the mev lane
		bidTx, bundleTxs, err := testutils.CreateAuctionTx(
			s.encodingConfig.TxConfig,
			s.accounts[1],
			sdk.NewCoin(s.gasTokenDenom, math.NewInt(1000000)),
			0,
			0,
			s.accounts[1:2],
			100,
		)
		s.Require().NoError(err)

		// create a free tx that will be inserted into the free lane
		freeTx, err := testutils.CreateFreeTx(
			s.encodingConfig.TxConfig,
			s.accounts[2],
			0,
			0,
			"test",
			sdk.NewCoin(s.gasTokenDenom, math.NewInt(1000000)),
			sdk.NewCoin(s.gasTokenDenom, math.NewInt(1000000)),
		)
		s.Require().NoError(err)

		// Mev lane
		mevLane := s.setUpTOBLane(math.LegacyMustNewDecFromStr("0.25"), map[sdk.Tx]bool{
			bidTx:        true,
			bundleTxs[0]: true,
		})
		freeLane := s.setUpFreeLane(math.LegacyMustNewDecFromStr("0.25"), map[sdk.Tx]bool{
			freeTx: true,
		})
		defaultLane := s.setUpStandardLane(math.LegacyMustNewDecFromStr("0.0"), map[sdk.Tx]bool{
			tx: true,
		})

		proposal := s.createProposal(map[string]uint64{defaultLane.Name(): 1, mevLane.Name(): 2, freeLane.Name(): 1}, bidTx, bundleTxs[0], freeTx, tx)

		proposalHandler := s.setUpProposalHandlers([]block.Lane{mevLane, freeLane, defaultLane}).ProcessProposalHandler()
		resp := proposalHandler(s.ctx, cometabci.RequestProcessProposal{Txs: proposal, Height: 2})
		s.Require().NotNil(resp)
		s.Require().Equal(cometabci.ResponseProcessProposal{Status: cometabci.ResponseProcessProposal_ACCEPT}, resp)
	})

	s.Run("rejects a proposal with mismatching block size", func() {
		tx, err := testutils.CreateRandomTx(
			s.encodingConfig.TxConfig,
			s.accounts[0],
			0,
			0,
			0,
			100,
			sdk.NewCoin(s.gasTokenDenom, math.NewInt(1000000)),
		)
		s.Require().NoError(err)

		mevLane := s.setUpTOBLane(math.LegacyMustNewDecFromStr("0.25"), map[sdk.Tx]bool{})
		defaultLane := s.setUpStandardLane(math.LegacyMustNewDecFromStr("0.0"), map[sdk.Tx]bool{
			tx: true,
		})

		proposal := s.createProposal(map[string]uint64{defaultLane.Name(): 1, mevLane.Name(): 0}, tx)

		// modify the block size to be 1
		info := s.getProposalInfo(proposal[0])
		info.BlockSize--
		infoBz, err := info.Marshal()
		s.Require().NoError(err)
		proposal[0] = infoBz

		proposalHandler := s.setUpProposalHandlers([]block.Lane{mevLane, defaultLane}).ProcessProposalHandler()
		resp := proposalHandler(s.ctx, cometabci.RequestProcessProposal{Txs: proposal, Height: 2})
		s.Require().Equal(cometabci.ResponseProcessProposal{Status: cometabci.ResponseProcessProposal_REJECT}, resp)
	})

	s.Run("rejects a proposal with mismatching gas limit", func() {
		tx, err := testutils.CreateRandomTx(
			s.encodingConfig.TxConfig,
			s.accounts[0],
			0,
			0,
			0,
			100,
			sdk.NewCoin(s.gasTokenDenom, math.NewInt(1000000)),
		)
		s.Require().NoError(err)

		mevLane := s.setUpTOBLane(math.LegacyMustNewDecFromStr("0.25"), map[sdk.Tx]bool{})
		defaultLane := s.setUpStandardLane(math.LegacyMustNewDecFromStr("0.0"), map[sdk.Tx]bool{
			tx: true,
		})

		proposal := s.createProposal(map[string]uint64{defaultLane.Name(): 1, mevLane.Name(): 0}, tx)

		// modify the block size to be 1
		info := s.getProposalInfo(proposal[0])
		info.GasLimit--
		infoBz, err := info.Marshal()
		s.Require().NoError(err)
		proposal[0] = infoBz

		proposalHandler := s.setUpProposalHandlers([]block.Lane{mevLane, defaultLane}).ProcessProposalHandler()
		resp := proposalHandler(s.ctx, cometabci.RequestProcessProposal{Txs: proposal, Height: 2})
		s.Require().Equal(cometabci.ResponseProcessProposal{Status: cometabci.ResponseProcessProposal_REJECT}, resp)
	})

	s.Run("rejects a proposal with bad txs", func() {
		mevLane := s.setUpTOBLane(math.LegacyMustNewDecFromStr("0.25"), map[sdk.Tx]bool{})
		freeLane := s.setUpFreeLane(math.LegacyMustNewDecFromStr("0.25"), map[sdk.Tx]bool{})
		defaultLane := s.setUpStandardLane(math.LegacyMustNewDecFromStr("0.0"), map[sdk.Tx]bool{})

		proposalHandler := s.setUpProposalHandlers([]block.Lane{mevLane, freeLane, defaultLane}).ProcessProposalHandler()

		info := s.createProposalInfoBytes(
			0,
			0,
			0,
			0,
			map[string]uint64{
				mevLane.Name():     0,
				freeLane.Name():    0,
				defaultLane.Name(): 1,
			},
		)
		proposal := [][]byte{info, {0x01, 0x02, 0x03}}

		resp := proposalHandler(s.ctx, cometabci.RequestProcessProposal{Txs: proposal, Height: 2})
		s.Require().Equal(cometabci.ResponseProcessProposal{Status: cometabci.ResponseProcessProposal_REJECT}, resp)
	})

	s.Run("rejects a proposal when a lane panics", func() {
		mevLane := s.setUpTOBLane(math.LegacyMustNewDecFromStr("0.25"), map[sdk.Tx]bool{})
		panicLane := s.setUpPanicLane(math.LegacyMustNewDecFromStr("0.0"))

		txbz, err := testutils.CreateRandomTxBz(
			s.encodingConfig.TxConfig,
			s.accounts[0],
			0,
			0,
			0,
			1,
		)
		s.Require().NoError(err)

		proposalHandler := s.setUpProposalHandlers([]block.Lane{mevLane, panicLane}).ProcessProposalHandler()

		info := s.createProposalInfoBytes(
			0,
			0,
			0,
			0,
			map[string]uint64{
				panicLane.Name(): 1,
			},
		)
		proposal := [][]byte{info, txbz}

		resp := proposalHandler(s.ctx, cometabci.RequestProcessProposal{Txs: proposal, Height: 2})
		s.Require().Equal(cometabci.ResponseProcessProposal{Status: cometabci.ResponseProcessProposal_REJECT}, resp)
	})

	s.Run("can process a invalid proposal (out of order)", func() {
		// Create a random transaction that will be inserted into the default lane
		tx, err := testutils.CreateAuctionTxWithSigners(
			s.encodingConfig.TxConfig,
			s.accounts[0],
			sdk.NewCoin(s.gasTokenDenom, math.NewInt(1000000)),
			0,
			1,
			nil,
		)
		s.Require().NoError(err)

		// Create a random transaction that will be inserted into the default lane
		tx2, err := testutils.CreateRandomTx(
			s.encodingConfig.TxConfig,
			s.accounts[2],
			0,
			1,
			0,
			1,
			sdk.NewCoin(s.gasTokenDenom, math.NewInt(2000000)),
		)
		s.Require().NoError(err)

		// Mev lane
		mevLane := s.setUpTOBLane(math.LegacyMustNewDecFromStr("0.0"), map[sdk.Tx]bool{tx: true})

		// Set up the default lane
		defaultLane := s.setUpStandardLane(math.LegacyMustNewDecFromStr("0.0"), map[sdk.Tx]bool{tx2: true})
		s.Require().NoError(defaultLane.Insert(sdk.Context{}, tx))

		proposal := s.createProposal(map[string]uint64{defaultLane.Name(): 1, mevLane.Name(): 1}, tx2, tx)

		proposalHandler := s.setUpProposalHandlers([]block.Lane{mevLane, defaultLane}).ProcessProposalHandler()
		resp := proposalHandler(s.ctx, cometabci.RequestProcessProposal{Txs: proposal, Height: 2})
		s.Require().NotNil(resp)
		s.Require().Equal(cometabci.ResponseProcessProposal{Status: cometabci.ResponseProcessProposal_REJECT}, resp)
	})

	s.Run("can process a invalid proposal where first lane is valid second is not", func() {
		bidTx, bundle, err := testutils.CreateAuctionTx(
			s.encodingConfig.TxConfig,
			s.accounts[0],
			sdk.NewCoin(s.gasTokenDenom, math.NewInt(1000000)),
			0,
			1,
			s.accounts[0:2],
			10,
		)
		s.Require().NoError(err)

		normalTx, err := testutils.CreateRandomTx(
			s.encodingConfig.TxConfig,
			s.accounts[1],
			0,
			1,
			0,
			1,
			sdk.NewCoin(s.gasTokenDenom, math.NewInt(3000000)),
		)
		s.Require().NoError(err)

		normalTx2, err := testutils.CreateRandomTx(
			s.encodingConfig.TxConfig,
			s.accounts[2],
			0,
			1,
			0,
			1,
			sdk.NewCoin(s.gasTokenDenom, math.NewInt(3000000)),
		)
		s.Require().NoError(err)

		// Set up the default lane
		defaultLane := s.setUpStandardLane(math.LegacyMustNewDecFromStr("0.5"), map[sdk.Tx]bool{
			normalTx:  true,
			normalTx2: false,
		})

		// Set up the TOB lane
		mevLane := s.setUpTOBLane(math.LegacyMustNewDecFromStr("0.5"), map[sdk.Tx]bool{
			bidTx:     true,
			bundle[0]: true,
			bundle[1]: true,
		})

		proposal := s.createProposal(map[string]uint64{defaultLane.Name(): 2, mevLane.Name(): 3}, bidTx, bundle[0], bundle[1], normalTx, normalTx2)

		proposalHandler := s.setUpProposalHandlers([]block.Lane{mevLane, defaultLane}).ProcessProposalHandler()
		resp := proposalHandler(s.ctx, cometabci.RequestProcessProposal{Txs: proposal, Height: 2})
		s.Require().NotNil(resp)
		s.Require().Equal(cometabci.ResponseProcessProposal{Status: cometabci.ResponseProcessProposal_REJECT}, resp)
	})

	s.Run("can process a invalid proposal where a lane consumes too much gas", func() {
		s.setBlockParams(1000, 10000000)

		bidTx, _, err := testutils.CreateAuctionTx(
			s.encodingConfig.TxConfig,
			s.accounts[0],
			sdk.NewCoin(s.gasTokenDenom, math.NewInt(1000000)),
			0,
			1,
			s.accounts[0:0],
			10000000000, // This should consume too much gas for the lane
		)
		s.Require().NoError(err)

		normalTx, err := testutils.CreateRandomTx(
			s.encodingConfig.TxConfig,
			s.accounts[1],
			0,
			1,
			0,
			1,
			sdk.NewCoin(s.gasTokenDenom, math.NewInt(3000000)),
		)
		s.Require().NoError(err)

		normalTx2, err := testutils.CreateRandomTx(
			s.encodingConfig.TxConfig,
			s.accounts[2],
			0,
			1,
			0,
			1,
			sdk.NewCoin(s.gasTokenDenom, math.NewInt(2000000)),
		)
		s.Require().NoError(err)

		// Set up the default lane
		defaultLane := s.setUpStandardLane(math.LegacyMustNewDecFromStr("0"), nil)

		// Set up the TOB lane
		mevLane := s.setUpTOBLane(math.LegacyMustNewDecFromStr("0.1"), nil)

		proposal := s.createProposal(map[string]uint64{defaultLane.Name(): 2, mevLane.Name(): 1}, bidTx, normalTx, normalTx2)

		proposalHandler := s.setUpProposalHandlers([]block.Lane{mevLane, defaultLane}).ProcessProposalHandler()
		resp := proposalHandler(s.ctx, cometabci.RequestProcessProposal{Txs: proposal, Height: 2})
		s.Require().NotNil(resp)
		s.Require().Equal(cometabci.ResponseProcessProposal{Status: cometabci.ResponseProcessProposal_REJECT}, resp)
	})

	s.Run("can process a invalid proposal where a lane consumes too much block space", func() {
		bidTx, _, err := testutils.CreateAuctionTx(
			s.encodingConfig.TxConfig,
			s.accounts[0],
			sdk.NewCoin(s.gasTokenDenom, math.NewInt(1000000)),
			0,
			1,
			s.accounts[0:0],
			1,
		)
		s.Require().NoError(err)

		normalTx, err := testutils.CreateRandomTx(
			s.encodingConfig.TxConfig,
			s.accounts[1],
			0,
			1,
			0,
			1,
			sdk.NewCoin(s.gasTokenDenom, math.NewInt(3000000)),
		)
		s.Require().NoError(err)

		normalTx2, err := testutils.CreateRandomTx(
			s.encodingConfig.TxConfig,
			s.accounts[2],
			0,
			1,
			0,
			1,
			sdk.NewCoin(s.gasTokenDenom, math.NewInt(2000000)),
		)
		s.Require().NoError(err)

		proposalTxs := s.getTxBytes(bidTx, normalTx, normalTx2)

		s.setBlockParams(1000, int64(len(proposalTxs[0])+len(proposalTxs[1])+len(proposalTxs[2])-1))

		// Set up the default lane
		defaultLane := s.setUpStandardLane(math.LegacyMustNewDecFromStr("0.5"), map[sdk.Tx]bool{
			normalTx:  true,
			normalTx2: true,
		})

		// Set up the TOB lane
		mevLane := s.setUpTOBLane(math.LegacyMustNewDecFromStr("0.5"), map[sdk.Tx]bool{
			bidTx: true,
		})

		proposal := s.createProposal(map[string]uint64{defaultLane.Name(): 2, mevLane.Name(): 1}, bidTx, normalTx, normalTx2)

		proposalHandler := s.setUpProposalHandlers([]block.Lane{mevLane, defaultLane}).ProcessProposalHandler()
		resp := proposalHandler(s.ctx, cometabci.RequestProcessProposal{Txs: proposal, Height: 2})
		s.Require().NotNil(resp)
		s.Require().Equal(cometabci.ResponseProcessProposal{Status: cometabci.ResponseProcessProposal_REJECT}, resp)
	})
}

func (s *ProposalsTestSuite) TestValidateBasic() {
	// Set up the default lane with no transactions
	mevlane := s.setUpTOBLane(math.LegacyMustNewDecFromStr("0.25"), nil)
	freelane := s.setUpFreeLane(math.LegacyMustNewDecFromStr("0.25"), nil)
	defaultLane := s.setUpStandardLane(math.LegacyMustNewDecFromStr("0.0"), nil)

	proposalHandlers := s.setUpProposalHandlers([]block.Lane{
		mevlane,
		freelane,
		defaultLane,
	})

	s.Run("can validate an empty proposal", func() {
		info := s.createProposalInfoBytes(0, 0, 0, 0, nil)
		proposal := [][]byte{info}

		_, partialProposals, err := proposalHandlers.ExtractLanes(proposal)
		s.Require().NoError(err)
		s.Require().Equal(3, len(partialProposals))

		for _, partialProposal := range partialProposals {
			s.Require().Equal(0, len(partialProposal))
		}
	})

	s.Run("should invalidate proposal with mismatch in transactions and proposal info", func() {
		info := s.createProposalInfoBytes(0, 0, 0, 0, nil)
		proposal := [][]byte{info, {0x01, 0x02, 0x03}}

		_, _, err := proposalHandlers.ExtractLanes(proposal)
		s.Require().Error(err)
	})

	s.Run("should invalidate proposal without info", func() {
		proposal := [][]byte{{0x01, 0x02, 0x03}}

		_, _, err := proposalHandlers.ExtractLanes(proposal)
		s.Require().Error(err)
	})

	s.Run("should invalidate completely empty proposal", func() {
		proposal := [][]byte{}

		_, _, err := proposalHandlers.ExtractLanes(proposal)
		s.Require().Error(err)
	})

	s.Run("should invalidate proposal with mismatch txs count with proposal info", func() {
		info := s.createProposalInfoBytes(0, 0, 0, 0, nil)
		proposal := [][]byte{info, {0x01, 0x02, 0x03}, {0x01, 0x02, 0x03}}

		_, _, err := proposalHandlers.ExtractLanes(proposal)
		s.Require().Error(err)
	})

	s.Run("can validate a proposal with a single tx", func() {
		tx, err := testutils.CreateRandomTx(
			s.encodingConfig.TxConfig,
			s.accounts[0],
			0,
			0,
			0,
			1,
		)
		s.Require().NoError(err)
		proposal := s.getTxBytes(tx)

		size, limit := s.getTxInfos(tx)
		maxSize, maxLimit := proposals.GetBlockLimits(s.ctx)
		info := s.createProposalInfoBytes(
			maxLimit,
			limit,
			maxSize,
			size,
			map[string]uint64{
				defaultLane.Name(): 1,
			},
		)

		proposal = append([][]byte{info}, proposal...)

		_, partialProposals, err := proposalHandlers.ExtractLanes(proposal)
		s.Require().NoError(err)

		s.Require().Equal(3, len(partialProposals))
		s.Require().Equal(0, len(partialProposals[0]))
		s.Require().Equal(0, len(partialProposals[1]))
		s.Require().Equal(1, len(partialProposals[2]))
		s.Require().Equal(proposal[1], partialProposals[2][0])
	})

	s.Run("can validate a proposal with multiple txs from single lane", func() {
		tx, err := testutils.CreateRandomTx(
			s.encodingConfig.TxConfig,
			s.accounts[0],
			0,
			0,
			0,
			1,
		)
		s.Require().NoError(err)

		tx2, err := testutils.CreateRandomTx(
			s.encodingConfig.TxConfig,
			s.accounts[1],
			0,
			0,
			0,
			1,
		)
		s.Require().NoError(err)

		proposal := s.getTxBytes(tx, tx2)

		size, limit := s.getTxInfos(tx, tx2)
		maxSize, maxLimit := proposals.GetBlockLimits(s.ctx)
		info := s.createProposalInfoBytes(
			maxLimit,
			limit,
			maxSize,
			size,
			map[string]uint64{
				defaultLane.Name(): 2,
			},
		)

		proposal = append([][]byte{info}, proposal...)

		_, partialProposals, err := proposalHandlers.ExtractLanes(proposal)
		s.Require().NoError(err)

		s.Require().Equal(3, len(partialProposals))
		s.Require().Equal(0, len(partialProposals[0]))
		s.Require().Equal(0, len(partialProposals[1]))
		s.Require().Equal(2, len(partialProposals[2]))
		s.Require().Equal(proposal[1], partialProposals[2][0])
		s.Require().Equal(proposal[2], partialProposals[2][1])
	})

	s.Run("can validate a proposal with 1 tx from each lane", func() {
		tx, err := testutils.CreateRandomTx(
			s.encodingConfig.TxConfig,
			s.accounts[0],
			0,
			0,
			0,
			1,
		)
		s.Require().NoError(err)

		tx2, err := testutils.CreateRandomTx(
			s.encodingConfig.TxConfig,
			s.accounts[1],
			0,
			0,
			0,
			1,
		)
		s.Require().NoError(err)

		tx3, err := testutils.CreateRandomTx(
			s.encodingConfig.TxConfig,
			s.accounts[2],
			0,
			0,
			0,
			1,
		)
		s.Require().NoError(err)

		proposal := s.getTxBytes(tx, tx2, tx3)

		size, limit := s.getTxInfos(tx, tx2, tx3)
		maxSize, maxLimit := proposals.GetBlockLimits(s.ctx)

		info := s.createProposalInfoBytes(
			maxLimit,
			limit,
			maxSize,
			size,
			map[string]uint64{
				defaultLane.Name(): 1,
				mevlane.Name():     1,
				freelane.Name():    1,
			},
		)

		proposal = append([][]byte{info}, proposal...)

		_, partialProposals, err := proposalHandlers.ExtractLanes(proposal)
		s.Require().NoError(err)

		s.Require().Equal(3, len(partialProposals))
		s.Require().Equal(1, len(partialProposals[0]))
		s.Require().Equal(proposal[1], partialProposals[0][0])

		s.Require().Equal(1, len(partialProposals[1]))
		s.Require().Equal(proposal[2], partialProposals[1][0])

		s.Require().Equal(1, len(partialProposals[2]))
		s.Require().Equal(proposal[3], partialProposals[2][0])
	})
}
