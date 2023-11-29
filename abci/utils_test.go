package abci_test

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	blocksdkmoduletypes "github.com/skip-mev/block-sdk/x/blocksdk/types"

	"cosmossdk.io/log"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	tmprototypes "github.com/cometbft/cometbft/proto/tendermint/types"

	"github.com/skip-mev/block-sdk/abci"
	signeradaptors "github.com/skip-mev/block-sdk/adapters/signer_extraction_adapter"
	"github.com/skip-mev/block-sdk/block"
	"github.com/skip-mev/block-sdk/block/base"
	defaultlane "github.com/skip-mev/block-sdk/lanes/base"
	"github.com/skip-mev/block-sdk/lanes/free"
	"github.com/skip-mev/block-sdk/lanes/mev"
)

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

func (s *ProposalsTestSuite) setUpCustomMatchHandlerLane(maxBlockSpace math.LegacyDec, expectedExecution map[sdk.Tx]bool, mh base.MatchHandler, name string) block.Lane {
	cfg := base.LaneConfig{
		Logger:          log.NewNopLogger(),
		TxEncoder:       s.encodingConfig.TxConfig.TxEncoder(),
		TxDecoder:       s.encodingConfig.TxConfig.TxDecoder(),
		AnteHandler:     s.setUpAnteHandler(expectedExecution),
		MaxBlockSpace:   maxBlockSpace,
		SignerExtractor: signeradaptors.NewDefaultAdapter(),
	}

	lane := base.NewBaseLane(
		cfg,
		name,
		base.NewMempool[string](base.DefaultTxPriority(), cfg.TxEncoder, cfg.SignerExtractor, 0),
		mh,
	)

	lane.SetPrepareLaneHandler(lane.DefaultPrepareLaneHandler())
	lane.SetProcessLaneHandler(lane.DefaultProcessLaneHandler())

	return lane
}

func (s *ProposalsTestSuite) setUpStandardLane(maxBlockSpace math.LegacyDec, expectedExecution map[sdk.Tx]bool) *defaultlane.DefaultLane {
	cfg := base.LaneConfig{
		Logger:          log.NewNopLogger(),
		TxEncoder:       s.encodingConfig.TxConfig.TxEncoder(),
		TxDecoder:       s.encodingConfig.TxConfig.TxDecoder(),
		AnteHandler:     s.setUpAnteHandler(expectedExecution),
		MaxBlockSpace:   maxBlockSpace,
		SignerExtractor: signeradaptors.NewDefaultAdapter(),
	}

	return defaultlane.NewDefaultLane(cfg, base.DefaultMatchHandler())
}

func (s *ProposalsTestSuite) setUpTOBLane(maxBlockSpace math.LegacyDec, expectedExecution map[sdk.Tx]bool) *mev.MEVLane {
	cfg := base.LaneConfig{
		Logger:          log.NewNopLogger(),
		TxEncoder:       s.encodingConfig.TxConfig.TxEncoder(),
		TxDecoder:       s.encodingConfig.TxConfig.TxDecoder(),
		AnteHandler:     s.setUpAnteHandler(expectedExecution),
		MaxBlockSpace:   maxBlockSpace,
		SignerExtractor: signeradaptors.NewDefaultAdapter(),
	}

	factory := mev.NewDefaultAuctionFactory(cfg.TxDecoder, signeradaptors.NewDefaultAdapter())
	return mev.NewMEVLane(cfg, factory, factory.MatchHandler())
}

func (s *ProposalsTestSuite) setUpFreeLane(maxBlockSpace math.LegacyDec, expectedExecution map[sdk.Tx]bool) *free.FreeLane {
	cfg := base.LaneConfig{
		Logger:          log.NewNopLogger(),
		TxEncoder:       s.encodingConfig.TxConfig.TxEncoder(),
		TxDecoder:       s.encodingConfig.TxConfig.TxDecoder(),
		AnteHandler:     s.setUpAnteHandler(expectedExecution),
		MaxBlockSpace:   maxBlockSpace,
		SignerExtractor: signeradaptors.NewDefaultAdapter(),
	}

	return free.NewFreeLane(cfg, base.DefaultTxPriority(), free.DefaultMatchHandler())
}

func (s *ProposalsTestSuite) setUpPanicLane(name string, maxBlockSpace math.LegacyDec) *base.BaseLane {
	cfg := base.LaneConfig{
		Logger:          log.NewNopLogger(),
		TxEncoder:       s.encodingConfig.TxConfig.TxEncoder(),
		TxDecoder:       s.encodingConfig.TxConfig.TxDecoder(),
		MaxBlockSpace:   maxBlockSpace,
		SignerExtractor: signeradaptors.NewDefaultAdapter(),
	}

	lane := base.NewBaseLane(
		cfg,
		name,
		base.NewMempool[string](base.DefaultTxPriority(), cfg.TxEncoder, cfg.SignerExtractor, 0),
		base.DefaultMatchHandler(),
	)

	lane.SetPrepareLaneHandler(base.PanicPrepareLaneHandler())
	lane.SetProcessLaneHandler(base.PanicProcessLaneHandler())

	return lane
}

func (s *ProposalsTestSuite) setUpProposalHandlers(lanes []block.Lane) *abci.ProposalHandler {
	blocksdkLanes := make([]blocksdkmoduletypes.Lane, len(lanes))
	for i, lane := range lanes {
		blocksdkLanes[i] = blocksdkmoduletypes.Lane{
			Id:            lane.Name(),
			MaxBlockSpace: lane.GetMaxBlockSpace(),
			Order:         uint64(i),
		}
	}

	laneFetcher := NewMockLaneFetcher(
		func() (blocksdkmoduletypes.Lane, error) {
			return blocksdkmoduletypes.Lane{}, nil
		},
		func() []blocksdkmoduletypes.Lane {
			return blocksdkLanes
		})

	mempool, err := block.NewLanedMempool(
		log.NewNopLogger(),
		lanes,
		laneFetcher,
	)
	s.Require().NoError(err)

	return abci.NewProposalHandler(
		log.NewNopLogger(),
		s.encodingConfig.TxConfig.TxDecoder(),
		s.encodingConfig.TxConfig.TxEncoder(),
		mempool,
	)
}

func (s *ProposalsTestSuite) createProposal(txs ...sdk.Tx) [][]byte {
	return s.getTxBytes(txs...)
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

func (s *ProposalsTestSuite) setBlockParams(maxGasLimit, maxBlockSize int64) {
	s.ctx = s.ctx.WithConsensusParams(
		tmprototypes.ConsensusParams{
			Block: &tmprototypes.BlockParams{
				MaxBytes: maxBlockSize,
				MaxGas:   maxGasLimit,
			},
		},
	)
}
