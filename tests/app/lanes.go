package app

import (
	"cosmossdk.io/math"
	signerextraction "github.com/skip-mev/block-sdk/adapters/signer_extraction_adapter"
	"github.com/skip-mev/block-sdk/block/base"
	defaultlane "github.com/skip-mev/block-sdk/lanes/base"
	freelane "github.com/skip-mev/block-sdk/lanes/free"
	mevlane "github.com/skip-mev/block-sdk/lanes/mev"
)

// CreateLanes walks through the process of creating the lanes for the block sdk. In this function
// we create three separate lanes - MEV, Free, and Default - and then return them.
//
// NOTE: Application Developers should closely replicate this function in their own application.
func CreateLanes(app *TestApp) (*mevlane.MEVLane, *freelane.FreeLane, *defaultlane.DefaultLane) {
	// 1. Create the signer extractor. This is used to extract the expected signers from
	// a transaction. Each lane can have a different signer extractor if needed.
	signerAdapter := signerextraction.NewDefaultAdapter()

	// 2. Create the configurations for each lane. These configurations determine how many
	// transactions the lane can store, the maximum block space the lane can consume, and
	// the signer extractor used to extract the expected signers from a transaction.
	//
	// IMPORTANT NOTE: If the block sdk module is utilized to store lanes, than the maximum
	// block space will be replaced with what is in state / in the genesis file.

	// Create a mev configuration that accepts 1000 transactions and consumes 20% of the
	// block space.
	mevConfig := base.LaneConfig{
		Logger:          app.Logger(),
		TxEncoder:       app.txConfig.TxEncoder(),
		TxDecoder:       app.txConfig.TxDecoder(),
		MaxBlockSpace:   math.LegacyMustNewDecFromStr("0.2"),
		SignerExtractor: signerAdapter,
		MaxTxs:          1000,
	}

	// Create a free configuration that accepts 1000 transactions and consumes 20% of the
	// block space.
	freeConfig := base.LaneConfig{
		Logger:          app.Logger(),
		TxEncoder:       app.txConfig.TxEncoder(),
		TxDecoder:       app.txConfig.TxDecoder(),
		MaxBlockSpace:   math.LegacyMustNewDecFromStr("0.2"),
		SignerExtractor: signerAdapter,
		MaxTxs:          1000,
	}

	// Create a default configuration that accepts 1000 transactions and consumes 60% of the
	// block space.
	defaultConfig := base.LaneConfig{
		Logger:          app.Logger(),
		TxEncoder:       app.txConfig.TxEncoder(),
		TxDecoder:       app.txConfig.TxDecoder(),
		MaxBlockSpace:   math.LegacyMustNewDecFromStr("0.6"),
		SignerExtractor: signerAdapter,
		MaxTxs:          1000,
	}

	// 3. Create the match handlers for each lane. These match handlers determine whether or not
	// a transaction belongs in the lane. We want each lane to be mutually exclusive, so we create
	// a match handler that matches transactions that belong in the lane and do not match with any
	// of the other lanes. The outcome of this looks like the following:
	// - MEV Lane: Matches transactions that belong in the MEV lane and do not match the free lane
	//   transactions. We do not consider the default lane transactions as this accepts all transactions
	//   and would always return false.
	// - Free Lane: Matches transactions that belong in the free lane and do not match the MEV lane
	//   transactions.
	// - Default Lane: Matches transactions that belong in the default lane and do not match the MEV
	//   or free lane.
	factory := mevlane.NewDefaultAuctionFactory(app.txConfig.TxDecoder(), signerAdapter)

	// Create the final match handler for the mev lane.
	mevMatchHandler := base.NewMatchHandler(
		factory.MatchHandler(),
		freelane.DefaultMatchHandler(),
	)

	// Create the final match handler for the free lane.
	freeMatchHandler := base.NewMatchHandler(
		freelane.DefaultMatchHandler(),
		factory.MatchHandler(),
	)

	// Create the final match handler for the default lane.
	defaultMatchHandler := base.NewMatchHandler(
		base.DefaultMatchHandler(),
		factory.MatchHandler(),
		freelane.DefaultMatchHandler(),
	)

	// 4. Create the lanes.
	mevLane := mevlane.NewMEVLane(
		mevConfig,
		factory,
		mevMatchHandler,
	)

	freeLane := freelane.NewFreeLane(
		freeConfig,
		base.DefaultTxPriority(),
		freeMatchHandler,
	)

	defaultLane := defaultlane.NewDefaultLane(
		defaultConfig,
		defaultMatchHandler,
	)

	return mevLane, freeLane, defaultLane
}
