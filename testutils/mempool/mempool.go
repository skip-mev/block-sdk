package mempool

import (
	"cosmossdk.io/math"
	"github.com/cometbft/cometbft/libs/log"

	signerextraction "github.com/skip-mev/block-sdk/adapters/signer_extraction_adapter"
	"github.com/skip-mev/block-sdk/block"
	"github.com/skip-mev/block-sdk/block/base"
	defaultlane "github.com/skip-mev/block-sdk/lanes/base"
	"github.com/skip-mev/block-sdk/lanes/free"
	"github.com/skip-mev/block-sdk/lanes/mev"
	"github.com/skip-mev/block-sdk/testutils"
)

func CreateMempool() *block.LanedMempool {
	encodingConfig := testutils.CreateTestEncodingConfig()
	signerExtractor := signerextraction.NewDefaultAdapter()

	mevConfig := base.LaneConfig{
		SignerExtractor: signerExtractor,
		Logger:          log.NewNopLogger(),
		TxEncoder:       encodingConfig.TxConfig.TxEncoder(),
		TxDecoder:       encodingConfig.TxConfig.TxDecoder(),
		AnteHandler:     nil,
		MaxBlockSpace:   math.LegacyMustNewDecFromStr("0.3"),
		MaxTxs:          0, // unlimited
	}
	factory := mev.NewDefaultAuctionFactory(encodingConfig.TxConfig.TxDecoder(), signerExtractor)
	mevLane := mev.NewMEVLane(mevConfig, factory, factory.MatchHandler())

	freeConfig := base.LaneConfig{
		SignerExtractor: signerExtractor,
		Logger:          log.NewNopLogger(),
		TxEncoder:       encodingConfig.TxConfig.TxEncoder(),
		TxDecoder:       encodingConfig.TxConfig.TxDecoder(),
		AnteHandler:     nil,
		MaxBlockSpace:   math.LegacyMustNewDecFromStr("0.3"),
		MaxTxs:          0, // unlimited
	}
	freeLane := free.NewFreeLane(freeConfig, base.DefaultTxPriority(), free.DefaultMatchHandler())

	defaultConfig := base.LaneConfig{
		SignerExtractor: signerExtractor,
		Logger:          log.NewNopLogger(),
		TxEncoder:       encodingConfig.TxConfig.TxEncoder(),
		TxDecoder:       encodingConfig.TxConfig.TxDecoder(),
		AnteHandler:     nil,
		MaxBlockSpace:   math.LegacyZeroDec(),
		MaxTxs:          0, // unlimited
	}
	defaultLane := defaultlane.NewDefaultLane(defaultConfig, base.DefaultMatchHandler())

	lanes := []block.Lane{mevLane, freeLane, defaultLane}
	mempool, err := block.NewLanedMempool(log.NewNopLogger(), lanes)
	if err != nil {
		panic(err)
	}

	return mempool
}
