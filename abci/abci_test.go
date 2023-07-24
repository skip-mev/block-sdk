package abci_test

import (
	"math/rand"
	"testing"
	"time"

	sdklogger "cosmossdk.io/log"
	comettypes "github.com/cometbft/cometbft/abci/types"
	"github.com/cometbft/cometbft/libs/log"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/golang/mock/gomock"
	"github.com/skip-mev/pob/abci"
	"github.com/skip-mev/pob/blockbuster"
	"github.com/skip-mev/pob/blockbuster/lanes/auction"
	"github.com/skip-mev/pob/blockbuster/lanes/base"
	testutils "github.com/skip-mev/pob/testutils"
	"github.com/skip-mev/pob/x/builder/ante"
	"github.com/skip-mev/pob/x/builder/keeper"
	buildertypes "github.com/skip-mev/pob/x/builder/types"
	"github.com/stretchr/testify/suite"
)

type ABCITestSuite struct {
	suite.Suite
	ctx sdk.Context

	// mempool and lane set up
	mempool  blockbuster.Mempool
	tobLane  *auction.TOBLane
	baseLane *base.DefaultLane

	logger               log.Logger
	encodingConfig       testutils.EncodingConfig
	proposalHandler      *abci.ProposalHandler
	voteExtensionHandler *abci.VoteExtensionHandler

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
	balance  sdk.Coin
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
	suite.ctx = testCtx.Ctx.WithBlockHeight(1)
	suite.logger = log.NewNopLogger()

	// Lanes configuration
	//
	// TOB lane set up
	config := blockbuster.BaseLaneConfig{
		Logger:        suite.logger,
		TxEncoder:     suite.encodingConfig.TxConfig.TxEncoder(),
		TxDecoder:     suite.encodingConfig.TxConfig.TxDecoder(),
		AnteHandler:   suite.anteHandler,
		MaxBlockSpace: sdk.ZeroDec(),
	}
	suite.tobLane = auction.NewTOBLane(
		config,
		0, // No bound on the number of transactions in the lane
		auction.NewDefaultAuctionFactory(suite.encodingConfig.TxConfig.TxDecoder()),
	)

	// Base lane set up
	suite.baseLane = base.NewDefaultLane(
		config,
	)

	// Mempool set up
	suite.mempool = blockbuster.NewMempool(
		suite.tobLane,
		suite.baseLane,
	)

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
	suite.builderDecorator = ante.NewBuilderDecorator(suite.builderKeeper, suite.encodingConfig.TxConfig.TxEncoder(), suite.tobLane, suite.mempool)

	// Accounts set up
	suite.accounts = testutils.RandomAccounts(suite.random, 10)
	suite.balance = sdk.NewCoin("foo", sdk.NewInt(1000000000000000000))
	suite.nonces = make(map[string]uint64)
	for _, acc := range suite.accounts {
		suite.nonces[acc.Address.String()] = 0
	}

	// Proposal handler set up
	suite.proposalHandler = abci.NewProposalHandler(
		[]blockbuster.Lane{suite.baseLane}, // only the base lane is used for proposal handling
		suite.tobLane,
		suite.logger,
		suite.encodingConfig.TxConfig.TxEncoder(),
		suite.encodingConfig.TxConfig.TxDecoder(),
		abci.NoOpValidateVoteExtensionsFn(),
	)
	suite.voteExtensionHandler = abci.NewVoteExtensionHandler(
		sdklogger.NewTestLogger(suite.T()),
		suite.tobLane,
		suite.encodingConfig.TxConfig.TxDecoder(),
		suite.encodingConfig.TxConfig.TxEncoder(),
	)
}

func (suite *ABCITestSuite) anteHandler(ctx sdk.Context, tx sdk.Tx, _ bool) (sdk.Context, error) {
	signer := tx.GetMsgs()[0].GetSigners()[0]
	suite.bankKeeper.EXPECT().GetBalance(ctx, signer, suite.balance.Denom).AnyTimes().Return(suite.balance)

	next := func(ctx sdk.Context, _ sdk.Tx, _ bool) (sdk.Context, error) {
		return ctx, nil
	}

	return suite.builderDecorator.AnteHandle(ctx, tx, false, next)
}

// fillBaseLane fills the base lane with numTxs transactions that are randomly created.
func (suite *ABCITestSuite) fillBaseLane(numTxs int) {
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
func (suite *ABCITestSuite) fillTOBLane(numTxs int, numBundledTxs int) {
	// Insert a bunch of auction transactions into the global mempool and auction mempool
	for i := 0; i < numTxs; i++ {
		// randomly select a bidder to create the tx
		randomIndex := suite.random.Intn(len(suite.accounts))
		acc := suite.accounts[randomIndex]

		// create a randomized auction transaction
		nonce := suite.nonces[acc.Address.String()]
		bidAmount := sdk.NewInt(int64(suite.random.Intn(1000) + 1))
		bid := sdk.NewCoin("foo", bidAmount)

		signers := []testutils.Account{}
		for j := 0; j < numBundledTxs; j++ {
			signers = append(signers, suite.accounts[0])
		}

		tx, err := testutils.CreateAuctionTxWithSigners(suite.encodingConfig.TxConfig, acc, bid, nonce, 1000, signers)
		suite.Require().NoError(err)

		// insert the auction tx into the global mempool
		suite.Require().NoError(suite.mempool.Insert(suite.ctx, tx))
		suite.nonces[acc.Address.String()]++
	}
}

func (suite *ABCITestSuite) createPrepareProposalRequest(maxBytes int64) comettypes.RequestPrepareProposal {
	voteExtensions := make([]comettypes.ExtendedVoteInfo, 0)

	auctionIterator := suite.tobLane.Select(suite.ctx, nil)
	for ; auctionIterator != nil; auctionIterator = auctionIterator.Next() {
		tx := auctionIterator.Tx()

		txBz, err := suite.encodingConfig.TxConfig.TxEncoder()(tx)
		suite.Require().NoError(err)

		voteExtensions = append(voteExtensions, comettypes.ExtendedVoteInfo{
			VoteExtension: txBz,
		})
	}

	return comettypes.RequestPrepareProposal{
		MaxTxBytes: maxBytes,
		LocalLastCommit: comettypes.ExtendedCommitInfo{
			Votes: voteExtensions,
		},
	}
}

func (suite *ABCITestSuite) createExtendedCommitInfoFromTxs(txs []sdk.Tx) comettypes.ExtendedCommitInfo {
	voteExtensions := make([][]byte, 0)
	for _, tx := range txs {
		bz, err := suite.encodingConfig.TxConfig.TxEncoder()(tx)
		suite.Require().NoError(err)

		voteExtensions = append(voteExtensions, bz)
	}

	return suite.createExtendedCommitInfo(voteExtensions)
}

func (suite *ABCITestSuite) createExtendedVoteInfo(voteExtensions [][]byte) []comettypes.ExtendedVoteInfo {
	commitInfo := make([]comettypes.ExtendedVoteInfo, 0)
	for _, voteExtension := range voteExtensions {
		info := comettypes.ExtendedVoteInfo{
			VoteExtension: voteExtension,
		}

		commitInfo = append(commitInfo, info)
	}

	return commitInfo
}

func (suite *ABCITestSuite) createExtendedCommitInfo(voteExtensions [][]byte) comettypes.ExtendedCommitInfo {
	commitInfo := comettypes.ExtendedCommitInfo{
		Votes: suite.createExtendedVoteInfo(voteExtensions),
	}

	return commitInfo
}

func (suite *ABCITestSuite) createExtendedCommitInfoFromTxBzs(txs [][]byte) []byte {
	voteExtensions := make([]comettypes.ExtendedVoteInfo, 0)

	for _, txBz := range txs {
		voteExtensions = append(voteExtensions, comettypes.ExtendedVoteInfo{
			VoteExtension: txBz,
		})
	}

	commitInfo := comettypes.ExtendedCommitInfo{
		Votes: voteExtensions,
	}

	commitInfoBz, err := commitInfo.Marshal()
	suite.Require().NoError(err)

	return commitInfoBz
}

func (suite *ABCITestSuite) createAuctionInfoFromTxBzs(txs [][]byte, numTxs uint64, maxTxBytes int64) []byte {
	auctionInfo := abci.AuctionInfo{
		ExtendedCommitInfo: suite.createExtendedCommitInfoFromTxBzs(txs),
		NumTxs:             numTxs,
		MaxTxBytes:         maxTxBytes,
	}

	auctionInfoBz, err := auctionInfo.Marshal()
	suite.Require().NoError(err)

	return auctionInfoBz
}

func (suite *ABCITestSuite) getAuctionBidInfoFromTxBz(txBz []byte) *buildertypes.BidInfo {
	tx, err := suite.encodingConfig.TxConfig.TxDecoder()(txBz)
	suite.Require().NoError(err)

	bidInfo, err := suite.tobLane.GetAuctionBidInfo(tx)
	suite.Require().NoError(err)

	return bidInfo
}
