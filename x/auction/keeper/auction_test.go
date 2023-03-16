package keeper_test

import (
	"math/rand"
	"time"

	"github.com/cosmos/cosmos-sdk/client"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	authsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/skip-mev/pob/x/auction/keeper"
	auctiontypes "github.com/skip-mev/pob/x/auction/types"
)

func (suite *IntegrationTestSuite) TestValidateAuctionMsg() {
	var (
		// Tx building variables
		accounts = []Account{} // tracks the order of signers in the bundle
		balance  = sdk.NewCoins(sdk.NewCoin("foo", sdk.NewInt(10000)))
		bid      = sdk.NewCoins(sdk.NewCoin("foo", sdk.NewInt(1000)))

		// Auction params
		maxBundleSize          uint32 = 10
		reserveFee                    = sdk.NewCoins(sdk.NewCoin("foo", sdk.NewInt(1000)))
		minBuyInFee                   = sdk.NewCoins(sdk.NewCoin("foo", sdk.NewInt(1000)))
		minBidIncrement               = sdk.NewCoins(sdk.NewCoin("foo", sdk.NewInt(1000)))
		escrowAddress                 = sdk.AccAddress([]byte("escrow"))
		frontRunningProtection        = true

		// mempool variables
		highestBid = sdk.NewCoins()
	)

	rnd := rand.New(rand.NewSource(time.Now().Unix()))
	bidder := RandomAccounts(rnd, 1)[0]

	cases := []struct {
		name     string
		malleate func()
		pass     bool
	}{
		{
			"insufficient bid amount",
			func() {
				bid = sdk.NewCoins()
			},
			false,
		},
		{
			"insufficient balance",
			func() {
				bid = sdk.NewCoins(sdk.NewCoin("foo", sdk.NewInt(1000)))
				balance = sdk.NewCoins()
			},
			false,
		},
		{
			"bid amount equals the balance (not accounting for the reserve fee)",
			func() {
				balance = sdk.NewCoins(sdk.NewCoin("foo", sdk.NewInt(2000)))
				bid = sdk.NewCoins(sdk.NewCoin("foo", sdk.NewInt(2000)))
			},
			false,
		},
		{
			"too many transactions in the bundle",
			func() {
				// reset the balance and bid to their original values
				bid = sdk.NewCoins(sdk.NewCoin("foo", sdk.NewInt(1000)))
				balance = sdk.NewCoins(sdk.NewCoin("foo", sdk.NewInt(10000)))
				accounts = RandomAccounts(rnd, int(maxBundleSize+1))
			},
			false,
		},
		{
			"frontrunning bundle",
			func() {
				randomAccount := RandomAccounts(rnd, 1)[0]
				accounts = []Account{bidder, randomAccount}
			},
			false,
		},
		{
			"sandwiching bundle",
			func() {
				randomAccount := RandomAccounts(rnd, 1)[0]
				accounts = []Account{bidder, randomAccount, bidder}
			},
			false,
		},
		{
			"valid bundle",
			func() {
				randomAccount := RandomAccounts(rnd, 1)[0]
				accounts = []Account{randomAccount, randomAccount, bidder, bidder, bidder}
			},
			true,
		},
		{
			"valid bundle with only bidder txs",
			func() {
				accounts = []Account{bidder, bidder, bidder, bidder}
			},
			true,
		},
		{
			"valid bundle with only random txs from single same user",
			func() {
				randomAccount := RandomAccounts(rnd, 1)[0]
				accounts = []Account{randomAccount, randomAccount, randomAccount, randomAccount}
			},
			true,
		},
		{
			"invalid bundle with random accounts",
			func() {
				accounts = RandomAccounts(rnd, 2)
			},
			false,
		},
		{
			"disabled front-running protection",
			func() {
				accounts = RandomAccounts(rnd, 10)
				frontRunningProtection = false
			},
			true,
		},
		{
			"invalid bundle that does not outbid the highest bid",
			func() {
				accounts = []Account{bidder, bidder, bidder}
				highestBid = sdk.NewCoins(sdk.NewCoin("foo", sdk.NewInt(500)))
				bid = sdk.NewCoins(sdk.NewCoin("foo", sdk.NewInt(500)))
			},
			false,
		},
		{
			"valid bundle that outbids the highest bid",
			func() {
				highestBid = sdk.NewCoins(sdk.NewCoin("foo", sdk.NewInt(500)))
				bid = sdk.NewCoins(sdk.NewCoin("foo", sdk.NewInt(1500)))
			},
			true,
		},
	}

	for _, tc := range cases {
		suite.Run(tc.name, func() {
			suite.SetupTest() // reset

			tc.malleate()

			// Set up the new auction keeper with mocks customized for this test case
			suite.bankKeeper.EXPECT().GetAllBalances(suite.ctx, bidder.Address).Return(balance).AnyTimes()
			suite.bankKeeper.EXPECT().SendCoins(suite.ctx, bidder.Address, escrowAddress, reserveFee).Return(nil).AnyTimes()

			suite.auctionKeeper = keeper.NewKeeper(
				suite.encCfg.Codec,
				suite.key,
				suite.accountKeeper,
				suite.bankKeeper,
				suite.authorityAccount.String(),
			)
			params := auctiontypes.Params{
				MaxBundleSize:          maxBundleSize,
				ReserveFee:             reserveFee,
				MinBuyInFee:            minBuyInFee,
				EscrowAccountAddress:   escrowAddress.String(),
				FrontRunningProtection: frontRunningProtection,
				MinBidIncrement:        minBidIncrement,
			}
			suite.auctionKeeper.SetParams(suite.ctx, params)

			// Create the bundle of transactions ordered by accounts
			bundle := make([]sdk.Tx, 0)
			for _, acc := range accounts {
				tx, err := createRandomTx(suite.encCfg.TxConfig, acc, 0, 1)
				suite.Require().NoError(err)
				bundle = append(bundle, tx)
			}

			err := suite.auctionKeeper.ValidateAuctionMsg(suite.ctx, bidder.Address, bid, highestBid, bundle)
			if tc.pass {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *IntegrationTestSuite) TestValidateBundle() {
	// TODO: Update this to be multi-dimensional to test multi-sig
	// https://github.com/skip-mev/pob/issues/14
	var accounts []Account // tracks the order of signers in the bundle

	rng := rand.New(rand.NewSource(time.Now().Unix()))
	bidder := RandomAccounts(rng, 1)[0]

	cases := []struct {
		name     string
		malleate func()
		pass     bool
	}{
		{
			"valid empty bundle",
			func() {
				accounts = make([]Account, 0)
			},
			true,
		},
		{
			"valid single tx bundle",
			func() {
				accounts = []Account{bidder}
			},
			true,
		},
		{
			"valid multi-tx bundle by same account",
			func() {
				accounts = []Account{bidder, bidder, bidder, bidder}
			},
			true,
		},
		{
			"valid single-tx bundle by a different account",
			func() {
				randomAccount := RandomAccounts(rng, 1)[0]
				accounts = []Account{randomAccount}
			},
			true,
		},
		{
			"valid multi-tx bundle by a different accounts",
			func() {
				randomAccount := RandomAccounts(rng, 1)[0]
				accounts = []Account{randomAccount, bidder}
			},
			true,
		},
		{
			"invalid frontrunning bundle",
			func() {
				randomAccount := RandomAccounts(rng, 1)[0]
				accounts = []Account{bidder, randomAccount}
			},
			false,
		},
		{
			"invalid sandwiching bundle",
			func() {
				randomAccount := RandomAccounts(rng, 1)[0]
				accounts = []Account{bidder, randomAccount, bidder}
			},
			false,
		},
		{
			"invalid multi account bundle",
			func() {
				accounts = RandomAccounts(rng, 3)
			},
			false,
		},
		{
			"invalid multi account bundle without bidder",
			func() {
				randomAccount1 := RandomAccounts(rng, 1)[0]
				randomAccount2 := RandomAccounts(rng, 1)[0]
				accounts = []Account{randomAccount1, randomAccount2}
			},
			false,
		},
	}

	for _, tc := range cases {
		suite.Run(tc.name, func() {
			suite.SetupTest() // reset

			// Malleate the test case
			tc.malleate()

			// Create the bundle of transactions ordered by accounts
			bundle := make([]sdk.Tx, 0)
			for _, acc := range accounts {
				// Create a random tx
				tx, err := createRandomTx(suite.encCfg.TxConfig, acc, 0, 1)
				suite.Require().NoError(err)
				bundle = append(bundle, tx)
			}

			// Validate the bundle
			err := suite.auctionKeeper.ValidateAuctionBundle(suite.ctx, bidder.Address, bundle)
			if tc.pass {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

// createRandomTx creates a random transaction with a given account, nonce, and number of messages.
func createRandomTx(txCfg client.TxConfig, account Account, nonce, numberMsgs uint64) (authsigning.Tx, error) {
	msgs := make([]sdk.Msg, numberMsgs)
	for i := 0; i < int(numberMsgs); i++ {
		msgs[i] = &banktypes.MsgSend{
			FromAddress: account.Address.String(),
			ToAddress:   account.Address.String(),
		}
	}

	txBuilder := txCfg.NewTxBuilder()
	if err := txBuilder.SetMsgs(msgs...); err != nil {
		return nil, err
	}

	sigV2 := signing.SignatureV2{
		PubKey: account.PrivKey.PubKey(),
		Data: &signing.SingleSignatureData{
			SignMode:  txCfg.SignModeHandler().DefaultMode(),
			Signature: nil,
		},
		Sequence: nonce,
	}
	if err := txBuilder.SetSignatures(sigV2); err != nil {
		return nil, err
	}

	return txBuilder.GetTx(), nil
}
