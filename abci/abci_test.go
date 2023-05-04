package abci_test

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/cometbft/cometbft/libs/log"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/golang/mock/gomock"
	"github.com/skip-mev/pob/abci"
	"github.com/skip-mev/pob/mempool"
	testutils "github.com/skip-mev/pob/testutils"
	"github.com/skip-mev/pob/x/builder/ante"
	"github.com/skip-mev/pob/x/builder/keeper"
	buildertypes "github.com/skip-mev/pob/x/builder/types"
	"github.com/stretchr/testify/suite"
)

type ABCITestSuite struct {
	suite.Suite
	ctx sdk.Context

	// mempool setup
	mempool              *mempool.AuctionMempool
	logger               log.Logger
	encodingConfig       testutils.EncodingConfig
	proposalHandler      *abci.ProposalHandler
	voteExtensionHandler *abci.VoteExtensionHandler
	config               mempool.Config
	txs                  map[string]struct{}

	// auction bid setup
	auctionBidAmount sdk.Coin
	minBidIncrement  sdk.Coin

	// builder setup
	builderKeeper    keeper.Keeper
	bankKeeper       *testutils.MockBankKeeper
	accountKeeper    *testutils.MockAccountKeeper
	distrKeeper      *testutils.MockDistributionKeeper
	stakingKeeper    *testutils.MockStakingKeeper
	builderDecorator ante.BuilderDecorator
	key              *storetypes.KVStoreKey
	authorityAccount sdk.AccAddress

	// account set up
	accounts []testutils.Account
	balances sdk.Coins
	random   *rand.Rand
	nonces   map[string]uint64
}

func TestABCISuite(t *testing.T) {
	suite.Run(t, new(ABCITestSuite))
}

func (suite *ABCITestSuite) SetupTest() {
	// General config
	suite.encodingConfig = testutils.CreateTestEncodingConfig()
	suite.random = rand.New(rand.NewSource(time.Now().Unix()))
	suite.key = storetypes.NewKVStoreKey(buildertypes.StoreKey)
	testCtx := testutil.DefaultContextWithDB(suite.T(), suite.key, storetypes.NewTransientStoreKey("transient_test"))
	suite.ctx = testCtx.Ctx

	// Mempool set up
	suite.config = mempool.NewDefaultConfig(suite.encodingConfig.TxConfig.TxDecoder())
	suite.mempool = mempool.NewAuctionMempool(suite.encodingConfig.TxConfig.TxDecoder(), suite.encodingConfig.TxConfig.TxEncoder(), 0, suite.config)
	suite.txs = make(map[string]struct{})
	suite.auctionBidAmount = sdk.NewCoin("foo", sdk.NewInt(1000000000))
	suite.minBidIncrement = sdk.NewCoin("foo", sdk.NewInt(1000))

	// Mock keepers set up
	ctrl := gomock.NewController(suite.T())
	suite.accountKeeper = testutils.NewMockAccountKeeper(ctrl)
	suite.accountKeeper.EXPECT().GetModuleAddress(buildertypes.ModuleName).Return(sdk.AccAddress{}).AnyTimes()
	suite.bankKeeper = testutils.NewMockBankKeeper(ctrl)
	suite.distrKeeper = testutils.NewMockDistributionKeeper(ctrl)
	suite.stakingKeeper = testutils.NewMockStakingKeeper(ctrl)
	suite.authorityAccount = sdk.AccAddress([]byte("authority"))

	// Builder keeper / decorator set up
	suite.builderKeeper = keeper.NewKeeper(
		suite.encodingConfig.Codec,
		suite.key,
		suite.accountKeeper,
		suite.bankKeeper,
		suite.distrKeeper,
		suite.stakingKeeper,
		suite.authorityAccount.String(),
	)
	err := suite.builderKeeper.SetParams(suite.ctx, buildertypes.DefaultParams())
	suite.Require().NoError(err)
	suite.builderDecorator = ante.NewBuilderDecorator(suite.builderKeeper, suite.encodingConfig.TxConfig.TxDecoder(), suite.encodingConfig.TxConfig.TxEncoder(), suite.mempool)

	// Accounts set up
	suite.accounts = testutils.RandomAccounts(suite.random, 1)
	suite.balances = sdk.NewCoins(sdk.NewCoin("foo", sdk.NewInt(1000000000000000000)))
	suite.nonces = make(map[string]uint64)
	for _, acc := range suite.accounts {
		suite.nonces[acc.Address.String()] = 0
	}

	// Proposal handler set up
	suite.logger = log.NewNopLogger()
	suite.proposalHandler = abci.NewProposalHandler(suite.mempool, suite.logger, suite.anteHandler, suite.encodingConfig.TxConfig.TxEncoder(), suite.encodingConfig.TxConfig.TxDecoder())
	suite.voteExtensionHandler = abci.NewVoteExtensionHandler(suite.mempool, suite.encodingConfig.TxConfig.TxDecoder(), suite.encodingConfig.TxConfig.TxEncoder(), suite.anteHandler)
}

func (suite *ABCITestSuite) anteHandler(ctx sdk.Context, tx sdk.Tx, simulate bool) (sdk.Context, error) {
	signer := tx.GetMsgs()[0].GetSigners()[0]
	suite.bankKeeper.EXPECT().GetAllBalances(ctx, signer).AnyTimes().Return(suite.balances)

	next := func(ctx sdk.Context, tx sdk.Tx, simulate bool) (sdk.Context, error) {
		return ctx, nil
	}

	ctx, err := suite.builderDecorator.AnteHandle(ctx, tx, false, next)
	if err != nil {
		return ctx, err
	}

	bz, err := suite.encodingConfig.TxConfig.TxEncoder()(tx)
	if err != nil {
		return ctx, err
	}

	if !simulate {
		hash := sha256.Sum256(bz)
		txHash := hex.EncodeToString(hash[:])
		if _, ok := suite.txs[txHash]; ok {
			return ctx, fmt.Errorf("tx already in mempool")
		}
		suite.txs[txHash] = struct{}{}
	}

	return ctx, nil
}

func (suite *ABCITestSuite) createFilledMempool(numNormalTxs, numAuctionTxs, numBundledTxs int, insertRefTxs bool) int {
	suite.mempool = mempool.NewAuctionMempool(suite.encodingConfig.TxConfig.TxDecoder(), suite.encodingConfig.TxConfig.TxEncoder(), 0, suite.config)

	// Insert a bunch of normal transactions into the global mempool
	for i := 0; i < numNormalTxs; i++ {
		// randomly select an account to create the tx
		randomIndex := suite.random.Intn(len(suite.accounts))
		acc := suite.accounts[randomIndex]

		// create a few random msgs
		randomMsgs := testutils.CreateRandomMsgs(acc.Address, 3)

		nonce := suite.nonces[acc.Address.String()]
		randomTx, err := testutils.CreateTx(suite.encodingConfig.TxConfig, acc, nonce, 1000, randomMsgs)
		suite.Require().NoError(err)

		suite.nonces[acc.Address.String()]++
		priority := suite.random.Int63n(100) + 1
		suite.Require().NoError(suite.mempool.Insert(suite.ctx.WithPriority(priority), randomTx))
	}

	suite.Require().Equal(numNormalTxs, suite.mempool.CountTx())
	suite.Require().Equal(0, suite.mempool.CountAuctionTx())

	// Insert a bunch of auction transactions into the global mempool and auction mempool
	for i := 0; i < numAuctionTxs; i++ {
		// randomly select a bidder to create the tx
		randomIndex := suite.random.Intn(len(suite.accounts))
		acc := suite.accounts[randomIndex]

		// create a new auction bid msg with numBundledTxs bundled transactions
		nonce := suite.nonces[acc.Address.String()]
		bidMsg, err := testutils.CreateMsgAuctionBid(suite.encodingConfig.TxConfig, acc, suite.auctionBidAmount, nonce, numBundledTxs)
		suite.nonces[acc.Address.String()] += uint64(numBundledTxs)
		suite.Require().NoError(err)

		// create the auction tx
		nonce = suite.nonces[acc.Address.String()]
		auctionTx, err := testutils.CreateTx(suite.encodingConfig.TxConfig, acc, nonce, 1000, []sdk.Msg{bidMsg})
		suite.Require().NoError(err)

		// insert the auction tx into the global mempool
		priority := suite.random.Int63n(100) + 1
		suite.Require().NoError(suite.mempool.Insert(suite.ctx.WithPriority(priority), auctionTx))
		suite.nonces[acc.Address.String()]++

		if insertRefTxs {
			for _, refRawTx := range bidMsg.GetTransactions() {
				refTx, err := suite.encodingConfig.TxConfig.TxDecoder()(refRawTx)
				suite.Require().NoError(err)
				priority := suite.random.Int63n(100) + 1
				suite.Require().NoError(suite.mempool.Insert(suite.ctx.WithPriority(priority), refTx))
			}
		}

		// decrement the bid amount for the next auction tx
		suite.auctionBidAmount = suite.auctionBidAmount.Sub(suite.minBidIncrement)
	}

	numSeenGlobalTxs := 0
	for iterator := suite.mempool.Select(suite.ctx, nil); iterator != nil; iterator = iterator.Next() {
		numSeenGlobalTxs++
	}

	numSeenAuctionTxs := 0
	for iterator := suite.mempool.AuctionBidSelect(suite.ctx); iterator != nil; iterator = iterator.Next() {
		numSeenAuctionTxs++
	}

	var totalNumTxs int
	suite.Require().Equal(numAuctionTxs, suite.mempool.CountAuctionTx())
	if insertRefTxs {
		totalNumTxs = numNormalTxs + numAuctionTxs*(numBundledTxs)
		suite.Require().Equal(totalNumTxs, suite.mempool.CountTx())
		suite.Require().Equal(totalNumTxs, numSeenGlobalTxs)
	} else {
		totalNumTxs = numNormalTxs
		suite.Require().Equal(totalNumTxs, suite.mempool.CountTx())
		suite.Require().Equal(totalNumTxs, numSeenGlobalTxs)
	}

	suite.Require().Equal(numAuctionTxs, numSeenAuctionTxs)

	return totalNumTxs
}

func (suite *ABCITestSuite) exportMempool(exportRefTxs bool) [][]byte {
	txs := make([][]byte, 0)
	seenTxs := make(map[string]bool)

	auctionIterator := suite.mempool.AuctionBidSelect(suite.ctx)
	for ; auctionIterator != nil; auctionIterator = auctionIterator.Next() {
		auctionTx := auctionIterator.Tx()
		txBz, err := suite.encodingConfig.TxConfig.TxEncoder()(auctionTx)
		suite.Require().NoError(err)

		txs = append(txs, txBz)

		if exportRefTxs {
			for _, refRawTx := range auctionTx.GetMsgs()[0].(*buildertypes.MsgAuctionBid).GetTransactions() {
				txs = append(txs, refRawTx)
				seenTxs[string(refRawTx)] = true
			}
		}

		seenTxs[string(txBz)] = true
	}

	iterator := suite.mempool.Select(suite.ctx, nil)
	for ; iterator != nil; iterator = iterator.Next() {
		txBz, err := suite.encodingConfig.TxConfig.TxEncoder()(iterator.Tx())
		suite.Require().NoError(err)

		if !seenTxs[string(txBz)] {
			txs = append(txs, txBz)
		}
	}

	return txs
}
