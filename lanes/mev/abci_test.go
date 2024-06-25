package mev_test

import (
	log "cosmossdk.io/log"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/skip-mev/block-sdk/v2/block"
	"github.com/skip-mev/block-sdk/v2/block/proposals"
	"github.com/skip-mev/block-sdk/v2/block/utils"
	"github.com/skip-mev/block-sdk/v2/lanes/mev"
	testutils "github.com/skip-mev/block-sdk/v2/testutils"
)

func (s *MEVTestSuite) TestPrepareLane() {
	s.Ctx = s.Ctx.WithExecMode(sdk.ExecModePrepareProposal)

	s.Run("can prepare a lane with no txs in mempool", func() {
		lane := s.InitLane(math.LegacyOneDec(), nil)
		proposal := proposals.NewProposal(log.NewNopLogger(), 200, 100)

		proposal, err := lane.PrepareLane(s.Ctx, proposal, block.NoOpPrepareLanesHandler())
		s.Require().NoError(err)
		s.Require().Equal(0, len(proposal.Txs))
		s.Require().Equal(0, len(proposal.Info.TxsByLane))
		s.Require().Equal(int64(0), proposal.Info.BlockSize)
		s.Require().Equal(uint64(0), proposal.Info.GasLimit)
	})

	s.Run("can prepare a lane with a single bid tx in mempool", func() {
		bidTx, _, err := testutils.CreateAuctionTx(
			s.EncCfg.TxConfig,
			s.Accounts[0],
			sdk.NewCoin(s.GasTokenDenom, math.NewInt(100)),
			0,
			0,
			nil,
			100,
		)
		s.Require().NoError(err)
		size := s.getTxSize(bidTx)

		lane := s.InitLane(math.LegacyOneDec(), map[sdk.Tx]bool{bidTx: true})
		s.Require().NoError(lane.Insert(s.Ctx, bidTx))

		proposal := proposals.NewProposal(log.NewNopLogger(), 200, 100)

		proposal, err = lane.PrepareLane(s.Ctx, proposal, block.NoOpPrepareLanesHandler())
		s.Require().NoError(err)
		s.Require().Equal(1, len(proposal.Txs))
		s.Require().Equal(1, len(proposal.Info.TxsByLane))
		s.Require().Equal(size, proposal.Info.BlockSize)
		s.Require().Equal(uint64(100), proposal.Info.GasLimit)

		expectedProposal := []sdk.Tx{bidTx}
		txBzs, err := utils.GetEncodedTxs(s.EncCfg.TxConfig.TxEncoder(), expectedProposal)
		s.Require().NoError(err)
		s.Require().Equal(txBzs[0], proposal.Txs[0])
	})

	s.Run("can prepare a lane with multiple bid txs where highest bid fails", func() {
		bidTx1, _, err := testutils.CreateAuctionTx(
			s.EncCfg.TxConfig,
			s.Accounts[0],
			sdk.NewCoin(s.GasTokenDenom, math.NewInt(100)),
			0,
			0,
			nil,
			100,
		)
		s.Require().NoError(err)

		bidTx2, _, err := testutils.CreateAuctionTx(
			s.EncCfg.TxConfig,
			s.Accounts[1],
			sdk.NewCoin(s.GasTokenDenom, math.NewInt(200)),
			0,
			0,
			nil,
			100,
		)
		s.Require().NoError(err)

		lane := s.InitLane(math.LegacyOneDec(), map[sdk.Tx]bool{bidTx1: true, bidTx2: false})
		s.Require().NoError(lane.Insert(s.Ctx, bidTx1))
		s.Require().NoError(lane.Insert(s.Ctx, bidTx2))

		proposal := proposals.NewProposal(log.NewNopLogger(), 20000, 100000)

		proposal, err = lane.PrepareLane(s.Ctx, proposal, block.NoOpPrepareLanesHandler())
		s.Require().NoError(err)
		s.Require().Equal(1, len(proposal.Txs))
		s.Require().Equal(1, len(proposal.Info.TxsByLane))
		s.Require().Equal(s.getTxSize(bidTx1), proposal.Info.BlockSize)
		s.Require().Equal(uint64(100), proposal.Info.GasLimit)

		expectedProposal := []sdk.Tx{bidTx1}
		txBzs, err := utils.GetEncodedTxs(s.EncCfg.TxConfig.TxEncoder(), expectedProposal)
		s.Require().NoError(err)
		s.Require().Equal(txBzs[0], proposal.Txs[0])
	})

	s.Run("can prepare a lane with multiple bid txs where highest bid passes", func() {
		bidTx1, _, err := testutils.CreateAuctionTx(
			s.EncCfg.TxConfig,
			s.Accounts[0],
			sdk.NewCoin(s.GasTokenDenom, math.NewInt(100)),
			0,
			0,
			nil,
			100,
		)
		s.Require().NoError(err)

		bidTx2, _, err := testutils.CreateAuctionTx(
			s.EncCfg.TxConfig,
			s.Accounts[1],
			sdk.NewCoin(s.GasTokenDenom, math.NewInt(200)),
			0,
			0,
			nil,
			100,
		)
		s.Require().NoError(err)

		lane := s.InitLane(math.LegacyOneDec(), map[sdk.Tx]bool{bidTx1: false, bidTx2: true})
		s.Require().NoError(lane.Insert(s.Ctx, bidTx1))
		s.Require().NoError(lane.Insert(s.Ctx, bidTx2))

		proposal := proposals.NewProposal(log.NewNopLogger(), 20000, 100000)

		proposal, err = lane.PrepareLane(s.Ctx, proposal, block.NoOpPrepareLanesHandler())
		s.Require().NoError(err)
		s.Require().Equal(1, len(proposal.Txs))
		s.Require().Equal(1, len(proposal.Info.TxsByLane))
		s.Require().Equal(s.getTxSize(bidTx2), proposal.Info.BlockSize)
		s.Require().Equal(uint64(100), proposal.Info.GasLimit)

		expectedProposal := []sdk.Tx{bidTx2}
		txBzs, err := utils.GetEncodedTxs(s.EncCfg.TxConfig.TxEncoder(), expectedProposal)
		s.Require().NoError(err)
		s.Require().Equal(txBzs[0], proposal.Txs[0])
	})

	s.Run("can build a proposal with bid tx that has a bundle", func() {
		bidTx, bundle, err := testutils.CreateAuctionTx(
			s.EncCfg.TxConfig,
			s.Accounts[0],
			sdk.NewCoin(s.GasTokenDenom, math.NewInt(100)),
			0,
			0,
			s.Accounts[0:2],
			100,
		)
		s.Require().NoError(err)

		lane := s.InitLane(math.LegacyOneDec(), map[sdk.Tx]bool{bidTx: true, bundle[0]: true, bundle[1]: true})
		s.Require().NoError(lane.Insert(s.Ctx, bidTx))

		proposal := proposals.NewProposal(log.NewNopLogger(), 20000, 100000)

		proposal, err = lane.PrepareLane(s.Ctx, proposal, block.NoOpPrepareLanesHandler())
		s.Require().NoError(err)
		s.Require().Equal(3, len(proposal.Txs))
		s.Require().Equal(1, len(proposal.Info.TxsByLane))
		s.Require().Equal(uint64(3), proposal.Info.TxsByLane[lane.Name()])
		s.Require().Equal(s.getTxSize(bidTx)+s.getTxSize(bundle[0])+s.getTxSize(bundle[1]), proposal.Info.BlockSize)
		s.Require().Equal(uint64(100), proposal.Info.GasLimit)

		expectedProposal := []sdk.Tx{bidTx, bundle[0], bundle[1]}
		txBzs, err := utils.GetEncodedTxs(s.EncCfg.TxConfig.TxEncoder(), expectedProposal)
		s.Require().NoError(err)
		s.Require().Equal(txBzs, proposal.Txs)
	})

	s.Run("can reject a bid that is too large", func() {
		bidTx, bundle, err := testutils.CreateAuctionTx(
			s.EncCfg.TxConfig,
			s.Accounts[0],
			sdk.NewCoin(s.GasTokenDenom, math.NewInt(200)),
			0,
			0,
			s.Accounts[0:2],
			100,
		)
		s.Require().NoError(err)

		lane := s.InitLane(math.LegacyOneDec(), map[sdk.Tx]bool{bidTx: true, bundle[0]: true, bundle[1]: true})
		s.Require().NoError(lane.Insert(s.Ctx, bidTx))

		proposal := proposals.NewProposal(log.NewNopLogger(), s.getTxSize(bidTx), 100000)

		proposal, err = lane.PrepareLane(s.Ctx, proposal, block.NoOpPrepareLanesHandler())
		s.Require().NoError(err)
		s.Require().Equal(0, len(proposal.Txs))
		s.Require().Equal(0, len(proposal.Info.TxsByLane))
		s.Require().Equal(int64(0), proposal.Info.BlockSize)
		s.Require().Equal(uint64(0), proposal.Info.GasLimit)
		s.Require().Equal(0, lane.CountTx())
	})

	s.Run("can reject a bid that is too gas intensive", func() {
		bidTx, _, err := testutils.CreateAuctionTx(
			s.EncCfg.TxConfig,
			s.Accounts[0],
			sdk.NewCoin(s.GasTokenDenom, math.NewInt(200)),
			0,
			0,
			nil,
			100,
		)
		s.Require().NoError(err)

		lane := s.InitLane(math.LegacyOneDec(), map[sdk.Tx]bool{bidTx: true})
		s.Require().NoError(lane.Insert(s.Ctx, bidTx))

		proposal := proposals.NewProposal(log.NewNopLogger(), s.getTxSize(bidTx), 99)

		proposal, err = lane.PrepareLane(s.Ctx, proposal, block.NoOpPrepareLanesHandler())
		s.Require().NoError(err)
		s.Require().Equal(0, len(proposal.Txs))
		s.Require().Equal(0, len(proposal.Info.TxsByLane))
		s.Require().Equal(int64(0), proposal.Info.BlockSize)
		s.Require().Equal(uint64(0), proposal.Info.GasLimit)
		s.Require().Equal(0, lane.CountTx())
	})
}

func (s *MEVTestSuite) TestProcessLane() {
	s.Ctx = s.Ctx.WithExecMode(sdk.ExecModeProcessProposal)

	s.Run("can process an empty proposal", func() {
		lane := s.InitLane(math.LegacyOneDec(), nil)
		proposal := proposals.NewProposal(log.NewNopLogger(), 200, 100)

		txsFromLane, remainingTxs, err := mev.NewProposalHandler(lane.BaseLane, lane.Factory).ProcessLaneHandler()(s.Ctx, nil)
		s.Require().NoError(err)
		s.Require().Equal(0, len(txsFromLane))
		s.Require().Equal(0, len(remainingTxs))

		proposal, err = lane.ProcessLane(s.Ctx, proposal, nil, block.NoOpProcessLanesHandler())
		s.Require().NoError(err)
		s.Require().Equal(0, len(proposal.Txs))
	})

	s.Run("can process a proposal with tx that does not belong to this lane", func() {
		tx, err := testutils.CreateRandomTx(s.EncCfg.TxConfig, s.Accounts[0], 0, 1, 0, 100)
		s.Require().NoError(err)

		lane := s.InitLane(math.LegacyOneDec(), nil)
		proposal := proposals.NewProposal(log.NewNopLogger(), 200, 100)

		txsFromLane, remainingTxs, err := mev.NewProposalHandler(lane.BaseLane, lane.Factory).ProcessLaneHandler()(s.Ctx, []sdk.Tx{tx})
		s.Require().NoError(err)
		s.Require().Equal(0, len(txsFromLane))
		s.Require().Equal(1, len(remainingTxs))

		finalProposal, err := lane.ProcessLane(s.Ctx, proposal, []sdk.Tx{tx}, block.NoOpProcessLanesHandler())
		s.Require().NoError(err)
		s.Require().Equal(0, len(finalProposal.Txs))
	})

	s.Run("can process a proposal with bad bid tx", func() {
		bidTx, _, err := testutils.CreateAuctionTx(
			s.EncCfg.TxConfig,
			s.Accounts[0],
			sdk.NewCoin(s.GasTokenDenom, math.NewInt(100)),
			0,
			0,
			nil,
			100,
		)
		s.Require().NoError(err)

		partialProposal := []sdk.Tx{bidTx}

		lane := s.InitLane(math.LegacyOneDec(), map[sdk.Tx]bool{bidTx: false})

		txsFromLane, remainingTxs, err := mev.NewProposalHandler(lane.BaseLane, lane.Factory).ProcessLaneHandler()(s.Ctx, partialProposal)
		s.Require().Error(err)
		s.Require().Equal(0, len(txsFromLane))
		s.Require().Equal(0, len(remainingTxs))

		proposal := proposals.NewProposal(log.NewNopLogger(), 200000, 1000000)
		_, err = lane.ProcessLane(s.Ctx, proposal, partialProposal, block.NoOpProcessLanesHandler())
		s.Require().Error(err)
	})

	s.Run("can process a proposal with a bad bundled tx", func() {
		bidTx, bundle, err := testutils.CreateAuctionTx(
			s.EncCfg.TxConfig,
			s.Accounts[0],
			sdk.NewCoin(s.GasTokenDenom, math.NewInt(100)),
			0,
			0,
			s.Accounts[0:2],
			100,
		)
		s.Require().NoError(err)

		partialProposal := []sdk.Tx{bidTx, bundle[0], bundle[1]}

		lane := s.InitLane(math.LegacyOneDec(), map[sdk.Tx]bool{bidTx: true, bundle[0]: true, bundle[1]: false})

		txsFromLane, remainingTxs, err := mev.NewProposalHandler(lane.BaseLane, lane.Factory).ProcessLaneHandler()(s.Ctx, partialProposal)
		s.Require().Error(err)
		s.Require().Equal(0, len(txsFromLane))
		s.Require().Equal(0, len(remainingTxs))

		proposal := proposals.NewProposal(log.NewNopLogger(), 200000, 1000000)
		_, err = lane.ProcessLane(s.Ctx, proposal, partialProposal, block.NoOpProcessLanesHandler())
		s.Require().Error(err)
	})

	s.Run("can process a proposal with mismatching txs in bundle", func() {
		bidTx, bundle, err := testutils.CreateAuctionTx(
			s.EncCfg.TxConfig,
			s.Accounts[0],
			sdk.NewCoin(s.GasTokenDenom, math.NewInt(100)),
			0,
			0,
			s.Accounts[0:2],
			100,
		)
		s.Require().NoError(err)

		partialProposal := []sdk.Tx{bidTx, bundle[1], bundle[0]}

		lane := s.InitLane(math.LegacyOneDec(), map[sdk.Tx]bool{bidTx: true, bundle[0]: true, bundle[1]: true})

		txsFromLane, remainingTxs, err := mev.NewProposalHandler(lane.BaseLane, lane.Factory).ProcessLaneHandler()(s.Ctx, partialProposal)
		s.Require().Error(err)
		s.Require().Equal(0, len(txsFromLane))
		s.Require().Equal(0, len(remainingTxs))

		proposal := proposals.NewProposal(log.NewNopLogger(), 200000, 1000000)
		_, err = lane.ProcessLane(s.Ctx, proposal, partialProposal, block.NoOpProcessLanesHandler())
		s.Require().Error(err)
	})

	s.Run("can process a proposal with missing bundle tx", func() {
		bidTx, bundle, err := testutils.CreateAuctionTx(
			s.EncCfg.TxConfig,
			s.Accounts[0],
			sdk.NewCoin(s.GasTokenDenom, math.NewInt(100)),
			0,
			0,
			s.Accounts[0:2],
			100,
		)
		s.Require().NoError(err)

		partialProposal := []sdk.Tx{bidTx, bundle[0]}

		lane := s.InitLane(math.LegacyOneDec(), map[sdk.Tx]bool{bidTx: true, bundle[0]: true})

		txsFromLane, remainingTxs, err := mev.NewProposalHandler(lane.BaseLane, lane.Factory).ProcessLaneHandler()(s.Ctx, partialProposal)
		s.Require().Error(err)
		s.Require().Equal(0, len(txsFromLane))
		s.Require().Equal(0, len(remainingTxs))

		proposal := proposals.NewProposal(log.NewNopLogger(), 200000, 1000000)
		_, err = lane.ProcessLane(s.Ctx, proposal, partialProposal, block.NoOpProcessLanesHandler())
		s.Require().Error(err)
	})

	s.Run("can process a valid proposal", func() {
		bidTx, bundle, err := testutils.CreateAuctionTx(
			s.EncCfg.TxConfig,
			s.Accounts[0],
			sdk.NewCoin(s.GasTokenDenom, math.NewInt(100)),
			0,
			0,
			s.Accounts[0:2],
			100,
		)
		s.Require().NoError(err)

		partialProposal := []sdk.Tx{bidTx, bundle[0], bundle[1]}

		lane := s.InitLane(math.LegacyOneDec(), map[sdk.Tx]bool{bidTx: true, bundle[0]: true, bundle[1]: true})

		txsFromLane, remainingTxs, err := mev.NewProposalHandler(lane.BaseLane, lane.Factory).ProcessLaneHandler()(s.Ctx, partialProposal)
		s.Require().NoError(err)
		s.Require().Equal(3, len(txsFromLane))
		s.Require().Equal(0, len(remainingTxs))

		proposal := proposals.NewProposal(log.NewNopLogger(), 200000, 1000000)
		_, err = lane.ProcessLane(s.Ctx, proposal, partialProposal, block.NoOpProcessLanesHandler())
		s.Require().NoError(err)
	})

	s.Run("can process a valid proposal with a single bid with no bundle", func() {
		bidTx, _, err := testutils.CreateAuctionTx(
			s.EncCfg.TxConfig,
			s.Accounts[0],
			sdk.NewCoin("stake", math.NewInt(100)),
			0,
			0,
			nil,
			100,
		)
		s.Require().NoError(err)

		partialProposal := []sdk.Tx{bidTx}

		lane := s.InitLane(math.LegacyOneDec(), map[sdk.Tx]bool{bidTx: true})

		txsFromLane, remainingTxs, err := mev.NewProposalHandler(lane.BaseLane, lane.Factory).ProcessLaneHandler()(s.Ctx, partialProposal)
		s.Require().NoError(err)
		s.Require().Equal(1, len(txsFromLane))
		s.Require().Equal(0, len(remainingTxs))

		proposal := proposals.NewProposal(log.NewNopLogger(), 200000, 1000000)
		_, err = lane.ProcessLane(s.Ctx, proposal, partialProposal, block.NoOpProcessLanesHandler())
		s.Require().NoError(err)
	})

	s.Run("can reject a block proposal that exceeds its gas limit", func() {
		bidTx, bundle, err := testutils.CreateAuctionTx(
			s.EncCfg.TxConfig,
			s.Accounts[0],
			sdk.NewCoin("stake", math.NewInt(100)),
			0,
			0,
			s.Accounts[0:2],
			100,
		)
		s.Require().NoError(err)

		partialProposal := []sdk.Tx{bidTx, bundle[0], bundle[1]}

		lane := s.InitLane(math.LegacyOneDec(), map[sdk.Tx]bool{bidTx: true, bundle[0]: true, bundle[1]: true})

		txsFromLane, remainingTxs, err := mev.NewProposalHandler(lane.BaseLane, lane.Factory).ProcessLaneHandler()(s.Ctx, partialProposal)
		s.Require().NoError(err)
		s.Require().Equal(3, len(txsFromLane))
		s.Require().Equal(0, len(remainingTxs))

		proposal := proposals.NewProposal(log.NewNopLogger(), 20000, 99)
		_, err = lane.ProcessLane(s.Ctx, proposal, partialProposal, block.NoOpProcessLanesHandler())
		s.Require().Error(err)
	})

	s.Run("can reject a block proposal that exceeds its block size", func() {
		bidTx, bundle, err := testutils.CreateAuctionTx(
			s.EncCfg.TxConfig,
			s.Accounts[0],
			sdk.NewCoin("stake", math.NewInt(100)),
			0,
			0,
			s.Accounts[0:2],
			100,
		)
		s.Require().NoError(err)

		partialProposal := []sdk.Tx{bidTx, bundle[0], bundle[1]}

		lane := s.InitLane(math.LegacyOneDec(), map[sdk.Tx]bool{bidTx: true, bundle[0]: true, bundle[1]: true})

		txsFromLane, remainingTxs, err := mev.NewProposalHandler(lane.BaseLane, lane.Factory).ProcessLaneHandler()(s.Ctx, partialProposal)
		s.Require().NoError(err)
		s.Require().Equal(3, len(txsFromLane))
		s.Require().Equal(0, len(remainingTxs))

		proposal := proposals.NewProposal(log.NewNopLogger(), 200, 100)
		_, err = lane.ProcessLane(s.Ctx, proposal, partialProposal, block.NoOpProcessLanesHandler())
		s.Require().Error(err)
	})

	s.Run("can accept a block proposal with bid and other txs", func() {
		bidTx, bundle, err := testutils.CreateAuctionTx(
			s.EncCfg.TxConfig,
			s.Accounts[0],
			sdk.NewCoin("stake", math.NewInt(100)),
			0,
			0,
			s.Accounts[0:2],
			100,
		)
		s.Require().NoError(err)

		otherTx, err := testutils.CreateRandomTx(s.EncCfg.TxConfig, s.Accounts[0], 0, 1, 0, 100)
		s.Require().NoError(err)

		partialProposal := []sdk.Tx{bidTx, bundle[0], bundle[1], otherTx}

		lane := s.InitLane(math.LegacyOneDec(), map[sdk.Tx]bool{bidTx: true, bundle[0]: true, bundle[1]: true})

		txsFromLane, remainingTxs, err := mev.NewProposalHandler(lane.BaseLane, lane.Factory).ProcessLaneHandler()(s.Ctx, partialProposal)
		s.Require().NoError(err)
		s.Require().Equal(3, len(txsFromLane))
		s.Require().Equal(1, len(remainingTxs))
		s.Require().Equal(otherTx, remainingTxs[0])

		proposal := proposals.NewProposal(log.NewNopLogger(), 200000, 1000000)
		proposal, err = lane.ProcessLane(s.Ctx, proposal, partialProposal, block.NoOpProcessLanesHandler())
		s.Require().NoError(err)
		s.Require().Len(proposal.Txs, 3)

		encodedTxs, err := utils.GetEncodedTxs(s.EncCfg.TxConfig.TxEncoder(), []sdk.Tx{bidTx, bundle[0], bundle[1]})
		s.Require().NoError(err)
		s.Require().Equal(encodedTxs, proposal.Txs)
	})

	s.Run("rejects a block where the bid tx is not the first tx", func() {
		bidTx, bundle, err := testutils.CreateAuctionTx(
			s.EncCfg.TxConfig,
			s.Accounts[0],
			sdk.NewCoin("stake", math.NewInt(100)),
			0,
			0,
			s.Accounts[0:2],
			100,
		)
		s.Require().NoError(err)

		otherTx, err := testutils.CreateRandomTx(s.EncCfg.TxConfig, s.Accounts[0], 0, 1, 0, 100)
		s.Require().NoError(err)

		partialProposal := []sdk.Tx{otherTx, bidTx, bundle[0], bundle[1]}

		lane := s.InitLane(math.LegacyOneDec(), map[sdk.Tx]bool{bidTx: true, bundle[0]: true, bundle[1]: true})

		txsFromLane, remainingTxs, err := mev.NewProposalHandler(lane.BaseLane, lane.Factory).ProcessLaneHandler()(s.Ctx, partialProposal)
		s.Require().Error(err)
		s.Require().Equal(0, len(txsFromLane))
		s.Require().Equal(0, len(remainingTxs))

		proposal := proposals.NewProposal(log.NewNopLogger(), 200000, 1000000)
		_, err = lane.ProcessLane(s.Ctx, proposal, partialProposal, block.NoOpProcessLanesHandler())
		s.Require().Error(err)
	})
}

func (s *MEVTestSuite) TestVerifyBidBasic() {
	lane := s.InitLane(math.LegacyOneDec(), nil)
	proposal := proposals.NewProposal(log.NewNopLogger(), 200, 100)
	limits := proposal.GetLaneLimits(lane.GetMaxBlockSpace())

	handler := mev.NewProposalHandler(lane.BaseLane, lane.Factory)

	s.Run("can verify a bid with no bundled txs", func() {
		bidTx, expectedBundle, err := testutils.CreateAuctionTx(
			s.EncCfg.TxConfig,
			s.Accounts[0],
			sdk.NewCoin(s.GasTokenDenom, math.NewInt(100)),
			0,
			0,
			nil,
			100,
		)
		s.Require().NoError(err)

		bundle, err := handler.VerifyBidBasic(s.Ctx, bidTx, proposal, limits)
		s.Require().NoError(err)
		s.compare(bundle, expectedBundle)
	})

	s.Run("can reject a tx that is not a bid", func() {
		tx, err := testutils.CreateRandomTx(
			s.EncCfg.TxConfig,
			s.Accounts[0],
			0,
			1,
			0,
			100,
		)
		s.Require().NoError(err)

		_, err = handler.VerifyBidBasic(s.Ctx, tx, proposal, limits)
		s.Require().Error(err)
	})

	s.Run("can reject a bundle that is too gas intensive", func() {
		bidTx, _, err := testutils.CreateAuctionTx(
			s.EncCfg.TxConfig,
			s.Accounts[0],
			sdk.NewCoin(s.GasTokenDenom, math.NewInt(100)),
			0,
			0,
			nil,
			101,
		)
		s.Require().NoError(err)

		_, err = handler.VerifyBidBasic(s.Ctx, bidTx, proposal, limits)
		s.Require().Error(err)
	})

	s.Run("can reject a bundle that is too large", func() {
		bidTx, _, err := testutils.CreateAuctionTx(
			s.EncCfg.TxConfig,
			s.Accounts[0],
			sdk.NewCoin(s.GasTokenDenom, math.NewInt(100)),
			0,
			0,
			s.Accounts[0:2],
			100,
		)
		s.Require().NoError(err)

		size := s.getTxSize(bidTx)
		proposal := proposals.NewProposal(log.NewNopLogger(), size-1, 100)
		limits := proposal.GetLaneLimits(lane.GetMaxBlockSpace())

		_, err = handler.VerifyBidBasic(s.Ctx, bidTx, proposal, limits)
		s.Require().Error(err)
	})

	s.Run("can reject a bundle with malformed txs", func() {
		bidMsg, err := testutils.CreateMsgAuctionBid(
			s.EncCfg.TxConfig,
			s.Accounts[0],
			sdk.NewCoin(s.GasTokenDenom, math.NewInt(100)),
			0,
			3,
		)
		s.Require().NoError(err)

		bidMsg.Transactions[2] = []byte("invalid")

		bidTx, err := testutils.CreateTx(
			s.EncCfg.TxConfig,
			s.Accounts[0],
			0,
			0,
			[]sdk.Msg{bidMsg},
		)
		s.Require().NoError(err)

		_, err = handler.VerifyBidBasic(s.Ctx, bidTx, proposal, limits)
		s.Require().Error(err)
	})
}

func (s *MEVTestSuite) TestVerifyBidTx() {
	s.Run("can verify a valid bid", func() {
		bidTx, bundle, err := testutils.CreateAuctionTx(
			s.EncCfg.TxConfig,
			s.Accounts[0],
			sdk.NewCoin(s.GasTokenDenom, math.NewInt(100)),
			0,
			0,
			nil,
			100,
		)
		s.Require().NoError(err)

		lane := s.InitLane(math.LegacyOneDec(), map[sdk.Tx]bool{bidTx: true})

		handler := mev.NewProposalHandler(lane.BaseLane, lane.Factory)
		s.Require().NoError(handler.VerifyBidTx(s.Ctx, bidTx, bundle))
	})

	s.Run("can reject a bid transaction", func() {
		bidTx, bundle, err := testutils.CreateAuctionTx(
			s.EncCfg.TxConfig,
			s.Accounts[0],
			sdk.NewCoin(s.GasTokenDenom, math.NewInt(100)),
			0,
			0,
			nil,
			100,
		)
		s.Require().NoError(err)

		lane := s.InitLane(math.LegacyOneDec(), map[sdk.Tx]bool{bidTx: false})

		handler := mev.NewProposalHandler(lane.BaseLane, lane.Factory)
		s.Require().Error(handler.VerifyBidTx(s.Ctx, bidTx, bundle))
	})

	s.Run("can reject a bid transaction with a bad bundle", func() {
		bidTx, bundle, err := testutils.CreateAuctionTx(
			s.EncCfg.TxConfig,
			s.Accounts[0],
			sdk.NewCoin(s.GasTokenDenom, math.NewInt(100)),
			0,
			0,
			s.Accounts[0:2],
			100,
		)
		s.Require().NoError(err)

		lane := s.InitLane(math.LegacyOneDec(), map[sdk.Tx]bool{bidTx: true, bundle[0]: true, bundle[1]: false})

		handler := mev.NewProposalHandler(lane.BaseLane, lane.Factory)
		s.Require().Error(handler.VerifyBidTx(s.Ctx, bidTx, bundle))
	})

	s.Run("can reject a bid transaction with a bundle that has another bid tx", func() {
		bidTx, bundle, err := testutils.CreateAuctionTx(
			s.EncCfg.TxConfig,
			s.Accounts[0],
			sdk.NewCoin(s.GasTokenDenom, math.NewInt(100)),
			0,
			0,
			s.Accounts[0:2],
			100,
		)
		s.Require().NoError(err)

		otherBidTx, _, err := testutils.CreateAuctionTx(
			s.EncCfg.TxConfig,
			s.Accounts[0],
			sdk.NewCoin(s.GasTokenDenom, math.NewInt(100)),
			0,
			0,
			nil,
			100,
		)
		s.Require().NoError(err)
		bundle = append(bundle, otherBidTx)

		lane := s.InitLane(math.LegacyOneDec(), map[sdk.Tx]bool{bidTx: true, bundle[0]: true, bundle[1]: true, otherBidTx: true})

		handler := mev.NewProposalHandler(lane.BaseLane, lane.Factory)
		s.Require().Error(handler.VerifyBidTx(s.Ctx, bidTx, bundle))
	})
}
