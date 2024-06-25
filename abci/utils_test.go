package abci_test

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"cosmossdk.io/log"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	tmprototypes "github.com/cometbft/cometbft/proto/tendermint/types"

	"github.com/skip-mev/block-sdk/v2/abci"
	signeradaptors "github.com/skip-mev/block-sdk/v2/adapters/signer_extraction_adapter"
	"github.com/skip-mev/block-sdk/v2/block"
	"github.com/skip-mev/block-sdk/v2/block/base"
	defaultlane "github.com/skip-mev/block-sdk/v2/lanes/base"
	"github.com/skip-mev/block-sdk/v2/lanes/free"
	"github.com/skip-mev/block-sdk/v2/lanes/mev"
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

	options := []base.LaneOption{
		base.WithMatchHandler(mh),
		base.WithMempoolConfigs(cfg, base.DefaultTxPriority()),
	}

	lane, err := base.NewBaseLane(
		cfg,
		name,
		options...,
	)
	s.Require().NoError(err)

	return lane
}

func (s *ProposalsTestSuite) setUpStandardLane(maxBlockSpace math.LegacyDec, expectedExecution map[sdk.Tx]bool) *base.BaseLane {
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

func (s *ProposalsTestSuite) setUpFreeLane(maxBlockSpace math.LegacyDec, expectedExecution map[sdk.Tx]bool) *base.BaseLane {
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

	options := []base.LaneOption{
		base.WithMatchHandler(base.DefaultMatchHandler()),
		base.WithMempoolConfigs(cfg, base.DefaultTxPriority()),
		base.WithPrepareLaneHandler(base.PanicPrepareLaneHandler()),
		base.WithProcessLaneHandler(base.PanicProcessLaneHandler()),
	}

	lane, err := base.NewBaseLane(
		cfg,
		name,
		options...,
	)
	s.Require().NoError(err)

	return lane
}

func (s *ProposalsTestSuite) setUpProposalHandlers(lanes []block.Lane) *abci.ProposalHandler {
	mempool, err := block.NewLanedMempool(
		log.NewNopLogger(),
		lanes,
	)
	s.Require().NoError(err)

	return abci.New(
		log.NewNopLogger(),
		s.encodingConfig.TxConfig.TxDecoder(),
		s.encodingConfig.TxConfig.TxEncoder(),
		mempool,
		true,
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
