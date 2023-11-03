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
	"github.com/skip-mev/block-sdk/block/proposals"
	"github.com/skip-mev/block-sdk/block/proposals/types"
	"github.com/skip-mev/block-sdk/block/utils"
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

func (s *ProposalsTestSuite) setUpStandardLane(maxBlockSpace math.LegacyDec, expectedExecution map[sdk.Tx]bool) *defaultlane.DefaultLane {
	cfg := base.LaneConfig{
		Logger:          log.NewTestLogger(s.T()),
		TxEncoder:       s.encodingConfig.TxConfig.TxEncoder(),
		TxDecoder:       s.encodingConfig.TxConfig.TxDecoder(),
		AnteHandler:     s.setUpAnteHandler(expectedExecution),
		MaxBlockSpace:   maxBlockSpace,
		SignerExtractor: signeradaptors.NewDefaultAdapter(),
	}

	return defaultlane.NewDefaultLane(cfg)
}

func (s *ProposalsTestSuite) setUpTOBLane(maxBlockSpace math.LegacyDec, expectedExecution map[sdk.Tx]bool) *mev.MEVLane {
	cfg := base.LaneConfig{
		Logger:          log.NewTestLogger(s.T()),
		TxEncoder:       s.encodingConfig.TxConfig.TxEncoder(),
		TxDecoder:       s.encodingConfig.TxConfig.TxDecoder(),
		AnteHandler:     s.setUpAnteHandler(expectedExecution),
		MaxBlockSpace:   maxBlockSpace,
		SignerExtractor: signeradaptors.NewDefaultAdapter(),
	}

	return mev.NewMEVLane(cfg, mev.NewDefaultAuctionFactory(cfg.TxDecoder, signeradaptors.NewDefaultAdapter()))
}

func (s *ProposalsTestSuite) setUpFreeLane(maxBlockSpace math.LegacyDec, expectedExecution map[sdk.Tx]bool) *free.FreeLane {
	cfg := base.LaneConfig{
		Logger:          log.NewTestLogger(s.T()),
		TxEncoder:       s.encodingConfig.TxConfig.TxEncoder(),
		TxDecoder:       s.encodingConfig.TxConfig.TxDecoder(),
		AnteHandler:     s.setUpAnteHandler(expectedExecution),
		MaxBlockSpace:   maxBlockSpace,
		SignerExtractor: signeradaptors.NewDefaultAdapter(),
	}

	return free.NewFreeLane(cfg, base.DefaultTxPriority(), free.DefaultMatchHandler())
}

func (s *ProposalsTestSuite) setUpPanicLane(maxBlockSpace math.LegacyDec) *base.BaseLane {
	cfg := base.LaneConfig{
		Logger:          log.NewTestLogger(s.T()),
		TxEncoder:       s.encodingConfig.TxConfig.TxEncoder(),
		TxDecoder:       s.encodingConfig.TxConfig.TxDecoder(),
		MaxBlockSpace:   maxBlockSpace,
		SignerExtractor: signeradaptors.NewDefaultAdapter(),
	}

	lane := base.NewBaseLane(
		cfg,
		"panic",
		base.NewMempool[string](base.DefaultTxPriority(), cfg.TxEncoder, cfg.SignerExtractor, 0),
		base.DefaultMatchHandler(),
	)

	lane.SetPrepareLaneHandler(base.PanicPrepareLaneHandler())
	lane.SetProcessLaneHandler(base.PanicProcessLaneHandler())

	return lane
}

func (s *ProposalsTestSuite) setUpProposalHandlers(lanes []block.Lane) *abci.ProposalHandler {
	laneFetcher := NewMockLaneFetcher(
		func() (blocksdkmoduletypes.Lane, error) {
			return blocksdkmoduletypes.Lane{}, nil
		},
		func() []blocksdkmoduletypes.Lane {
			blocksdkLanes := make([]blocksdkmoduletypes.Lane, len(lanes))
			for i, lane := range lanes {
				blocksdkLanes[i] = blocksdkmoduletypes.Lane{
					Id:            lane.Name(),
					MaxBlockSpace: lane.GetMaxBlockSpace(),
					Order:         uint64(i),
				}
			}
			return blocksdkLanes
		})

	mempool := block.NewLanedMempool(log.NewTestLogger(
		s.T()),
		true,
		laneFetcher,
		lanes...,
	)

	return abci.NewProposalHandler(
		log.NewTestLogger(s.T()),
		s.encodingConfig.TxConfig.TxDecoder(),
		s.encodingConfig.TxConfig.TxEncoder(),
		mempool,
	)
}

func (s *ProposalsTestSuite) createProposal(distribution map[string]uint64, txs ...sdk.Tx) [][]byte {
	maxSize, maxGasLimit := proposals.GetBlockLimits(s.ctx)
	size, limit := s.getTxInfos(txs...)

	info := s.createProposalInfoBytes(
		maxGasLimit,
		limit,
		maxSize,
		size,
		distribution,
	)

	proposal := s.getTxBytes(txs...)
	return append([][]byte{info}, proposal...)
}

func (s *ProposalsTestSuite) getProposalInfo(bz []byte) types.ProposalInfo {
	var info types.ProposalInfo
	s.Require().NoError(info.Unmarshal(bz))
	return info
}

func (s *ProposalsTestSuite) createProposalInfo(
	maxGasLimit, gasLimit uint64,
	maxBlockSize, blockSize int64,
	txsByLane map[string]uint64,
) types.ProposalInfo {
	return types.ProposalInfo{
		MaxGasLimit:  maxGasLimit,
		GasLimit:     gasLimit,
		MaxBlockSize: maxBlockSize,
		BlockSize:    blockSize,
		TxsByLane:    txsByLane,
	}
}

func (s *ProposalsTestSuite) createProposalInfoBytes(
	maxGasLimit, gasLimit uint64,
	maxBlockSize, blockSize int64,
	txsByLane map[string]uint64,
) []byte {
	info := s.createProposalInfo(maxGasLimit, gasLimit, maxBlockSize, blockSize, txsByLane)
	bz, err := info.Marshal()
	s.Require().NoError(err)
	return bz
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

func (s *ProposalsTestSuite) getTxInfos(txs ...sdk.Tx) (int64, uint64) {
	totalSize := int64(0)
	totalGasLimit := uint64(0)

	for _, tx := range txs {
		info, err := utils.GetTxInfo(s.encodingConfig.TxConfig.TxEncoder(), tx)
		s.Require().NoError(err)

		totalSize += info.Size
		totalGasLimit += info.GasLimit
	}

	return totalSize, totalGasLimit
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
