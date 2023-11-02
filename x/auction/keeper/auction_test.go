package keeper_test

import (
	"fmt"
	"math/rand"
	"time"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	mock "github.com/stretchr/testify/mock"

	testutils "github.com/skip-mev/block-sdk/testutils"
	"github.com/skip-mev/block-sdk/x/auction/keeper"
	"github.com/skip-mev/block-sdk/x/auction/types"
	"github.com/skip-mev/block-sdk/x/auction/types/mocks"
)

func (s *KeeperTestSuite) TestValidateAuctionBid() {
	rng := rand.New(rand.NewSource(time.Now().Unix()))
	bidder := testutils.RandomAccounts(rng, 1)[0].Address
	bankSendErr := fmt.Errorf("bank send error")

	params := types.Params{
		ReserveFee:           sdk.NewCoin("stake", math.NewInt(1000)),
		EscrowAccountAddress: sdk.AccAddress([]byte("escrow")),
		MinBidIncrement:      sdk.NewCoin("stake", math.NewInt(1000)),
		ProposerFee:          math.LegacyZeroDec(),
	}
	s.Require().NoError(s.auctionkeeper.SetParams(s.ctx, params))

	s.Run("nil bid", func() {
		highestBid := sdk.NewCoin("stake", math.NewInt(1000))
		s.Require().Error(s.auctionkeeper.ValidateAuctionBid(s.ctx, bidder, sdk.Coin{}, highestBid))
	})

	s.Run("reserve fee and bid denom mismatch", func() {
		highestBid := sdk.NewCoin("stake", math.NewInt(1000))
		bid := sdk.NewCoin("stake2", math.NewInt(1000))
		s.Require().Error(s.auctionkeeper.ValidateAuctionBid(s.ctx, bidder, bid, highestBid))
	})

	s.Run("bid less than reserve fee", func() {
		highestBid := sdk.NewCoin("stake", math.NewInt(1000))
		bid := sdk.NewCoin("stake", math.NewInt(500))
		s.Require().Error(s.auctionkeeper.ValidateAuctionBid(s.ctx, bidder, bid, highestBid))
	})

	s.Run("bid less than highest bid + min bid increment", func() {
		highestBid := sdk.NewCoin("stake", math.NewInt(1000))
		bid := sdk.NewCoin("stake", math.NewInt(1500))
		s.Require().Error(s.auctionkeeper.ValidateAuctionBid(s.ctx, bidder, bid, highestBid))
	})

	s.Run("valid bid", func() {
		highestBid := sdk.Coin{}
		bid := sdk.NewCoin("stake", math.NewInt(1500))

		s.bankKeeper.On("SendCoins", mock.Anything, mock.Anything, mock.Anything, sdk.NewCoins(bid)).Return(nil)

		err := s.auctionkeeper.ValidateAuctionBid(s.ctx, bidder, bid, highestBid)
		s.Require().NoError(err)
	})

	s.Run("insufficient funds", func() {
		highestBid := sdk.Coin{}
		bid := sdk.NewCoin("stake", math.NewInt(1500))

		s.bankKeeper = mocks.NewBankKeeper(s.T())
		s.auctionkeeper = keeper.NewKeeper(
			s.encCfg.Codec,
			s.key,
			s.accountKeeper,
			s.bankKeeper,
			s.distrKeeper,
			s.stakingKeeper,
			s.authorityAccount.String(),
		)

		s.bankKeeper.On("SendCoins", mock.Anything, mock.Anything, mock.Anything, sdk.NewCoins(bid)).Return(bankSendErr)

		err := s.auctionkeeper.ValidateAuctionBid(s.ctx, bidder, bid, highestBid)
		s.Require().Error(err)
	})

	s.Run("valid bid with proposer split", func() {
		highestBid := sdk.Coin{}
		bid := sdk.NewCoin("stake", math.NewInt(1000))

		s.bankKeeper = mocks.NewBankKeeper(s.T())
		rewardsProvider := mocks.NewRewardsAddressProvider(s.T())
		rewardsAddr := sdk.AccAddress([]byte("rewards"))
		rewardsProvider.On("GetRewardsAddress", mock.Anything).Return(rewardsAddr, nil)

		s.auctionkeeper = keeper.NewKeeperWithRewardsAddressProvider(
			s.encCfg.Codec,
			s.key,
			s.accountKeeper,
			s.bankKeeper,
			rewardsProvider,
			s.authorityAccount.String(),
		)

		params := types.Params{
			ProposerFee:          math.LegacyMustNewDecFromStr("0.1"),
			ReserveFee:           sdk.NewCoin("stake", math.NewInt(1000)),
			EscrowAccountAddress: sdk.AccAddress([]byte("escrow")),
			MinBidIncrement:      sdk.NewCoin("stake", math.NewInt(1000)),
		}
		s.Require().NoError(s.auctionkeeper.SetParams(s.ctx, params))

		proposalSplit := sdk.NewCoin("stake", math.NewInt(100))
		escrowSplit := sdk.NewCoin("stake", math.NewInt(900))
		s.bankKeeper.On("SendCoins", mock.Anything, mock.Anything, mock.Anything, sdk.NewCoins(proposalSplit)).Return(nil)
		s.bankKeeper.On("SendCoins", mock.Anything, mock.Anything, mock.Anything, sdk.NewCoins(escrowSplit)).Return(nil)

		err := s.auctionkeeper.ValidateAuctionBid(s.ctx, bidder, bid, highestBid)
		s.Require().NoError(err)
	})
}

func (s *KeeperTestSuite) TestValidateBundle() {
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
		s.Run(tc.name, func() {
			s.SetupTest() // reset

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
			err := s.auctionkeeper.ValidateAuctionBundle(bidder.Address, signers)
			if tc.pass {
				s.Require().NoError(err)
			} else {
				s.Require().Error(err)
			}
		})
	}
}

func (s *KeeperTestSuite) TestValidateBundleTimeouts() {
	s.Run("can validate valid bundle timeouts", func() {
		bidder := sdk.AccAddress([]byte("bidder"))
		other := sdk.AccAddress([]byte("other"))

		bidInfo := &types.BidInfo{
			Bidder:              sdk.AccAddress([]byte("bidder")),
			Timeout:             10,
			TransactionTimeouts: []uint64{0, 10, 0},
			Signers: []map[string]struct{}{
				{other.String(): {}},
				{bidder.String(): {}},
				{other.String(): {}},
			},
		}

		err := s.auctionkeeper.ValidateBundleTimeouts(bidInfo)
		s.Require().NoError(err)
	})

	s.Run("can invalidate invalid bundle timeouts", func() {
		bidder := sdk.AccAddress([]byte("bidder"))
		other := sdk.AccAddress([]byte("other"))

		bidInfo := &types.BidInfo{
			Bidder:              sdk.AccAddress([]byte("bidder")),
			Timeout:             10,
			TransactionTimeouts: []uint64{0, 10, 0},
			Signers: []map[string]struct{}{
				{other.String(): {}},
				{bidder.String(): {}},
				{bidder.String(): {}},
			},
		}

		err := s.auctionkeeper.ValidateBundleTimeouts(bidInfo)
		s.Require().Error(err)
	})

	s.Run("can invalidate invalid bundle timeout with single tx", func() {
		bidder := sdk.AccAddress([]byte("bidder"))

		bidInfo := &types.BidInfo{
			Bidder:              sdk.AccAddress([]byte("bidder")),
			Timeout:             10,
			TransactionTimeouts: []uint64{0},
			Signers: []map[string]struct{}{
				{bidder.String(): {}},
			},
		}

		err := s.auctionkeeper.ValidateBundleTimeouts(bidInfo)
		s.Require().Error(err)
	})
}
