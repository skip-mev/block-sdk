//go:build e2e

package e2e

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/skip-mev/pob/tests/app"
)

func (s *IntegrationTestSuite) TestGetBuilderParams() {
	params := s.queryBuilderParams()
	s.Require().NotNil(params)
}

// TestBundles tests the execution of various auction bids. There are a few
// invariants that are tested:
//
//  1. The order of transactions in a bundle is preserved when bids are valid.
//  2. All transactions execute as expected.
//  3. The balance of the escrow account should be updated correctly.
//  4. Top of block bids will be included in block proposals before other transactions
//     that are included in the same block.
func (s *IntegrationTestSuite) TestBundles() {
	// Create the accounts that will create transactions to be included in bundles
	initBalance := sdk.NewInt64Coin(app.BondDenom, 10000000000)
	numAccounts := 3
	accounts := s.createTestAccounts(numAccounts, initBalance)

	// basic send amount
	defaultSendAmount := sdk.NewCoins(sdk.NewCoin(app.BondDenom, sdk.NewInt(10)))

	// auction parameters
	params := s.queryBuilderParams()
	reserveFee := params.ReserveFee
	minBidIncrement := params.MinBidIncrement
	maxBundleSize := params.MaxBundleSize
	escrowAddress := params.EscrowAccountAddress

	testCases := []struct {
		name string
		test func()
	}{
		{
			name: "Valid auction bid",
			test: func() {
				// Create a bundle with a single transaction
				bundle := []string{
					s.createMsgSendTx(accounts[0], accounts[1].Address.String(), defaultSendAmount, 0, 1000),
				}

				// Create a bid transaction that includes the bundle and is valid
				bid := reserveFee
				height := s.queryCurrentHeight()
				bidTxHash := s.execAuctionBidTx(0, bid, height+1, bundle)
				s.displayExpectedBundle("Valid auction bid", bidTxHash, bundle)

				// Wait for a block to be created
				s.waitForABlock()

				// Ensure that the block was correctly created and executed in the order expected
				bundleHashes := s.bundleToTxHashes(bundle)
				expectedExecution := map[string]bool{
					bidTxHash:       true,
					bundleHashes[0]: true,
				}
				s.verifyBlock(height+1, bidTxHash, bundleHashes, expectedExecution)

				// Ensure that the escrow account has the correct balance
				expectedEscrowFee := s.calculateProposerEscrowSplit(bid)
				s.Require().Equal(expectedEscrowFee, s.queryBalanceOf(escrowAddress, app.BondDenom))
			},
		},
		{
			name: "invalid auction bid with a bid smaller than the reserve fee",
			test: func() {
				// Get escrow account balance to ensure that it is not changed
				escrowBalance := s.queryBalanceOf(escrowAddress, app.BondDenom)

				// Create a bundle with a single transaction (this should not be included in the block proposal)
				bundle := []string{
					s.createMsgSendTx(accounts[0], accounts[1].Address.String(), defaultSendAmount, 0, 1000),
				}

				// Create a bid transaction that includes a bid that is smaller than the reserve fee
				bid := reserveFee.Sub(sdk.NewInt64Coin(app.BondDenom, 1))
				height := s.queryCurrentHeight()
				bidTxHash := s.execAuctionBidTx(0, bid, height+1, bundle)
				s.displayExpectedBundle("invalid bid", bidTxHash, bundle)

				// Wait for a block to be created
				s.waitForABlock()

				// Ensure that no transactions were executed
				bundleHashes := s.bundleToTxHashes(bundle)
				expectedExecution := map[string]bool{
					bidTxHash:       false,
					bundleHashes[0]: false,
				}
				s.verifyBlock(height+1, bidTxHash, bundleHashes, expectedExecution)

				// Ensure that the escrow account has the correct balance
				s.Require().Equal(escrowBalance, s.queryBalanceOf(escrowAddress, app.BondDenom))
			},
		},
		{
			name: "invalid auction bid with too many transactions in the bundle",
			test: func() {
				// Get escrow account balance to ensure that it is not changed
				escrowBalance := s.queryBalanceOf(escrowAddress, app.BondDenom)

				// Create a bundle with too many transactions
				bundle := []string{}
				for i := 0; i < int(maxBundleSize)+1; i++ {
					bundle = append(bundle, s.createMsgSendTx(accounts[0], accounts[1].Address.String(), defaultSendAmount, i, 1000))
				}

				// Create a bid transaction that includes the bundle
				bid := reserveFee
				height := s.queryCurrentHeight()
				bidTxHash := s.execAuctionBidTx(0, bid, height+1, bundle)
				s.displayExpectedBundle("invalid bid", bidTxHash, bundle)

				// Wait for a block to be created
				s.waitForABlock()

				// Ensure that no transactions were executed
				expectedExecution := map[string]bool{
					bidTxHash: false,
				}

				bundleHashes := s.bundleToTxHashes(bundle)
				for _, hash := range bundleHashes {
					expectedExecution[hash] = false
				}
				s.verifyBlock(height+1, bidTxHash, bundleHashes, expectedExecution)

				// Ensure that the escrow account has the correct balance
				s.Require().Equal(escrowBalance, s.queryBalanceOf(escrowAddress, app.BondDenom))
			},
		},
		{
			name: "invalid auction bid that has an invalid timeout",
			test: func() {
				// Get escrow account balance to ensure that it is not changed
				escrowBalance := s.queryBalanceOf(escrowAddress, app.BondDenom)

				// Create a bundle with a single transaction
				bundle := []string{
					s.createMsgSendTx(accounts[0], accounts[1].Address.String(), defaultSendAmount, 0, 1000),
				}

				// Create a bid transaction that includes the bundle and has a bad timeout
				bid := reserveFee
				height := s.queryCurrentHeight()
				bidTxHash := s.execAuctionBidTx(0, bid, height, bundle)
				s.displayExpectedBundle("invalid bid", bidTxHash, bundle)

				// Wait for a block to be created
				s.waitForABlock()

				// Ensure that no transactions were executed
				bundleHashes := s.bundleToTxHashes(bundle)
				expectedExecution := map[string]bool{
					bidTxHash:       false,
					bundleHashes[0]: false,
				}
				s.verifyBlock(height+1, bidTxHash, bundleHashes, expectedExecution)

				// Ensure that the escrow account has the correct balance
				s.Require().Equal(escrowBalance, s.queryBalanceOf(escrowAddress, app.BondDenom))
			},
		},
		{
			name: "Multiple transactions with second bid being smaller than min bid increment",
			test: func() {
				// Get escrow account balance
				escrowBalance := s.queryBalanceOf(escrowAddress, app.BondDenom)

				// Create a bundle with a single transaction
				bundle := []string{
					s.createMsgSendTx(accounts[0], accounts[1].Address.String(), defaultSendAmount, 0, 1000),
				}

				// Create a bid transaction that includes the bundle and is valid
				bid := reserveFee
				height := s.queryCurrentHeight()
				bidTxHash := s.execAuctionBidTx(0, bid, height+1, bundle)
				s.displayExpectedBundle("bid 1", bidTxHash, bundle)

				// Create a second bid transaction that includes the bundle and is valid (but smaller than the min bid increment)
				badBid := reserveFee.Add(sdk.NewInt64Coin(app.BondDenom, 10))
				bidTxHash2 := s.execAuctionBidTx(0, badBid, height+1, bundle)
				s.displayExpectedBundle("bid 2", bidTxHash2, bundle)

				// Wait for a block to be created
				s.waitForABlock()

				// Ensure only the first bid was executed
				bundleHashes := s.bundleToTxHashes(bundle)
				expectedExecution := map[string]bool{
					bidTxHash:       true,
					bundleHashes[0]: true,
					bidTxHash2:      false,
				}
				s.verifyBlock(height+1, bidTxHash, bundleHashes, expectedExecution)

				// Ensure that the escrow account has the correct balance
				expectedEscrowFee := s.calculateProposerEscrowSplit(bid)
				s.Require().Equal(expectedEscrowFee.Add(escrowBalance), s.queryBalanceOf(escrowAddress, app.BondDenom))
			},
		},
		{
			name: "Multiple transactions with increasing bids but first bid has same bundle so it should fail",
			test: func() {
				// Get escrow account balance
				escrowBalance := s.queryBalanceOf(escrowAddress, app.BondDenom)

				// Create a bundle with a single transaction
				bundle := []string{
					s.createMsgSendTx(accounts[0], accounts[1].Address.String(), defaultSendAmount, 0, 1000),
				}

				// Create a bid transaction that includes the bundle and is valid
				bid := reserveFee
				height := s.queryCurrentHeight()
				bidTxHash := s.execAuctionBidTx(0, bid, height+2, bundle)
				s.displayExpectedBundle("bid 1", bidTxHash, bundle)

				// Create a second bid transaction that includes the bundle and is valid
				bid2 := reserveFee.Add(minBidIncrement)
				bidTxHash2 := s.execAuctionBidTx(1, bid2, height+1, bundle)
				s.displayExpectedBundle("bid 2", bidTxHash2, bundle)

				// Wait for a block to be created
				s.waitForABlock()

				// Ensure only the second bid was executed
				bundleHashes := s.bundleToTxHashes(bundle)
				expectedExecution := map[string]bool{
					bidTxHash:       false,
					bundleHashes[0]: true,
					bidTxHash2:      true,
				}
				s.verifyBlock(height+1, bidTxHash2, bundleHashes, expectedExecution)

				// Wait for a block to be created and ensure that the first bid was not executed
				s.waitForABlock()
				s.verifyBlock(height+2, bidTxHash, bundleHashes, expectedExecution)

				// Ensure that the escrow account has the correct balance
				expectedEscrowFee := s.calculateProposerEscrowSplit(bid2)
				s.Require().Equal(expectedEscrowFee.Add(escrowBalance), s.queryBalanceOf(escrowAddress, app.BondDenom))
			},
		},
		{
			name: "Multiple transactions with increasing bids and different bundles (both should execute)",
			test: func() {
				// Get escrow account balance
				escrowBalance := s.queryBalanceOf(escrowAddress, app.BondDenom)

				// Create a bundle with a single transaction
				firstBundle := []string{
					s.createMsgSendTx(accounts[0], accounts[1].Address.String(), defaultSendAmount, 0, 1000),
				}

				// Create a bundle with a single transaction
				secondBundle := []string{
					s.createMsgSendTx(accounts[1], accounts[0].Address.String(), defaultSendAmount, 0, 1000),
				}

				// Create a bid transaction that includes the bundle and is valid
				bid := reserveFee
				height := s.queryCurrentHeight()
				bidTxHash := s.execAuctionBidTx(0, bid, height+2, firstBundle) // height+2 to ensure it is executed after the second bid
				s.displayExpectedBundle("bid 1", bidTxHash, firstBundle)

				// Create a second bid transaction that includes the bundle and is valid
				bid2 := reserveFee.Add(minBidIncrement)
				bidTxHash2 := s.execAuctionBidTx(1, bid2, height+1, secondBundle)
				s.displayExpectedBundle("bid 2", bidTxHash2, secondBundle)

				// Wait for a block to be created
				s.waitForABlock()

				// Ensure only the second bid was executed
				firstBundleHashes := s.bundleToTxHashes(firstBundle)
				secondBundleHashes := s.bundleToTxHashes(secondBundle)
				expectedExecution := map[string]bool{
					bidTxHash2:            true,
					secondBundleHashes[0]: true,
				}
				s.verifyBlock(height+1, bidTxHash2, secondBundleHashes, expectedExecution)

				// Wait for a block to be created and ensure that the second bid is executed
				s.waitForABlock()
				expectedExecution[bidTxHash] = true
				expectedExecution[firstBundleHashes[0]] = true
				s.verifyBlock(height+2, bidTxHash, firstBundleHashes, expectedExecution)

				// Ensure that the escrow account has the correct balance
				expectedEscrowFee := s.calculateProposerEscrowSplit(bid.Add(bid2))
				s.Require().Equal(expectedEscrowFee.Add(escrowBalance), s.queryBalanceOf(escrowAddress, app.BondDenom))
			},
		},
		{
			name: "Invalid bid that includes an invalid bundle tx",
			test: func() {
				// Get escrow account balance to ensure that it is not changed
				escrowBalance := s.queryBalanceOf(escrowAddress, app.BondDenom)

				// Create a bundle with a single transaction that is invalid (sequence number is wrong)
				bundle := []string{
					s.createMsgSendTx(accounts[0], accounts[1].Address.String(), defaultSendAmount, 1000, 1000),
				}

				// Create a bid transaction that includes the bundle and is valid
				bid := reserveFee
				height := s.queryCurrentHeight()
				bidTxHash := s.execAuctionBidTx(0, bid, height+1, bundle)
				s.displayExpectedBundle("bad bid", bidTxHash, bundle)

				// Wait for a block to be created
				s.waitForABlock()

				bundleHashes := s.bundleToTxHashes(bundle)
				expectedExecution := map[string]bool{
					bidTxHash:       false,
					bundleHashes[0]: false,
				}
				s.verifyBlock(height+1, bidTxHash, bundleHashes, expectedExecution)

				// Ensure that the escrow account has the correct balance
				s.Require().Equal(escrowBalance, s.queryBalanceOf(escrowAddress, app.BondDenom))
			},
		},
		{
			name: "Invalid bid that is attempting to front-run/sandwich",
			test: func() {
				// Get escrow account balance to ensure that it is not changed
				escrowBalance := s.queryBalanceOf(escrowAddress, app.BondDenom)

				// Create a front-running bundle
				bundle := []string{
					s.createMsgSendTx(accounts[0], accounts[1].Address.String(), defaultSendAmount, 0, 1000),
					s.createMsgSendTx(accounts[1], accounts[0].Address.String(), defaultSendAmount, 0, 1000),
					s.createMsgSendTx(accounts[2], accounts[1].Address.String(), defaultSendAmount, 0, 1000),
				}

				// Create a bid transaction that includes the bundle and is valid
				bid := reserveFee
				height := s.queryCurrentHeight()
				bidTxHash := s.execAuctionBidTx(0, bid, height+1, bundle)
				s.displayExpectedBundle("bad bid", bidTxHash, bundle)

				// Wait for a block to be created
				s.waitForABlock()

				bundleHashes := s.bundleToTxHashes(bundle)
				expectedExecution := map[string]bool{
					bidTxHash:       false,
					bundleHashes[0]: false,
					bundleHashes[1]: false,
					bundleHashes[2]: false,
				}
				s.verifyBlock(height+1, bidTxHash, bundleHashes, expectedExecution)

				// Ensure that the escrow account has the correct balance
				s.Require().Equal(escrowBalance, s.queryBalanceOf(escrowAddress, app.BondDenom))
			},
		},
		{
			name: "Invalid bid that is attempting to bid more than their balance",
			test: func() {
				// Get escrow account balance to ensure that it is not changed
				escrowBalance := s.queryBalanceOf(escrowAddress, app.BondDenom)

				// Create a bundle with a single transaction that is valid
				bundle := []string{
					s.createMsgSendTx(accounts[0], accounts[1].Address.String(), defaultSendAmount, 0, 1000),
				}

				// Create a bid transaction that includes the bundle and is valid
				bid := sdk.NewCoin(app.BondDenom, sdk.NewInt(999999999999999999))
				height := s.queryCurrentHeight()
				bidTxHash := s.execAuctionBidTx(0, bid, height+1, bundle)
				s.displayExpectedBundle("bad bid", bidTxHash, bundle)

				// Wait for a block to be created
				s.waitForABlock()

				bundleHashes := s.bundleToTxHashes(bundle)
				expectedExecution := map[string]bool{
					bidTxHash:       false,
					bundleHashes[0]: false,
				}
				s.verifyBlock(height+1, bidTxHash, bundleHashes, expectedExecution)

				// Ensure that the escrow account has the correct balance
				s.Require().Equal(escrowBalance, s.queryBalanceOf(escrowAddress, app.BondDenom))
			},
		},
		{
			name: "Valid bid with multiple other transactions",
			test: func() {
				// Get escrow account balance to ensure that it is updated correctly
				escrowBalance := s.queryBalanceOf(escrowAddress, app.BondDenom)

				// Create a bundle with a multiple transaction that is valid
				bundle := []string{
					s.createMsgSendTx(accounts[0], accounts[1].Address.String(), defaultSendAmount, 0, 1000),
					s.createMsgSendTx(accounts[0], accounts[1].Address.String(), defaultSendAmount, 1, 1000),
					s.createMsgSendTx(accounts[0], accounts[1].Address.String(), defaultSendAmount, 2, 1000),
					s.createMsgSendTx(accounts[0], accounts[1].Address.String(), defaultSendAmount, 3, 1000),
				}

				// Wait for a block to ensure all transactions are included in the same block
				s.waitForABlock()

				// Create a bid transaction that includes the bundle and is valid
				bid := reserveFee
				height := s.queryCurrentHeight()
				bidTxHash := s.execAuctionBidTx(0, bid, height+1, bundle)
				s.displayExpectedBundle("good bid", bidTxHash, bundle)

				// Execute a few other messages to be included in the block after the bid and bundle
				txHash1 := s.execMsgSendTx(1, accounts[0].Address, sdk.NewCoin(app.BondDenom, sdk.NewInt(100)))
				txHash2 := s.execMsgSendTx(2, accounts[0].Address, sdk.NewCoin(app.BondDenom, sdk.NewInt(100)))
				txHash3 := s.execMsgSendTx(3, accounts[0].Address, sdk.NewCoin(app.BondDenom, sdk.NewInt(100)))

				// Wait for a block to be created
				s.waitForABlock()

				bundleHashes := s.bundleToTxHashes(bundle)
				expectedExecution := map[string]bool{
					bidTxHash:       true,
					bundleHashes[0]: true,
					bundleHashes[1]: true,
					bundleHashes[2]: true,
					bundleHashes[3]: true,
					txHash1:         true,
					txHash2:         true,
					txHash3:         true,
				}
				s.verifyBlock(height+1, bidTxHash, bundleHashes, expectedExecution)

				// Ensure that the escrow account has the correct balance
				expectedEscrowFee := s.calculateProposerEscrowSplit(bid)
				s.Require().Equal(escrowBalance.Add(expectedEscrowFee), s.queryBalanceOf(escrowAddress, app.BondDenom))
			},
		},
	}

	for _, tc := range testCases {
		s.waitForABlock()
		s.Run(tc.name, tc.test)
	}
}
