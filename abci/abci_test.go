package abci_test

import (
	"context"
	"math/rand"
	"testing"

	"cosmossdk.io/log"
	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	cometabci "github.com/cometbft/cometbft/abci/types"
	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/suite"

	"github.com/skip-mev/block-sdk/v2/abci"
	"github.com/skip-mev/block-sdk/v2/block"
	"github.com/skip-mev/block-sdk/v2/lanes/free"
	testutils "github.com/skip-mev/block-sdk/v2/testutils"
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

		maxTxBytes := s.ctx.ConsensusParams().Block.MaxBytes
		resp, err := proposalHandler(s.ctx, &cometabci.RequestPrepareProposal{Height: 2, MaxTxBytes: maxTxBytes})
		s.Require().NoError(err)
		s.Require().NotNil(resp)
		s.Require().Equal(0, len(resp.Txs))
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

		maxTxBytes := s.ctx.ConsensusParams().Block.MaxBytes
		resp, err := proposalHandler(s.ctx, &cometabci.RequestPrepareProposal{Height: 2, MaxTxBytes: maxTxBytes})
		s.Require().NotNil(resp)
		s.Require().NoError(err)

		proposal := s.getTxBytes(tx)
		s.Require().Equal(1, len(resp.Txs))
		s.Require().Equal(proposal, resp.Txs)
	})

	s.Run("can build a proposal with multiple txs from the default lane", func() {
		// Create a random transaction that will be inserted into the default lane
		tx1, err := testutils.CreateRandomTx(
			s.encodingConfig.TxConfig,
			s.accounts[0],
			1,
			1,
			0,
			1,
			sdk.NewCoin(s.gasTokenDenom, math.NewInt(1000000)),
		)
		s.Require().NoError(err)

		// Create a second random transaction that will be inserted into the default lane
		tx2, err := testutils.CreateRandomTx(
			s.encodingConfig.TxConfig,
			s.accounts[0],
			0,
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

		maxTxBytes := s.ctx.ConsensusParams().Block.MaxBytes
		resp, err := proposalHandler(s.ctx, &cometabci.RequestPrepareProposal{Height: 2, MaxTxBytes: maxTxBytes})
		s.Require().NotNil(resp)
		s.Require().NoError(err)

		proposal := s.getTxBytes(tx2, tx1)
		s.Require().Equal(2, len(resp.Txs))
		s.Require().Equal(proposal, resp.Txs)
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

		maxTxBytes := s.ctx.ConsensusParams().Block.MaxBytes
		resp, err := proposalHandler(s.ctx, &cometabci.RequestPrepareProposal{Height: 2, MaxTxBytes: maxTxBytes})
		s.Require().NotNil(resp)
		s.Require().NoError(err)

		proposal := s.getTxBytes(tx1)
		s.Require().Equal(1, len(resp.Txs))
		s.Require().Equal(proposal, resp.Txs)
	})

	s.Run("can build a proposal an empty proposal with multiple lanes", func() {
		mevLane := s.setUpTOBLane(math.LegacyMustNewDecFromStr("0.5"), nil)
		defaultLane := s.setUpStandardLane(math.LegacyMustNewDecFromStr("0.5"), nil)

		proposalHandler := s.setUpProposalHandlers([]block.Lane{mevLane, defaultLane}).PrepareProposalHandler()

		maxTxBytes := s.ctx.ConsensusParams().Block.MaxBytes
		resp, err := proposalHandler(s.ctx, &cometabci.RequestPrepareProposal{Height: 2, MaxTxBytes: maxTxBytes})
		s.Require().NoError(err)
		s.Require().NotNil(resp)
		s.Require().Equal(0, len(resp.Txs))
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

		maxTxBytes := s.ctx.ConsensusParams().Block.MaxBytes
		resp, err := proposalHandler(s.ctx, &cometabci.RequestPrepareProposal{Height: 2, MaxTxBytes: maxTxBytes})
		s.Require().NoError(err)
		s.Require().NotNil(resp)

		proposal := s.getTxBytes(tx, bundleTxs[0])
		s.Require().Equal(2, len(resp.Txs))
		s.Require().Equal(proposal, resp.Txs)
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

		maxTxBytes := s.ctx.ConsensusParams().Block.MaxBytes
		resp, err := proposalHandler(s.ctx, &cometabci.RequestPrepareProposal{Height: 2, MaxTxBytes: maxTxBytes})
		s.Require().NoError(err)
		s.Require().NotNil(resp)

		proposal := s.getTxBytes(tx, bundleTxs[0])
		s.Require().Equal(2, len(resp.Txs))
		s.Require().Equal(proposal, resp.Txs)
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

		maxTxBytes := s.ctx.ConsensusParams().Block.MaxBytes
		resp, err := proposalHandler(s.ctx, &cometabci.RequestPrepareProposal{Height: 2, MaxTxBytes: maxTxBytes})
		s.Require().NoError(err)
		s.Require().NotNil(resp)

		proposal := s.getTxBytes(bundleTxs[0])
		s.Require().Equal(1, len(resp.Txs))
		s.Require().Equal(proposal, resp.Txs)
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

		maxTxBytes := s.ctx.ConsensusParams().Block.MaxBytes
		resp, err := proposalHandler(s.ctx, &cometabci.RequestPrepareProposal{Height: 2, MaxTxBytes: maxTxBytes})
		s.Require().NoError(err)
		s.Require().NotNil(resp)

		s.Require().Equal(1, len(resp.Txs))
		s.Require().Equal(proposal, resp.Txs)
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

		maxTxBytes := s.ctx.ConsensusParams().Block.MaxBytes
		resp, err := proposalHandler(s.ctx, &cometabci.RequestPrepareProposal{Height: 2, MaxTxBytes: maxTxBytes})
		s.Require().NoError(err)
		s.Require().NotNil(resp)

		s.Require().Equal(7, len(resp.Txs))
		s.Require().Equal(proposal, resp.Txs)
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
		maxTxBytes := s.ctx.ConsensusParams().Block.MaxBytes
		resp, err := proposalHandler(s.ctx, &cometabci.RequestPrepareProposal{Height: 2, MaxTxBytes: maxTxBytes})
		s.Require().NoError(err)
		s.Require().NotNil(resp)

		s.Require().Equal(1, len(resp.Txs))
		s.Require().Equal(proposal[2:], resp.Txs)
	})
}

func (s *ProposalsTestSuite) TestPrepareProposalEdgeCases() {
	s.Run("can build a proposal if a lane panics first", func() {
		panicLane := s.setUpPanicLane("panik1", math.LegacyMustNewDecFromStr("0.25"))

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

		lanes := []block.Lane{
			panicLane,
			defaultLane,
		}

		mempool, err := block.NewLanedMempool(
			log.NewNopLogger(),
			lanes,
		)
		s.Require().NoError(err)

		proposalHandler := abci.New(
			log.NewNopLogger(),
			s.encodingConfig.TxConfig.TxDecoder(),
			s.encodingConfig.TxConfig.TxEncoder(),
			mempool,
			true,
		).PrepareProposalHandler()

		maxTxBytes := s.ctx.ConsensusParams().Block.MaxBytes
		resp, err := proposalHandler(s.ctx, &cometabci.RequestPrepareProposal{Height: 2, MaxTxBytes: maxTxBytes})
		s.Require().NoError(err)
		s.Require().NotNil(resp)

		proposal := s.getTxBytes(tx)
		s.Require().Equal(1, len(resp.Txs))
		s.Require().Equal(proposal, resp.Txs)
	})

	s.Run("can build a proposal if second lane panics", func() {
		panicLane := s.setUpPanicLane("panik1", math.LegacyMustNewDecFromStr("0.25"))

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

		lanes := []block.Lane{
			defaultLane,
			panicLane,
		}

		mempool, err := block.NewLanedMempool(
			log.NewNopLogger(),
			lanes,
		)
		s.Require().NoError(err)

		proposalHandler := abci.New(
			log.NewNopLogger(),
			s.encodingConfig.TxConfig.TxDecoder(),
			s.encodingConfig.TxConfig.TxEncoder(),
			mempool,
			true,
		).PrepareProposalHandler()

		maxTxBytes := s.ctx.ConsensusParams().Block.MaxBytes
		resp, err := proposalHandler(s.ctx, &cometabci.RequestPrepareProposal{Height: 2, MaxTxBytes: maxTxBytes})
		s.Require().NoError(err)
		s.Require().NotNil(resp)

		proposal := s.getTxBytes(tx)
		s.Require().Equal(1, len(resp.Txs))
		s.Require().Equal(proposal, resp.Txs)
	})

	s.Run("can build a proposal if multiple consecutive lanes panic", func() {
		panicLane := s.setUpPanicLane("panik1", math.LegacyMustNewDecFromStr("0.25"))
		panicLane2 := s.setUpPanicLane("panik2", math.LegacyMustNewDecFromStr("0.25"))

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

		lanes := []block.Lane{
			panicLane,
			panicLane2,
			defaultLane,
		}

		mempool, err := block.NewLanedMempool(
			log.NewNopLogger(),
			lanes,
		)
		s.Require().NoError(err)

		proposalHandler := abci.New(
			log.NewNopLogger(),
			s.encodingConfig.TxConfig.TxDecoder(),
			s.encodingConfig.TxConfig.TxEncoder(),
			mempool,
			true,
		).PrepareProposalHandler()

		maxTxBytes := s.ctx.ConsensusParams().Block.MaxBytes
		resp, err := proposalHandler(s.ctx, &cometabci.RequestPrepareProposal{Height: 2, MaxTxBytes: maxTxBytes})
		s.Require().NoError(err)
		s.Require().NotNil(resp)

		proposal := s.getTxBytes(tx)
		s.Require().Equal(1, len(resp.Txs))
		s.Require().Equal(proposal, resp.Txs)
	})

	s.Run("can build a proposal if the last few lanes panic", func() {
		panicLane := s.setUpPanicLane("panik1", math.LegacyMustNewDecFromStr("0.25"))
		panicLane2 := s.setUpPanicLane("panik2", math.LegacyMustNewDecFromStr("0.25"))

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

		lanes := []block.Lane{
			defaultLane,
			panicLane,
			panicLane2,
		}

		mempool, err := block.NewLanedMempool(
			log.NewNopLogger(),
			lanes,
		)
		s.Require().NoError(err)

		proposalHandler := abci.New(
			log.NewNopLogger(),
			s.encodingConfig.TxConfig.TxDecoder(),
			s.encodingConfig.TxConfig.TxEncoder(),
			mempool,
			true,
		).PrepareProposalHandler()

		maxTxBytes := s.ctx.ConsensusParams().Block.MaxBytes
		resp, err := proposalHandler(s.ctx, &cometabci.RequestPrepareProposal{Height: 2, MaxTxBytes: maxTxBytes})
		s.Require().NoError(err)
		s.Require().NotNil(resp)

		proposal := s.getTxBytes(tx)
		s.Require().Equal(1, len(resp.Txs))
		s.Require().Equal(proposal, resp.Txs)
	})
}

func (s *ProposalsTestSuite) TestProcessProposal() {
	s.Run("can process a valid empty proposal", func() {
		mevLane := s.setUpTOBLane(math.LegacyMustNewDecFromStr("0.25"), map[sdk.Tx]bool{})
		freeLane := s.setUpFreeLane(math.LegacyMustNewDecFromStr("0.25"), map[sdk.Tx]bool{})
		defaultLane := s.setUpStandardLane(math.LegacyMustNewDecFromStr("0.0"), map[sdk.Tx]bool{})

		proposalHandler := s.setUpProposalHandlers([]block.Lane{mevLane, freeLane, defaultLane}).ProcessProposalHandler()
		var proposal [][]byte

		resp, err := proposalHandler(s.ctx, &cometabci.RequestProcessProposal{Txs: proposal, Height: 2})
		s.Require().NoError(err)
		s.Require().NotNil(resp)
		s.Require().Equal(&cometabci.ResponseProcessProposal{Status: cometabci.ResponseProcessProposal_ACCEPT}, resp)
	})

	s.Run("can accept proposal where txs are broadcasted with different sequence numbers", func() {
		tx1, err := testutils.CreateRandomTx(
			s.encodingConfig.TxConfig,
			s.accounts[0],
			0,
			0,
			0,
			1,
			sdk.NewCoin(s.gasTokenDenom, math.NewInt(1)),
		)
		s.Require().NoError(err)

		tx2, err := testutils.CreateRandomTx(
			s.encodingConfig.TxConfig,
			s.accounts[0],
			1,
			0,
			0,
			1,
			sdk.NewCoin(s.gasTokenDenom, math.NewInt(2)),
		)
		s.Require().NoError(err)

		defaultLane := s.setUpStandardLane(math.LegacyMustNewDecFromStr("0.0"), map[sdk.Tx]bool{
			tx1: true,
			tx2: true,
		})

		s.Require().NoError(defaultLane.Insert(sdk.Context{}, tx1))
		s.Require().NoError(defaultLane.Insert(sdk.Context{}, tx2))

		var txs [][]sdk.Tx
		for iterator := defaultLane.Select(context.Background(), nil); iterator != nil; iterator = iterator.Next() {
			txs = append(txs, []sdk.Tx{iterator.Tx()})
		}

		s.Require().Equal(2, len(txs))

		proposal := s.createProposal(tx1, tx2)

		proposalHandler := s.setUpProposalHandlers([]block.Lane{defaultLane}).ProcessProposalHandler()
		resp, err := proposalHandler(s.ctx, &cometabci.RequestProcessProposal{Txs: proposal, Height: 2})
		s.Require().NoError(err)
		s.Require().NotNil(resp)
		s.Require().Equal(&cometabci.ResponseProcessProposal{Status: cometabci.ResponseProcessProposal_ACCEPT}, resp)
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

		proposal := s.createProposal(tx)

		proposalHandler := s.setUpProposalHandlers([]block.Lane{mevLane, defaultLane}).ProcessProposalHandler()
		resp, err := proposalHandler(s.ctx, &cometabci.RequestProcessProposal{Txs: proposal, Height: 2})
		s.Require().NoError(err)
		s.Require().NotNil(resp)
		s.Require().Equal(&cometabci.ResponseProcessProposal{Status: cometabci.ResponseProcessProposal_ACCEPT}, resp)
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

		proposal := s.createProposal(bidTx, bundleTxs[0], freeTx, tx)

		proposalHandler := s.setUpProposalHandlers([]block.Lane{mevLane, freeLane, defaultLane}).ProcessProposalHandler()
		resp, err := proposalHandler(s.ctx, &cometabci.RequestProcessProposal{Txs: proposal, Height: 2})
		s.Require().NoError(err)
		s.Require().NotNil(resp)
		s.Require().Equal(&cometabci.ResponseProcessProposal{Status: cometabci.ResponseProcessProposal_ACCEPT}, resp)
	})

	s.Run("can reject a proposal with txs from multiple lanes", func() {
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

		proposal := s.createProposal(bidTx, bundleTxs[0], tx, freeTx) // tx and freeTx are out of order

		proposalHandler := s.setUpProposalHandlers([]block.Lane{mevLane, freeLane, defaultLane}).ProcessProposalHandler()
		resp, err := proposalHandler(s.ctx, &cometabci.RequestProcessProposal{Txs: proposal, Height: 2})
		s.Require().Error(err)
		s.Require().NotNil(resp)
		s.Require().Equal(&cometabci.ResponseProcessProposal{Status: cometabci.ResponseProcessProposal_REJECT}, resp)
	})

	s.Run("can reject a proposal with txs from multiple lanes (mev is mixed up)", func() {
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

		proposal := s.createProposal(freeTx, tx, bidTx, bundleTxs[0])

		proposalHandler := s.setUpProposalHandlers([]block.Lane{mevLane, freeLane, defaultLane}).ProcessProposalHandler()
		resp, err := proposalHandler(s.ctx, &cometabci.RequestProcessProposal{Txs: proposal, Height: 2})
		s.Require().Error(err)
		s.Require().NotNil(resp)
		s.Require().Equal(&cometabci.ResponseProcessProposal{Status: cometabci.ResponseProcessProposal_REJECT}, resp)
	})

	s.Run("rejects a proposal with bad txs", func() {
		mevLane := s.setUpTOBLane(math.LegacyMustNewDecFromStr("0.25"), map[sdk.Tx]bool{})
		freeLane := s.setUpFreeLane(math.LegacyMustNewDecFromStr("0.25"), map[sdk.Tx]bool{})
		defaultLane := s.setUpStandardLane(math.LegacyMustNewDecFromStr("0.0"), map[sdk.Tx]bool{})

		proposalHandler := s.setUpProposalHandlers([]block.Lane{mevLane, freeLane, defaultLane}).ProcessProposalHandler()
		proposal := [][]byte{{0x01, 0x02, 0x03}}

		resp, err := proposalHandler(s.ctx, &cometabci.RequestProcessProposal{Txs: proposal, Height: 2})
		s.Require().Error(err)
		s.Require().Equal(&cometabci.ResponseProcessProposal{Status: cometabci.ResponseProcessProposal_REJECT}, resp)
	})

	s.Run("rejects a proposal when a lane panics", func() {
		mevLane := s.setUpTOBLane(math.LegacyMustNewDecFromStr("0.25"), map[sdk.Tx]bool{})
		panicLane := s.setUpPanicLane("default", math.LegacyMustNewDecFromStr("0.0"))

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
		proposal := [][]byte{txbz}

		resp, err := proposalHandler(s.ctx, &cometabci.RequestProcessProposal{Txs: proposal, Height: 2})
		s.Require().Error(err)
		s.Require().Equal(&cometabci.ResponseProcessProposal{Status: cometabci.ResponseProcessProposal_REJECT}, resp)
	})

	s.Run("can process a invalid proposal (default lane out of order)", func() {
		// Create a random transaction that will be inserted into the default lane
		tx1, err := testutils.CreateRandomTx(
			s.encodingConfig.TxConfig,
			s.accounts[2],
			0,
			1,
			0,
			1,
			sdk.NewCoin(s.gasTokenDenom, math.NewInt(2000000)),
		)
		s.Require().NoError(err)

		// Create a random transaction that will be inserted into the default lane
		tx2, err := testutils.CreateRandomTx(
			s.encodingConfig.TxConfig,
			s.accounts[2],
			1,
			1,
			0,
			1,
			sdk.NewCoin(s.gasTokenDenom, math.NewInt(2000000)),
		)
		s.Require().NoError(err)

		// Mev lane
		mevLane := s.setUpTOBLane(math.LegacyMustNewDecFromStr("0.3"), map[sdk.Tx]bool{})

		// Set up the default lane
		defaultLane := s.setUpStandardLane(math.LegacyMustNewDecFromStr("0.0"), map[sdk.Tx]bool{tx2: true, tx1: true})

		proposal := s.createProposal(tx2, tx1)

		proposalHandler := s.setUpProposalHandlers([]block.Lane{mevLane, defaultLane}).ProcessProposalHandler()
		resp, err := proposalHandler(s.ctx, &cometabci.RequestProcessProposal{Txs: proposal, Height: 2})
		s.Require().NotNil(resp)
		s.Require().Error(err)
		s.Require().Equal(&cometabci.ResponseProcessProposal{Status: cometabci.ResponseProcessProposal_REJECT}, resp)
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

		proposal := s.createProposal(bidTx, bundle[0], bundle[1], normalTx, normalTx2)

		proposalHandler := s.setUpProposalHandlers([]block.Lane{mevLane, defaultLane}).ProcessProposalHandler()
		resp, err := proposalHandler(s.ctx, &cometabci.RequestProcessProposal{Txs: proposal, Height: 2})
		s.Require().NotNil(resp)
		s.Require().Error(err)
		s.Require().Equal(&cometabci.ResponseProcessProposal{Status: cometabci.ResponseProcessProposal_REJECT}, resp)
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

		proposal := s.createProposal(bidTx, normalTx, normalTx2)

		proposalHandler := s.setUpProposalHandlers([]block.Lane{mevLane, defaultLane}).ProcessProposalHandler()
		resp, err := proposalHandler(s.ctx, &cometabci.RequestProcessProposal{Txs: proposal, Height: 2})
		s.Require().NotNil(resp)
		s.Require().Error(err)
		s.Require().Equal(&cometabci.ResponseProcessProposal{Status: cometabci.ResponseProcessProposal_REJECT}, resp)
		s.Require().Equal(&cometabci.ResponseProcessProposal{Status: cometabci.ResponseProcessProposal_REJECT}, resp)
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

		proposal := s.createProposal(bidTx, normalTx, normalTx2)

		proposalHandler := s.setUpProposalHandlers([]block.Lane{mevLane, defaultLane}).ProcessProposalHandler()
		resp, err := proposalHandler(s.ctx, &cometabci.RequestProcessProposal{Txs: proposal, Height: 2})
		s.Require().NotNil(resp)
		s.Require().Error(err)
		s.Require().Equal(&cometabci.ResponseProcessProposal{Status: cometabci.ResponseProcessProposal_REJECT}, resp)
	})

	s.Run("rejects a proposal where there are transactions remaining that have been unverified", func() {
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

		freeTx, err := testutils.CreateFreeTx(
			s.encodingConfig.TxConfig,
			s.accounts[2],
			0,
			1,
			"test",
			sdk.NewCoin(s.gasTokenDenom, math.NewInt(2000000)),
			sdk.NewCoin(s.gasTokenDenom, math.NewInt(2000000)),
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

		// Set up the top of block lane
		mevLane := s.setUpTOBLane(math.LegacyMustNewDecFromStr("0.5"), map[sdk.Tx]bool{
			bidTx: true,
		})

		// Set up the default lane
		freeLane := s.setUpCustomMatchHandlerLane(
			math.LegacyMustNewDecFromStr("0.0"),
			map[sdk.Tx]bool{
				freeTx: true,
			},
			free.DefaultMatchHandler(),
			"default",
		)

		proposal := s.createProposal(bidTx, freeTx, normalTx)

		proposalHandler := s.setUpProposalHandlers([]block.Lane{mevLane, freeLane}).ProcessProposalHandler()
		resp, err := proposalHandler(s.ctx, &cometabci.RequestProcessProposal{Txs: proposal, Height: 2})
		s.Require().NotNil(resp)
		s.Require().Error(err)
		s.Require().Equal(&cometabci.ResponseProcessProposal{Status: cometabci.ResponseProcessProposal_REJECT}, resp)
	})
}

func (s *ProposalsTestSuite) TestPrepareProcessParity() {
	// Define a large enough block size and gas limit to ensure that the proposal is accepted
	s.setBlockParams(1000000000000, 1000000000000)

	// Create a random transaction that will be inserted into the default/free lane
	numTxsPerAccount := uint64(25)
	numAccounts := 25
	accounts := testutils.RandomAccounts(s.random, numAccounts)

	feeDenoms := []string{
		s.gasTokenDenom,
		"eth",
		"btc",
		"usdt",
		"usdc",
	}

	// Create a bunch of transactions to insert into the default lane
	txsToInsert := []sdk.Tx{}
	validationMap := make(map[sdk.Tx]bool)
	for nonce := uint64(0); nonce < numTxsPerAccount*uint64(numAccounts); nonce++ {
		fees := []sdk.Coin{}
		// choose a random set of fee denoms
		perm := rand.Perm(len(feeDenoms))
		for i := 0; i < 1+rand.Intn(len(feeDenoms)-1); i++ {
			fees = append(fees, sdk.NewCoin(feeDenoms[perm[i]], math.NewInt(int64(rand.Intn(100000)))))
		}

		// choose a random set of accounts
		perm = rand.Perm(len(accounts))
		signers := []testutils.Account{}
		for i := 0; i < 1+rand.Intn(len(accounts)-1); i++ {
			signers = append(signers, accounts[perm[i]])
		}

		// create a random fee amount
		tx, err := testutils.CreateRandomTxMultipleSigners(
			s.encodingConfig.TxConfig,
			signers,
			nonce,
			1,
			0,
			1,
			fees...,
		)
		s.Require().NoError(err)

		txsToInsert = append(txsToInsert, tx)
		validationMap[tx] = true
	}

	// Set up the default lane with the transactions
	defaultLane := s.setUpStandardLane(math.LegacyZeroDec(), validationMap)
	for _, tx := range txsToInsert {
		s.Require().NoError(defaultLane.Insert(s.ctx, tx))
	}

	// Create a bunch of transactions to insert into the free lane
	var freeTxsToInsert []sdk.Tx
	freeValidationMap := make(map[sdk.Tx]bool)
	for _, account := range accounts {
		for nonce := uint64(0); nonce < numTxsPerAccount; nonce++ {
			// create a random fee amount
			feeAmount := math.NewInt(int64(rand.Intn(100000)))
			tx, err := testutils.CreateFreeTx(
				s.encodingConfig.TxConfig,
				account,
				nonce,
				1,
				"test",
				sdk.NewCoin(s.gasTokenDenom, feeAmount),
				sdk.NewCoin(s.gasTokenDenom, feeAmount),
			)
			s.Require().NoError(err)

			freeTxsToInsert = append(freeTxsToInsert, tx)
			freeValidationMap[tx] = true
		}
	}

	freelane := s.setUpFreeLane(math.LegacyMustNewDecFromStr("0.25"), freeValidationMap)
	for _, tx := range freeTxsToInsert {
		s.Require().NoError(freelane.Insert(s.ctx, tx))
	}

	// Retrieve the transactions from the default lane in the same way the prepare function would.
	var retrievedTxs []sdk.Tx
	for iterator := defaultLane.Select(context.Background(), nil); iterator != nil; iterator = iterator.Next() {
		retrievedTxs = append(retrievedTxs, iterator.Tx())
	}
	s.Require().Equal(len(txsToInsert), len(retrievedTxs))

	// Retrieve the transactions from the free lane in the same way the prepare function would.
	var freeRetrievedTxs []sdk.Tx
	for iterator := freelane.Select(context.Background(), nil); iterator != nil; iterator = iterator.Next() {
		freeRetrievedTxs = append(freeRetrievedTxs, iterator.Tx())
	}
	s.Require().Equal(len(freeTxsToInsert), len(freeRetrievedTxs))

	numTxsPerLane := numTxsPerAccount * uint64(numAccounts)
	s.Require().Equal(numTxsPerLane, uint64(len(retrievedTxs)))
	s.Require().Equal(numTxsPerLane, uint64(len(freeRetrievedTxs)))

	// Create a proposal with the retrieved transactions
	// Set up the default lane with no transactions
	proposalHandler := s.setUpProposalHandlers([]block.Lane{freelane, defaultLane}).PrepareProposalHandler()

	maxTxBytes := s.ctx.ConsensusParams().Block.MaxBytes
	resp, err := proposalHandler(s.ctx, &cometabci.RequestPrepareProposal{Height: 2, MaxTxBytes: maxTxBytes})
	s.Require().NoError(err)
	s.Require().NotNil(resp)

	// Ensure the transactions are in the correct order for the free lane
	for i := 0; i < int(numTxsPerLane); i++ {
		bz, err := s.encodingConfig.TxConfig.TxEncoder()(freeRetrievedTxs[i])
		s.Require().NoError(err)
		s.Require().Equal(bz, resp.Txs[i])
	}

	// Ensure the transactions are in the correct order for the default lane
	for i := 0; i < int(numTxsPerLane); i++ {
		bz, err := s.encodingConfig.TxConfig.TxEncoder()(retrievedTxs[i])
		s.Require().NoError(err)
		s.Require().Equal(bz, resp.Txs[i+int(numTxsPerLane)])
	}

	proposal := s.createProposal(
		append(freeRetrievedTxs, retrievedTxs...)...,
	)

	// Validate the proposal
	processHandler := s.setUpProposalHandlers([]block.Lane{freelane, defaultLane}).ProcessProposalHandler()
	processResp, err := processHandler(s.ctx, &cometabci.RequestProcessProposal{Txs: proposal, Height: 2})
	s.Require().NotNil(processResp)
	s.Require().NoError(err)
}

func (s *ProposalsTestSuite) TestIterateMempoolAndProcessProposalParity() {
	// Define a large enough block size and gas limit to ensure that the proposal is accepted
	s.setBlockParams(1000000000000, 1000000000000)

	// Create a random transaction that will be inserted into the default/free lane
	numTxsPerAccount := uint64(25)
	numAccounts := 25
	accounts := testutils.RandomAccounts(s.random, numAccounts)

	// Create a bunch of transactions to insert into the default lane
	var txsToInsert []sdk.Tx
	validationMap := make(map[sdk.Tx]bool)
	for _, account := range accounts {
		for nonce := uint64(0); nonce < numTxsPerAccount; nonce++ {
			// create a random fee amount
			feeAmount := math.NewInt(int64(rand.Intn(100000)))
			tx, err := testutils.CreateRandomTx(
				s.encodingConfig.TxConfig,
				account,
				nonce,
				1,
				0,
				1,
				sdk.NewCoin(s.gasTokenDenom, feeAmount),
			)
			s.Require().NoError(err)

			txsToInsert = append(txsToInsert, tx)
			validationMap[tx] = true
		}
	}

	// Set up the default lane with the transactions
	defaultLane := s.setUpStandardLane(math.LegacyZeroDec(), validationMap)
	for _, tx := range txsToInsert {
		s.Require().NoError(defaultLane.Insert(s.ctx, tx))
	}

	// Create a bunch of transactions to insert into the free lane
	var freeTxsToInsert []sdk.Tx
	freeValidationMap := make(map[sdk.Tx]bool)
	for _, account := range accounts {
		for nonce := uint64(0); nonce < numTxsPerAccount; nonce++ {
			// create a random fee amount
			feeAmount := math.NewInt(int64(rand.Intn(100000)))
			tx, err := testutils.CreateFreeTx(
				s.encodingConfig.TxConfig,
				account,
				nonce,
				1,
				"test",
				sdk.NewCoin(s.gasTokenDenom, feeAmount),
				sdk.NewCoin(s.gasTokenDenom, feeAmount),
			)
			s.Require().NoError(err)

			freeTxsToInsert = append(freeTxsToInsert, tx)
			freeValidationMap[tx] = true
		}
	}

	freelane := s.setUpFreeLane(math.LegacyMustNewDecFromStr("0.3"), freeValidationMap)
	for _, tx := range freeTxsToInsert {
		s.Require().NoError(freelane.Insert(s.ctx, tx))
	}

	// Retrieve the transactions from the default lane in the same way the prepare function would.
	var retrievedTxs []sdk.Tx
	for iterator := defaultLane.Select(context.Background(), nil); iterator != nil; iterator = iterator.Next() {
		retrievedTxs = append(retrievedTxs, iterator.Tx())
	}
	s.Require().Equal(len(txsToInsert), len(retrievedTxs))

	// Retrieve the transactions from the free lane in the same way the prepare function would.
	var freeRetrievedTxs []sdk.Tx
	for iterator := freelane.Select(context.Background(), nil); iterator != nil; iterator = iterator.Next() {
		freeRetrievedTxs = append(freeRetrievedTxs, iterator.Tx())
	}
	s.Require().Equal(len(freeTxsToInsert), len(freeRetrievedTxs))

	// Create a proposal with the retrieved transactions
	numTxsPerLane := numTxsPerAccount * uint64(numAccounts)
	s.Require().Equal(numTxsPerLane, uint64(len(retrievedTxs)))
	s.Require().Equal(numTxsPerLane, uint64(len(freeRetrievedTxs)))

	proposal := s.createProposal(
		append(freeRetrievedTxs, retrievedTxs...)...,
	)

	// Validate the proposal
	proposalHandler := s.setUpProposalHandlers([]block.Lane{freelane, defaultLane}).ProcessProposalHandler()
	resp, err := proposalHandler(s.ctx, &cometabci.RequestProcessProposal{Txs: proposal, Height: 2})
	s.Require().NotNil(resp)
	s.Require().NoError(err)
}
