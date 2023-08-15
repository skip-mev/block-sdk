package abci_test

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/rand"
	"testing"

	"cosmossdk.io/log"
	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	cometabci "github.com/cometbft/cometbft/abci/types"
	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/skip-mev/pob/blockbuster"
	"github.com/skip-mev/pob/blockbuster/abci"
	"github.com/skip-mev/pob/blockbuster/lanes/auction"
	"github.com/skip-mev/pob/blockbuster/lanes/base"
	"github.com/skip-mev/pob/blockbuster/lanes/free"
	testutils "github.com/skip-mev/pob/testutils"
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

func TestBlockBusterTestSuite(t *testing.T) {
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
}

func (s *ProposalsTestSuite) TestPrepareProposal() {
	s.Run("can prepare a proposal with no transactions", func() {
		// Set up the default lane with no transactions
		defaultLane := s.setUpDefaultLane(math.LegacyMustNewDecFromStr("1"), nil)

		proposalHandler := s.setUpProposalHandlers([]blockbuster.Lane{defaultLane}).PrepareProposalHandler()

		resp, err := proposalHandler(s.ctx, &cometabci.RequestPrepareProposal{})
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
			sdk.NewCoin(s.gasTokenDenom, math.NewInt(1000000)),
		)
		s.Require().NoError(err)

		// Set up the default lane
		defaultLane := s.setUpDefaultLane(math.LegacyMustNewDecFromStr("1"), map[sdk.Tx]bool{tx: true})
		s.Require().NoError(defaultLane.Insert(sdk.Context{}, tx))

		proposalHandler := s.setUpProposalHandlers([]blockbuster.Lane{defaultLane}).PrepareProposalHandler()
		resp, err := proposalHandler(s.ctx, &cometabci.RequestPrepareProposal{MaxTxBytes: 10000000000})
		s.Require().NotNil(resp)
		s.Require().NoError(err)

		proposal := s.getTxBytes(tx)
		s.Require().Equal(1, len(resp.Txs))
		s.Require().Equal(proposal, resp.Txs)
	})

	s.Run("can build a proposal with multiple txs from the lane", func() {
		// Create a random transaction that will be inserted into the default lane
		tx1, err := testutils.CreateRandomTx(
			s.encodingConfig.TxConfig,
			s.accounts[0],
			0,
			1,
			0,
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
			sdk.NewCoin(s.gasTokenDenom, math.NewInt(200000000)),
		)
		s.Require().NoError(err)

		// Set up the default lane with both transactions passing
		defaultLane := s.setUpDefaultLane(math.LegacyMustNewDecFromStr("1"), map[sdk.Tx]bool{tx1: true, tx2: true})
		s.Require().NoError(defaultLane.Insert(sdk.Context{}, tx1))
		s.Require().NoError(defaultLane.Insert(sdk.Context{}, tx2))

		proposalHandler := s.setUpProposalHandlers([]blockbuster.Lane{defaultLane}).PrepareProposalHandler()
		resp, err := proposalHandler(s.ctx, &cometabci.RequestPrepareProposal{MaxTxBytes: 10000000000})
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
			sdk.NewCoin(s.gasTokenDenom, math.NewInt(200000000)),
		)
		s.Require().NoError(err)

		// Set up the default lane with both transactions passing
		defaultLane := s.setUpDefaultLane(math.LegacyMustNewDecFromStr("1"), map[sdk.Tx]bool{tx1: true, tx2: false})
		s.Require().NoError(defaultLane.Insert(sdk.Context{}, tx1))
		s.Require().NoError(defaultLane.Insert(sdk.Context{}, tx2))

		proposalHandler := s.setUpProposalHandlers([]blockbuster.Lane{defaultLane}).PrepareProposalHandler()
		resp, err := proposalHandler(s.ctx, &cometabci.RequestPrepareProposal{MaxTxBytes: 10000000000})
		s.Require().NotNil(resp)
		s.Require().NoError(err)

		proposal := s.getTxBytes(tx1)
		s.Require().Equal(1, len(resp.Txs))
		s.Require().Equal(proposal, resp.Txs)
	})

	s.Run("can build a proposal an empty proposal with multiple lanes", func() {
		tobLane := s.setUpTOBLane(math.LegacyMustNewDecFromStr("0.5"), nil)
		defaultLane := s.setUpDefaultLane(math.LegacyMustNewDecFromStr("0.5"), nil)

		proposalHandler := s.setUpProposalHandlers([]blockbuster.Lane{tobLane, defaultLane}).PrepareProposalHandler()

		resp, err := proposalHandler(s.ctx, &cometabci.RequestPrepareProposal{})
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
		)
		s.Require().NoError(err)

		// Set up the TOB lane with the bid tx and the bundled tx
		tobLane := s.setUpTOBLane(math.LegacyMustNewDecFromStr("0.5"), map[sdk.Tx]bool{
			tx:           true,
			bundleTxs[0]: true,
		})
		s.Require().NoError(tobLane.Insert(sdk.Context{}, tx))

		defaultLane := s.setUpDefaultLane(math.LegacyMustNewDecFromStr("0.5"), nil)

		proposalHandler := s.setUpProposalHandlers([]blockbuster.Lane{tobLane, defaultLane}).PrepareProposalHandler()

		resp, err := proposalHandler(s.ctx, &cometabci.RequestPrepareProposal{MaxTxBytes: 10000000000})
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
		)
		s.Require().NoError(err)

		// Set up the TOB lane with the bid tx and the bundled tx
		tobLane := s.setUpTOBLane(math.LegacyMustNewDecFromStr("0.5"), map[sdk.Tx]bool{
			tx:           true,
			bundleTxs[0]: true,
		})
		s.Require().NoError(tobLane.Insert(sdk.Context{}, tx))

		// Set up the default lane with the bid tx and the bundled tx
		defaultLane := s.setUpDefaultLane(math.LegacyMustNewDecFromStr("0.5"), map[sdk.Tx]bool{
			tx:           true,
			bundleTxs[0]: true,
		})
		s.Require().NoError(defaultLane.Insert(sdk.Context{}, tx))
		s.Require().NoError(defaultLane.Insert(sdk.Context{}, bundleTxs[0]))

		proposalHandler := s.setUpProposalHandlers([]blockbuster.Lane{tobLane, defaultLane}).PrepareProposalHandler()

		resp, err := proposalHandler(s.ctx, &cometabci.RequestPrepareProposal{MaxTxBytes: 10000000000})
		s.Require().NoError(err)
		s.Require().NotNil(resp)

		proposal := s.getTxBytes(tx, bundleTxs[0])
		s.Require().Equal(2, len(resp.Txs))
		s.Require().Equal(proposal, resp.Txs)
	})

	s.Run("can build a proposal where first lane has too large of a tx and second lane has a valid tx", func() {
		// Create a bid tx that includes a single bundled tx
		tx, bundleTxs, err := testutils.CreateAuctionTx(
			s.encodingConfig.TxConfig,
			s.accounts[0],
			sdk.NewCoin(s.gasTokenDenom, math.NewInt(1000000)),
			0,
			0,
			s.accounts[0:1],
		)
		s.Require().NoError(err)

		// Set up the TOB lane with the bid tx and the bundled tx
		tobLane := s.setUpTOBLane(math.LegacyMustNewDecFromStr("0.5"), map[sdk.Tx]bool{
			tx:           false,
			bundleTxs[0]: true,
		})
		s.Require().NoError(tobLane.Insert(sdk.Context{}, tx))

		// Set up the default lane with the bid tx and the bundled tx
		defaultLane := s.setUpDefaultLane(math.LegacyMustNewDecFromStr("0.5"), map[sdk.Tx]bool{
			// Even though this passes it should not include it in the proposal because it is in the ignore list
			tx:           true,
			bundleTxs[0]: true,
		})
		s.Require().NoError(defaultLane.Insert(sdk.Context{}, tx))
		s.Require().NoError(defaultLane.Insert(sdk.Context{}, bundleTxs[0]))

		proposalHandler := s.setUpProposalHandlers([]blockbuster.Lane{tobLane, defaultLane}).PrepareProposalHandler()

		resp, err := proposalHandler(s.ctx, &cometabci.RequestPrepareProposal{MaxTxBytes: 10000000000})
		s.Require().NoError(err)
		s.Require().NotNil(resp)

		proposal := s.getTxBytes(bundleTxs[0])
		s.Require().Equal(1, len(resp.Txs))
		s.Require().Equal(proposal, resp.Txs)
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
		)
		s.Require().NoError(err)

		// Set up the TOB lane with the bid tx and the bundled tx
		tobLane := s.setUpTOBLane(math.LegacyMustNewDecFromStr("0.5"), map[sdk.Tx]bool{
			tx:           true,
			bundleTxs[0]: true,
		})
		s.Require().NoError(tobLane.Insert(sdk.Context{}, tx))

		// Set up the default lane with the bid tx and the bundled tx
		defaultLane := s.setUpDefaultLane(math.LegacyMustNewDecFromStr("0.0"), map[sdk.Tx]bool{
			// Even though this passes it should not include it in the proposal because it is in the ignore list
			tx:           true,
			bundleTxs[0]: true,
		})
		s.Require().NoError(defaultLane.Insert(sdk.Context{}, tx))
		s.Require().NoError(defaultLane.Insert(sdk.Context{}, bundleTxs[0]))

		proposalHandler := s.setUpProposalHandlers([]blockbuster.Lane{tobLane, defaultLane}).PrepareProposalHandler()
		proposal := s.getTxBytes(tx, bundleTxs[0])
		size := int64(len(proposal[0]) - 1)

		resp, err := proposalHandler(s.ctx, &cometabci.RequestPrepareProposal{MaxTxBytes: size})
		s.Require().NoError(err)
		s.Require().NotNil(resp)

		s.Require().Equal(1, len(resp.Txs))
		s.Require().Equal(proposal[1:], resp.Txs)
	})

	s.Run("can build a proposal with single tx from middle lane", func() {
		tobLane := s.setUpTOBLane(math.LegacyMustNewDecFromStr("0.25"), nil)

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

		defaultLane := s.setUpDefaultLane(math.LegacyMustNewDecFromStr("0.0"), map[sdk.Tx]bool{
			freeTx: true,
		})
		s.Require().NoError(defaultLane.Insert(sdk.Context{}, freeTx))

		freeLane := s.setUpFreeLane(math.LegacyMustNewDecFromStr("0.25"), map[sdk.Tx]bool{
			freeTx: true,
		})
		s.Require().NoError(freeLane.Insert(sdk.Context{}, freeTx))

		proposalHandler := s.setUpProposalHandlers([]blockbuster.Lane{tobLane, freeLane, defaultLane}).PrepareProposalHandler()

		proposal := s.getTxBytes(freeTx)

		resp, err := proposalHandler(s.ctx, &cometabci.RequestPrepareProposal{MaxTxBytes: 1000000})
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
		)
		s.Require().NoError(err)

		// Create a normal tx
		normalTx, err := testutils.CreateRandomTx(
			s.encodingConfig.TxConfig,
			s.accounts[1],
			0,
			0,
			0,
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

		tobLane := s.setUpTOBLane(math.LegacyMustNewDecFromStr("0.25"), map[sdk.Tx]bool{
			tx:           true,
			bundleTxs[0]: true,
			bundleTxs[1]: true,
			bundleTxs[2]: true,
			bundleTxs[3]: true,
		})
		tobLane.Insert(sdk.Context{}, tx)

		defaultLane := s.setUpDefaultLane(math.LegacyMustNewDecFromStr("0.0"), map[sdk.Tx]bool{
			normalTx: true,
		})
		defaultLane.Insert(sdk.Context{}, normalTx)

		freeLane := s.setUpFreeLane(math.LegacyMustNewDecFromStr("0.25"), map[sdk.Tx]bool{
			freeTx: true,
		})
		freeLane.Insert(sdk.Context{}, freeTx)

		proposalHandler := s.setUpProposalHandlers([]blockbuster.Lane{tobLane, freeLane, defaultLane}).PrepareProposalHandler()
		proposal := s.getTxBytes(tx, bundleTxs[0], bundleTxs[1], bundleTxs[2], bundleTxs[3], freeTx, normalTx)

		resp, err := proposalHandler(s.ctx, &cometabci.RequestPrepareProposal{MaxTxBytes: 1000000000})
		s.Require().NoError(err)
		s.Require().NotNil(resp)

		s.Require().Equal(7, len(resp.Txs))
		s.Require().Equal(proposal, resp.Txs)
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
			sdk.NewCoin(s.gasTokenDenom, math.NewInt(1000000)),
		)
		s.Require().NoError(err)

		defaultLane := s.setUpDefaultLane(math.LegacyMustNewDecFromStr("0.0"), map[sdk.Tx]bool{
			tx: true,
		})
		s.Require().NoError(defaultLane.Insert(sdk.Context{}, tx))

		proposalHandler := abci.NewProposalHandler(
			log.NewTestLogger(s.T()),
			s.encodingConfig.TxConfig.TxDecoder(),
			[]blockbuster.Lane{panicLane, defaultLane},
		).PrepareProposalHandler()

		resp, err := proposalHandler(s.ctx, &cometabci.RequestPrepareProposal{MaxTxBytes: 1000000})
		s.Require().NoError(err)
		s.Require().NotNil(resp)

		proposal := s.getTxBytes(tx)
		s.Require().Equal(1, len(resp.Txs))
		s.Require().Equal(proposal, resp.Txs)
	})

	s.Run("can build a proposal if second lane panics", func() {
		panicLane := s.setUpPanicLane(math.LegacyMustNewDecFromStr("0.25"))

		tx, err := testutils.CreateRandomTx(
			s.encodingConfig.TxConfig,
			s.accounts[0],
			0,
			0,
			0,
			sdk.NewCoin(s.gasTokenDenom, math.NewInt(1000000)),
		)
		s.Require().NoError(err)

		defaultLane := s.setUpDefaultLane(math.LegacyMustNewDecFromStr("0.0"), map[sdk.Tx]bool{
			tx: true,
		})
		s.Require().NoError(defaultLane.Insert(sdk.Context{}, tx))

		proposalHandler := abci.NewProposalHandler(
			log.NewTestLogger(s.T()),
			s.encodingConfig.TxConfig.TxDecoder(),
			[]blockbuster.Lane{defaultLane, panicLane},
		).PrepareProposalHandler()

		resp, err := proposalHandler(s.ctx, &cometabci.RequestPrepareProposal{MaxTxBytes: 1000000})
		s.Require().NoError(err)
		s.Require().NotNil(resp)

		proposal := s.getTxBytes(tx)
		s.Require().Equal(1, len(resp.Txs))
		s.Require().Equal(proposal, resp.Txs)
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
			sdk.NewCoin(s.gasTokenDenom, math.NewInt(1000000)),
		)
		s.Require().NoError(err)

		defaultLane := s.setUpDefaultLane(math.LegacyMustNewDecFromStr("0.0"), map[sdk.Tx]bool{
			tx: true,
		})
		s.Require().NoError(defaultLane.Insert(sdk.Context{}, tx))

		proposalHandler := abci.NewProposalHandler(
			log.NewTestLogger(s.T()),
			s.encodingConfig.TxConfig.TxDecoder(),
			[]blockbuster.Lane{panicLane, panicLane2, defaultLane},
		).PrepareProposalHandler()

		resp, err := proposalHandler(s.ctx, &cometabci.RequestPrepareProposal{MaxTxBytes: 1000000})
		s.Require().NoError(err)
		s.Require().NotNil(resp)

		proposal := s.getTxBytes(tx)
		s.Require().Equal(1, len(resp.Txs))
		s.Require().Equal(proposal, resp.Txs)
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
			sdk.NewCoin(s.gasTokenDenom, math.NewInt(1000000)),
		)
		s.Require().NoError(err)

		defaultLane := s.setUpDefaultLane(math.LegacyMustNewDecFromStr("0.0"), map[sdk.Tx]bool{
			tx: true,
		})
		s.Require().NoError(defaultLane.Insert(sdk.Context{}, tx))

		proposalHandler := abci.NewProposalHandler(
			log.NewTestLogger(s.T()),
			s.encodingConfig.TxConfig.TxDecoder(),
			[]blockbuster.Lane{defaultLane, panicLane, panicLane2},
		).PrepareProposalHandler()

		resp, err := proposalHandler(s.ctx, &cometabci.RequestPrepareProposal{MaxTxBytes: 1000000})
		s.Require().NoError(err)
		s.Require().NotNil(resp)

		proposal := s.getTxBytes(tx)
		s.Require().Equal(1, len(resp.Txs))
		s.Require().Equal(proposal, resp.Txs)
	})
}

func (s *ProposalsTestSuite) TestProcessProposal() {
	s.Run("can process a valid empty proposal", func() {
		tobLane := s.setUpTOBLane(math.LegacyMustNewDecFromStr("0.25"), map[sdk.Tx]bool{})
		freeLane := s.setUpFreeLane(math.LegacyMustNewDecFromStr("0.25"), map[sdk.Tx]bool{})
		defaultLane := s.setUpDefaultLane(math.LegacyMustNewDecFromStr("0.0"), map[sdk.Tx]bool{})

		proposalHandler := s.setUpProposalHandlers([]blockbuster.Lane{tobLane, freeLane, defaultLane}).ProcessProposalHandler()

		resp, err := proposalHandler(s.ctx, &cometabci.RequestProcessProposal{Txs: nil})
		s.Require().NoError(err)
		s.Require().NotNil(resp)
		s.Require().Equal(&cometabci.ResponseProcessProposal{Status: cometabci.ResponseProcessProposal_ACCEPT}, resp)
	})

	s.Run("rejects a proposal with bad txs", func() {
		tobLane := s.setUpTOBLane(math.LegacyMustNewDecFromStr("0.25"), map[sdk.Tx]bool{})
		freeLane := s.setUpFreeLane(math.LegacyMustNewDecFromStr("0.25"), map[sdk.Tx]bool{})
		defaultLane := s.setUpDefaultLane(math.LegacyMustNewDecFromStr("0.0"), map[sdk.Tx]bool{})

		proposalHandler := s.setUpProposalHandlers([]blockbuster.Lane{tobLane, freeLane, defaultLane}).ProcessProposalHandler()

		resp, err := proposalHandler(s.ctx, &cometabci.RequestProcessProposal{Txs: [][]byte{{0x01, 0x02, 0x03}}})
		s.Require().Error(err)
		s.Require().Equal(&cometabci.ResponseProcessProposal{Status: cometabci.ResponseProcessProposal_REJECT}, resp)
	})

	s.Run("rejects a proposal when a lane panics", func() {
		tobLane := s.setUpTOBLane(math.LegacyMustNewDecFromStr("0.25"), map[sdk.Tx]bool{})
		panicLane := s.setUpPanicLane(math.LegacyMustNewDecFromStr("0.0"))

		txbz, err := testutils.CreateRandomTxBz(
			s.encodingConfig.TxConfig,
			s.accounts[0],
			0,
			0,
			0,
		)
		s.Require().NoError(err)

		proposalHandler := s.setUpProposalHandlers([]blockbuster.Lane{tobLane, panicLane}).ProcessProposalHandler()
		resp, err := proposalHandler(s.ctx, &cometabci.RequestProcessProposal{Txs: [][]byte{txbz}})
		s.Require().Error(err)
		s.Require().Equal(&cometabci.ResponseProcessProposal{Status: cometabci.ResponseProcessProposal_REJECT}, resp)
	})

	s.Run("can process a invalid proposal (out of order)", func() {
		// Create a random transaction that will be inserted into the default lane
		tx, err := testutils.CreateRandomTx(
			s.encodingConfig.TxConfig,
			s.accounts[0],
			0,
			1,
			0,
			sdk.NewCoin(s.gasTokenDenom, math.NewInt(1000000)),
		)
		s.Require().NoError(err)

		// Create a random transaction that will be inserted into the default lane
		tx2, err := testutils.CreateRandomTx(
			s.encodingConfig.TxConfig,
			s.accounts[2],
			0,
			1,
			0,
			sdk.NewCoin(s.gasTokenDenom, math.NewInt(2000000)),
		)
		s.Require().NoError(err)

		// Set up the default lane
		defaultLane := s.setUpDefaultLane(math.LegacyMustNewDecFromStr("1"), map[sdk.Tx]bool{tx: true})
		s.Require().NoError(defaultLane.Insert(sdk.Context{}, tx))

		proposalHandler := s.setUpProposalHandlers([]blockbuster.Lane{defaultLane}).ProcessProposalHandler()
		resp, err := proposalHandler(s.ctx, &cometabci.RequestProcessProposal{Txs: s.getTxBytes(tx, tx2)})
		s.Require().NotNil(resp)
		s.Require().Error(err)
	})

	s.Run("can process a invalid proposal where first lane is valid second is not", func() {
		bidTx, bundle, err := testutils.CreateAuctionTx(
			s.encodingConfig.TxConfig,
			s.accounts[0],
			sdk.NewCoin(s.gasTokenDenom, math.NewInt(1000000)),
			0,
			1,
			s.accounts[0:2],
		)
		s.Require().NoError(err)

		normalTx, err := testutils.CreateRandomTx(
			s.encodingConfig.TxConfig,
			s.accounts[1],
			0,
			1,
			0,
			sdk.NewCoin(s.gasTokenDenom, math.NewInt(2000000)),
		)
		s.Require().NoError(err)

		normalTx2, err := testutils.CreateRandomTx(
			s.encodingConfig.TxConfig,
			s.accounts[2],
			0,
			1,
			0,
			sdk.NewCoin(s.gasTokenDenom, math.NewInt(3000000)),
		)
		s.Require().NoError(err)

		// Set up the default lane
		defaultLane := s.setUpDefaultLane(math.LegacyMustNewDecFromStr("0.5"), nil)
		defaultLane.SetProcessLaneHandler(blockbuster.NoOpProcessLaneHandler())

		// Set up the TOB lane
		tobLane := s.setUpTOBLane(math.LegacyMustNewDecFromStr("0.5"), nil)
		tobLane.SetProcessLaneHandler(blockbuster.NoOpProcessLaneHandler())

		proposalHandler := s.setUpProposalHandlers([]blockbuster.Lane{tobLane, defaultLane}).ProcessProposalHandler()
		resp, err := proposalHandler(s.ctx, &cometabci.RequestProcessProposal{Txs: s.getTxBytes(bidTx, bundle[0], bundle[1], normalTx, normalTx2)})
		s.Require().NotNil(resp)
		s.Require().Error(err)
	})
}

func (s *ProposalsTestSuite) setUpAnteHandler(expectedExecution map[sdk.Tx]bool) sdk.AnteHandler {
	txCache := make(map[string]bool)
	for tx, pass := range expectedExecution {
		bz, err := s.encodingConfig.TxConfig.TxEncoder()(tx)
		s.Require().NoError(err)

		hash := sha256.Sum256(bz)
		hashStr := hex.EncodeToString(hash[:])
		txCache[hashStr] = pass
	}

	anteHandler := func(ctx sdk.Context, tx sdk.Tx, simulate bool) (newCtx sdk.Context, err error) {
		bz, err := s.encodingConfig.TxConfig.TxEncoder()(tx)
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

func (s *ProposalsTestSuite) setUpDefaultLane(maxBlockSpace math.LegacyDec, expectedExecution map[sdk.Tx]bool) *base.DefaultLane {
	cfg := blockbuster.LaneConfig{
		Logger:        log.NewTestLogger(s.T()),
		TxEncoder:     s.encodingConfig.TxConfig.TxEncoder(),
		TxDecoder:     s.encodingConfig.TxConfig.TxDecoder(),
		AnteHandler:   s.setUpAnteHandler(expectedExecution),
		MaxBlockSpace: maxBlockSpace,
	}

	return base.NewDefaultLane(cfg)
}

func (s *ProposalsTestSuite) setUpTOBLane(maxBlockSpace math.LegacyDec, expectedExecution map[sdk.Tx]bool) *auction.TOBLane {
	cfg := blockbuster.LaneConfig{
		Logger:        log.NewTestLogger(s.T()),
		TxEncoder:     s.encodingConfig.TxConfig.TxEncoder(),
		TxDecoder:     s.encodingConfig.TxConfig.TxDecoder(),
		AnteHandler:   s.setUpAnteHandler(expectedExecution),
		MaxBlockSpace: maxBlockSpace,
	}

	return auction.NewTOBLane(cfg, auction.NewDefaultAuctionFactory(cfg.TxDecoder))
}

func (s *ProposalsTestSuite) setUpFreeLane(maxBlockSpace math.LegacyDec, expectedExecution map[sdk.Tx]bool) *free.FreeLane {
	cfg := blockbuster.LaneConfig{
		Logger:        log.NewTestLogger(s.T()),
		TxEncoder:     s.encodingConfig.TxConfig.TxEncoder(),
		TxDecoder:     s.encodingConfig.TxConfig.TxDecoder(),
		AnteHandler:   s.setUpAnteHandler(expectedExecution),
		MaxBlockSpace: maxBlockSpace,
	}

	return free.NewFreeLane(cfg, blockbuster.DefaultTxPriority(), free.DefaultMatchHandler())
}

func (s *ProposalsTestSuite) setUpPanicLane(maxBlockSpace math.LegacyDec) *blockbuster.LaneConstructor {
	cfg := blockbuster.LaneConfig{
		Logger:        log.NewTestLogger(s.T()),
		TxEncoder:     s.encodingConfig.TxConfig.TxEncoder(),
		TxDecoder:     s.encodingConfig.TxConfig.TxDecoder(),
		MaxBlockSpace: maxBlockSpace,
	}

	lane := blockbuster.NewLaneConstructor(
		cfg,
		"panic",
		blockbuster.NewConstructorMempool[string](blockbuster.DefaultTxPriority(), cfg.TxEncoder, 0),
		blockbuster.DefaultMatchHandler(),
	)

	lane.SetPrepareLaneHandler(blockbuster.PanicPrepareLaneHandler())
	lane.SetProcessLaneHandler(blockbuster.PanicProcessLaneHandler())

	return lane
}

func (s *ProposalsTestSuite) setUpProposalHandlers(lanes []blockbuster.Lane) *abci.ProposalHandler {
	mempool := blockbuster.NewMempool(log.NewTestLogger(s.T()), true, lanes...)

	return abci.NewProposalHandler(
		log.NewTestLogger(s.T()),
		s.encodingConfig.TxConfig.TxDecoder(),
		mempool.Registry(),
	)
}

func (s *ProposalsTestSuite) getTxBytes(txs ...sdk.Tx) [][]byte {
	txBytes := make([][]byte, len(txs))
	for i, tx := range txs {
		bz, err := s.encodingConfig.TxConfig.TxEncoder()(tx)
		s.Require().NoError(err)

		txBytes[i] = bz
	}
	return txBytes
}
