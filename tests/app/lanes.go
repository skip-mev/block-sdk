package app

import (
	"cosmossdk.io/math"
	signerextraction "github.com/skip-mev/block-sdk/adapters/signer_extraction_adapter"
	"github.com/skip-mev/block-sdk/block/base"
	defaultlane "github.com/skip-mev/block-sdk/lanes/base"
	freelane "github.com/skip-mev/block-sdk/lanes/free"
	mevlane "github.com/skip-mev/block-sdk/lanes/mev"
)

func CreateLanes(app *TestApp) (*mevlane.MEVLane, *freelane.FreeLane, *defaultlane.DefaultLane) {
	// Create the signer extractor. This is used to extract the expected signers from
	// a transaction. Each lane can have a different signer extractor if needed.
	signerAdapter := signerextraction.NewDefaultAdapter()

	// Create the configurations for each lane. These configurations determine how many
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

	// Create the match handlers for each lane. These match handlers determine whether or not
	// a transaction belongs in the lane.
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

	// Create the lanes.
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