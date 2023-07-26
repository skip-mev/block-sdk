package keeper_test

import (
	"math/rand"
	"time"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	testutils "github.com/skip-mev/pob/testutils"
	"github.com/skip-mev/pob/x/builder/keeper"
	"github.com/skip-mev/pob/x/builder/types"
)

func (suite *KeeperTestSuite) TestValidateBidInfo() {
	var (
		// Tx building variables
		accounts = []testutils.Account{} // tracks the order of signers in the bundle
		balance  = sdk.NewCoin("stake", math.NewInt(10000))
		bid      = sdk.NewCoin("stake", math.NewInt(1000))

		// Auction params
		maxBundleSize          uint32 = 10
		reserveFee                    = sdk.NewCoin("stake", math.NewInt(1000))
		minBidIncrement               = sdk.NewCoin("stake", math.NewInt(1000))
		escrowAddress                 = sdk.AccAddress([]byte("escrow"))
		frontRunningProtection        = true

		// mempool variables
		highestBid = sdk.Coin{}
	)

	rnd := rand.New(rand.NewSource(time.Now().Unix()))
	bidder := testutils.RandomAccounts(rnd, 1)[0]

	cases := []struct {
		name     string
		malleate func()
		pass     bool
	}{
		{
			"insufficient bid amount",
			func() {
				bid = sdk.Coin{}
			},
			false,
		},
		{
			"insufficient balance",
			func() {
				bid = sdk.NewCoin("stake", math.NewInt(1000))
				balance = sdk.NewCoin("stake", math.NewInt(100))
			},
			false,
		},
		{
			"too many transactions in the bundle",
			func() {
				// reset the balance and bid to their original values
				bid = sdk.NewCoin("stake", math.NewInt(1000))
				balance = sdk.NewCoin("stake", math.NewInt(10000))
				accounts = testutils.RandomAccounts(rnd, int(maxBundleSize+1))
			},
			false,
		},
		{
			"frontrunning bundle",
			func() {
				randomAccount := testutils.RandomAccounts(rnd, 1)[0]
				accounts = []testutils.Account{bidder, randomAccount}
			},
			false,
		},
		{
			"sandwiching bundle",
			func() {
				randomAccount := testutils.RandomAccounts(rnd, 1)[0]
				accounts = []testutils.Account{bidder, randomAccount, bidder}
			},
			false,
		},
		{
			"valid bundle",
			func() {
				randomAccount := testutils.RandomAccounts(rnd, 1)[0]
				accounts = []testutils.Account{randomAccount, randomAccount, bidder, bidder, bidder}
			},
			true,
		},
		{
			"valid bundle with only bidder txs",
			func() {
				accounts = []testutils.Account{bidder, bidder, bidder, bidder}
			},
			true,
		},
		{
			"valid bundle with only random txs from single same user",
			func() {
				randomAccount := testutils.RandomAccounts(rnd, 1)[0]
				accounts = []testutils.Account{randomAccount, randomAccount, randomAccount, randomAccount}
			},
			true,
		},
		{
			"invalid bundle with random accounts",
			func() {
				accounts = testutils.RandomAccounts(rnd, 2)
			},
			false,
		},
		{
			"disabled front-running protection",
			func() {
				accounts = testutils.RandomAccounts(rnd, 10)
				frontRunningProtection = false
			},
			true,
		},
		{
			"invalid bundle that does not outbid the highest bid",
			func() {
				accounts = []testutils.Account{bidder, bidder, bidder}
				highestBid = sdk.NewCoin("stake", math.NewInt(500))
				bid = sdk.NewCoin("stake", math.NewInt(500))
			},
			false,
		},
		{
			"valid bundle that outbids the highest bid",
			func() {
				highestBid = sdk.NewCoin("stake", math.NewInt(500))
				bid = sdk.NewCoin("stake", math.NewInt(1500))
			},
			true,
		},
		{
			"attempting to bid with a different denom",
			func() {
				highestBid = sdk.NewCoin("stake", math.NewInt(500))
				bid = sdk.NewCoin("stake2", math.NewInt(1500))
			},
			false,
		},
		{
			"min bid increment is different from bid denom", // THIS SHOULD NEVER HAPPEN
			func() {
				highestBid = sdk.NewCoin("stake", math.NewInt(500))
				bid = sdk.NewCoin("stake", math.NewInt(1500))
				minBidIncrement = sdk.NewCoin("stake2", math.NewInt(1000))
			},
			false,
		},
	}

	for _, tc := range cases {
		suite.Run(tc.name, func() {
			suite.SetupTest() // reset

			tc.malleate()

			// Set up the new builder keeper with mocks customized for this test case
			suite.bankKeeper.EXPECT().GetBalance(suite.ctx, bidder.Address, minBidIncrement.Denom).Return(balance).AnyTimes()
			suite.bankKeeper.EXPECT().SendCoins(suite.ctx, bidder.Address, escrowAddress, reserveFee).Return(nil).AnyTimes()

			suite.builderKeeper = keeper.NewKeeper(
				suite.encCfg.Codec,
				suite.key,
				suite.accountKeeper,
				suite.bankKeeper,
				suite.distrKeeper,
				suite.stakingKeeper,
				suite.authorityAccount.String(),
			)
			params := types.Params{
				MaxBundleSize:          maxBundleSize,
				ReserveFee:             reserveFee,
				EscrowAccountAddress:   escrowAddress,
				FrontRunningProtection: frontRunningProtection,
				MinBidIncrement:        minBidIncrement,
			}
			suite.builderKeeper.SetParams(suite.ctx, params)

			// Create the bundle of transactions ordered by accounts
			bundle := make([][]byte, 0)
			for _, acc := range accounts {
				tx, err := testutils.CreateRandomTx(suite.encCfg.TxConfig, acc, 0, 1, 100)
				suite.Require().NoError(err)

				txBz, err := suite.encCfg.TxConfig.TxEncoder()(tx)
				suite.Require().NoError(err)
				bundle = append(bundle, txBz)
			}

			signers := make([]map[string]struct{}, len(accounts))
			for index, acc := range accounts {
				txSigners := map[string]struct{}{
					acc.Address.String(): {},
				}

				signers[index] = txSigners
			}

			bidInfo := &types.BidInfo{
				Bidder:       bidder.Address,
				Bid:          bid,
				Transactions: bundle,
				Signers:      signers,
			}

			err := suite.builderKeeper.ValidateBidInfo(suite.ctx, highestBid, bidInfo)
			if tc.pass {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestValidateBundle() {
	// TODO: Update this to be multi-dimensional to test multi-sig
	// https://github.com/skip-mev/pob/issues/14
	var accounts []testutils.Account // tracks the order of signers in the bundle

	rng := rand.New(rand.NewSource(time.Now().Unix()))
	bidder := testutils.RandomAccounts(rng, 1)[0]

	cases := []struct {
		name     string
		malleate func()
		pass     bool
	}{
		{
			"valid empty bundle",
			func() {
				accounts = make([]testutils.Account, 0)
			},
			true,
		},
		{
			"valid single tx bundle",
			func() {
				accounts = []testutils.Account{bidder}
			},
			true,
		},
		{
			"valid multi-tx bundle by same account",
			func() {
				accounts = []testutils.Account{bidder, bidder, bidder, bidder}
			},
			true,
		},
		{
			"valid single-tx bundle by a different account",
			func() {
				randomAccount := testutils.RandomAccounts(rng, 1)[0]
				accounts = []testutils.Account{randomAccount}
			},
			true,
		},
		{
			"valid multi-tx bundle by a different accounts",
			func() {
				randomAccount := testutils.RandomAccounts(rng, 1)[0]
				accounts = []testutils.Account{randomAccount, bidder}
			},
			true,
		},
		{
			"invalid frontrunning bundle",
			func() {
				randomAccount := testutils.RandomAccounts(rng, 1)[0]
				accounts = []testutils.Account{bidder, randomAccount}
			},
			false,
		},
		{
			"invalid sandwiching bundle",
			func() {
				randomAccount := testutils.RandomAccounts(rng, 1)[0]
				accounts = []testutils.Account{bidder, randomAccount, bidder}
			},
			false,
		},
		{
			"invalid multi account bundle",
			func() {
				accounts = testutils.RandomAccounts(rng, 3)
			},
			false,
		},
		{
			"invalid multi account bundle without bidder",
			func() {
				randomAccount1 := testutils.RandomAccounts(rng, 1)[0]
				randomAccount2 := testutils.RandomAccounts(rng, 1)[0]
				accounts = []testutils.Account{randomAccount1, randomAccount2}
			},
			false,
		},
	}

	for _, tc := range cases {
		suite.Run(tc.name, func() {
			suite.SetupTest() // reset

			// Malleate the test case
			tc.malleate()

			signers := make([]map[string]struct{}, len(accounts))
			for index, acc := range accounts {
				txSigners := map[string]struct{}{
					acc.Address.String(): {},
				}

				signers[index] = txSigners
			}

			// Validate the bundle
			err := suite.builderKeeper.ValidateAuctionBundle(bidder.Address, signers)
			if tc.pass {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}
