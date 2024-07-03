package checktx_test

import (
	"github.com/cosmos/cosmos-sdk/store"
	"testing"

	"cosmossdk.io/math"
	db "github.com/cometbft/cometbft-db"
	cometabci "github.com/cometbft/cometbft/abci/types"
	"github.com/cometbft/cometbft/libs/log"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/suite"

	"github.com/skip-mev/block-sdk/abci/checktx"
	"github.com/skip-mev/block-sdk/block"
	"github.com/skip-mev/block-sdk/block/utils"
	mevlanetestutils "github.com/skip-mev/block-sdk/lanes/mev/testutils"
	"github.com/skip-mev/block-sdk/testutils"
	auctiontypes "github.com/skip-mev/block-sdk/x/auction/types"
)

type CheckTxTestSuite struct {
	mevlanetestutils.MEVLaneTestSuiteBase
}

func TestCheckTxTestSuite(t *testing.T) {
	suite.Run(t, new(CheckTxTestSuite))
}

func (s *CheckTxTestSuite) TestCheckTxMempoolParity() {
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

	hugeBidTx, _, err := testutils.CreateNAuctionTx(
		s.EncCfg.TxConfig,
		s.Accounts[0],
		sdk.NewCoin(s.GasTokenDenom, math.NewInt(100)),
		0,
		0,
		[]testutils.Account{s.Accounts[0]},
		100,
		100000,
	)
	s.Require().NoError(err)

	// create a tx that should not be inserted in the mev-lane
	bidTx2, _, err := testutils.CreateAuctionTx(
		s.EncCfg.TxConfig,
		s.Accounts[0],
		sdk.NewCoin(s.GasTokenDenom, math.NewInt(100)),
		1,
		0,
		nil,
		100,
	)
	s.Require().NoError(err)

	txs := map[sdk.Tx]bool{
		bidTx:     true,
		hugeBidTx: true,
	}

	mevLane := s.InitLane(math.LegacyOneDec(), txs, true)
	mempool, err := block.NewLanedMempool(s.Ctx.Logger(), []block.Lane{mevLane})
	s.Require().NoError(err)

	cacheDecoder, err := utils.NewDefaultCacheTxDecoder(s.EncCfg.TxConfig.TxDecoder())
	s.Require().NoError(err)

	ba := &baseApp{
		s.Ctx,
	}

	mevLaneHandler := checktx.NewMEVCheckTxHandler(
		ba,
		cacheDecoder.TxDecoder(),
		mevLane,
		s.SetUpAnteHandler(txs),
		ba.CheckTx,
		s.Ctx.ChainID(),
	).CheckTx()

	handler := checktx.NewMempoolParityCheckTx(
		s.Ctx.Logger(),
		mempool,
		cacheDecoder.TxDecoder(),
		mevLaneHandler,
		ba,
	).CheckTx()

	// test that a bid can be successfully inserted to mev-lane on CheckTx
	s.Run("test bid insertion on CheckTx", func() {
		txBz, err := s.EncCfg.TxConfig.TxEncoder()(bidTx)
		s.Require().NoError(err)

		// check tx
		res := handler(cometabci.RequestCheckTx{Tx: txBz, Type: cometabci.CheckTxType_New})

		s.Require().Equal(uint32(0), res.Code)

		// check that the mev-lane contains the bid
		s.Require().True(mevLane.Contains(bidTx))
	})

	// test that a bid will fail to be inserted as it is too large
	s.Run("test bid insertion failure on CheckTx - too large", func() {
		txBz, err := s.EncCfg.TxConfig.TxEncoder()(hugeBidTx)
		s.Require().NoError(err)

		// check tx
		res := handler(cometabci.RequestCheckTx{Tx: txBz, Type: cometabci.CheckTxType_New})

		s.Require().Equal(uint32(1), res.Code)

		// check that the mev-lane does not contain the bid
		s.Require().False(mevLane.Contains(bidTx))
	})

	// test that a bid can be successfully inserted to mev-lane on CheckTx
	s.Run("test bid insertion on CheckTx", func() {
		txBz, err := s.EncCfg.TxConfig.TxEncoder()(bidTx)
		s.Require().NoError(err)

		// check tx
		res := handler(cometabci.RequestCheckTx{Tx: txBz, Type: cometabci.CheckTxType_New})
		s.Require().NoError(err)

		s.Require().Equal(uint32(0), res.Code)

		// check that the mev-lane contains the bid
		s.Require().True(mevLane.Contains(bidTx))
	})

	// test that a bid-tx (not in mev-lane) can be removed from the mempool on ReCheck
	s.Run("test bid removal on ReCheckTx", func() {
		// assert that the mev-lane does not contain the bidTx2
		s.Require().False(mevLane.Contains(bidTx2))

		// check tx
		txBz, err := s.EncCfg.TxConfig.TxEncoder()(bidTx2)
		s.Require().NoError(err)

		res := handler(cometabci.RequestCheckTx{Tx: txBz, Type: cometabci.CheckTxType_Recheck})

		s.Require().Equal(uint32(1), res.Code)
	})
}

func (s *CheckTxTestSuite) TestRemovalOnRecheckTx() {
	// create a tx that should not be inserted in the mev-lane
	tx, _, err := testutils.CreateAuctionTx(
		s.EncCfg.TxConfig,
		s.Accounts[0],
		sdk.NewCoin(s.GasTokenDenom, math.NewInt(100)),
		1,
		0,
		nil,
		100,
	)
	s.Require().NoError(err)

	mevLane := s.InitLane(math.LegacyOneDec(), nil, true)
	mempool, err := block.NewLanedMempool(s.Ctx.Logger(), []block.Lane{mevLane})
	s.Require().NoError(err)

	cacheDecoder, err := utils.NewDefaultCacheTxDecoder(s.EncCfg.TxConfig.TxDecoder())
	s.Require().NoError(err)

	ba := &baseApp{
		s.Ctx,
	}

	handler := checktx.NewMempoolParityCheckTx(
		s.Ctx.Logger(),
		mempool,
		cacheDecoder.TxDecoder(),
		func(cometabci.RequestCheckTx) cometabci.ResponseCheckTx {
			// always fail
			return cometabci.ResponseCheckTx{Code: 1}
		},
		ba,
	).CheckTx()

	s.Run("tx is removed on check-tx failure when re-check", func() {
		// check that tx exists in mempool
		txBz, err := s.EncCfg.TxConfig.TxEncoder()(tx)
		s.Require().NoError(err)

		s.Require().NoError(mempool.Insert(s.Ctx, tx))
		s.Require().True(mempool.Contains(tx))

		// check tx
		res := handler(cometabci.RequestCheckTx{Tx: txBz, Type: cometabci.CheckTxType_Recheck})
		s.Require().NoError(err)

		s.Require().Equal(uint32(1), res.Code)

		// check that tx is removed from mempool
		s.Require().False(mempool.Contains(tx))
	})
}

func (s *CheckTxTestSuite) TestMempoolParityCheckTx() {
	s.Run("tx fails tx-decoding", func() {
		cacheDecoder, err := utils.NewDefaultCacheTxDecoder(s.EncCfg.TxConfig.TxDecoder())
		s.Require().NoError(err)

		ba := &baseApp{
			s.Ctx,
		}

		handler := checktx.NewMempoolParityCheckTx(
			s.Ctx.Logger(),
			nil,
			cacheDecoder.TxDecoder(),
			nil,
			ba,
		)

		res := handler.CheckTx()(cometabci.RequestCheckTx{Tx: []byte("invalid-tx")})

		s.Require().Equal(uint32(1), res.Code)
	})
}

func (s *CheckTxTestSuite) TestMEVCheckTxHandler() {
	txs := map[sdk.Tx]bool{}

	mevLane := s.InitLane(math.LegacyOneDec(), txs, true)
	mempool, err := block.NewLanedMempool(s.Ctx.Logger(), []block.Lane{mevLane})
	s.Require().NoError(err)

	ba := &baseApp{
		s.Ctx,
	}

	acc := s.Accounts[0]
	// create a tx that should not be inserted in the mev-lane
	normalTx, err := testutils.CreateRandomTxBz(s.EncCfg.TxConfig, acc, 0, 1, 0, 0)
	s.Require().NoError(err)

	cacheDecoder, err := utils.NewDefaultCacheTxDecoder(s.EncCfg.TxConfig.TxDecoder())
	s.Require().NoError(err)

	var gotTx []byte
	mevLaneHandler := checktx.NewMEVCheckTxHandler(
		ba,
		cacheDecoder.TxDecoder(),
		mevLane,
		s.SetUpAnteHandler(txs),
		func(req cometabci.RequestCheckTx) cometabci.ResponseCheckTx {
			// expect the above free tx to be sent here
			gotTx = req.Tx
			return cometabci.ResponseCheckTx{
				Code: uint32(0),
			}
		},
		s.Ctx.ChainID(),
	).CheckTx()

	handler := checktx.NewMempoolParityCheckTx(
		s.Ctx.Logger(),
		mempool,
		cacheDecoder.TxDecoder(),
		mevLaneHandler,
		ba,
	).CheckTx()

	// test that a normal tx can be successfully inserted to the mempool
	s.Run("test non-mev tx insertion on CheckTx", func() {
		res := handler(cometabci.RequestCheckTx{Tx: normalTx, Type: cometabci.CheckTxType_New})

		s.Require().Equal(uint32(0), res.Code)
		s.Require().Equal(normalTx, gotTx)
	})
}

func (s *CheckTxTestSuite) TestValidateBidTx() {
	validBidTx, bundled, err := testutils.CreateAuctionTx(
		s.EncCfg.TxConfig,
		s.Accounts[0],
		sdk.NewCoin(s.GasTokenDenom, math.NewInt(100)),
		0,
		0,
		[]testutils.Account{s.Accounts[0]},
		100,
	)
	s.Require().NoError(err)

	txBz, err := s.EncCfg.TxConfig.TxEncoder()(validBidTx)
	s.Require().NoError(err)

	// create an invalid bid-tx (nested)
	bidMsg := auctiontypes.NewMsgAuctionBid(s.Accounts[0].Address, sdk.NewCoin(s.GasTokenDenom, math.NewInt(100)), [][]byte{
		txBz,
	})
	nestedBidTx, err := testutils.CreateTx(
		s.EncCfg.TxConfig,
		s.Accounts[0],
		0,
		0,
		[]sdk.Msg{bidMsg},
	)
	s.Require().NoError(err)

	// create an invalid bid-tx (signer invalid)
	invalidBidMsg := auctiontypes.MsgAuctionBid{
		Bidder:       "",
		Bid:          sdk.NewCoin(s.GasTokenDenom, math.NewInt(100)),
		Transactions: nil,
	}
	invalidBidTx, err := testutils.CreateTx(
		s.EncCfg.TxConfig,
		s.Accounts[0],
		0,
		0,
		[]sdk.Msg{&invalidBidMsg},
	)
	s.Require().NoError(err)

	// create a tx that should not be inserted in the mev-lane
	s.Require().NoError(err)

	txs := map[sdk.Tx]bool{
		validBidTx:   true,
		bundled[0]:   true,
		nestedBidTx:  true,
		invalidBidTx: true,
	}

	mevLane := s.InitLane(math.LegacyOneDec(), txs, true)

	cacheDecoder, err := utils.NewDefaultCacheTxDecoder(s.EncCfg.TxConfig.TxDecoder())
	s.Require().NoError(err)

	ba := &baseApp{
		s.Ctx,
	}
	mevLaneHandler := checktx.NewMEVCheckTxHandler(
		ba,
		cacheDecoder.TxDecoder(),
		mevLane,
		s.SetUpAnteHandler(txs),
		ba.CheckTx,
		s.Ctx.ChainID(),
	)
	s.Run("expected bid-tx", func() {
		bundledTx, err := s.EncCfg.TxConfig.TxEncoder()(bundled[0])
		s.Require().NoError(err)

		_, err = mevLaneHandler.ValidateBidTx(s.Ctx, validBidTx, &auctiontypes.BidInfo{
			Transactions: [][]byte{bundledTx},
		})
		s.Require().NoError(err)
	})

	s.Run("nested bid-tx", func() {
		nestedBidTxBz, err := s.EncCfg.TxConfig.TxEncoder()(nestedBidTx)
		s.Require().NoError(err)

		_, err = mevLaneHandler.ValidateBidTx(s.Ctx, nestedBidTx, &auctiontypes.BidInfo{
			Transactions: [][]byte{nestedBidTxBz},
		})
		s.Require().Error(err)
		s.Require().Contains(err.Error(), "bundled tx cannot be a bid tx")
	})

	s.Run("invalid bid-tx", func() {
		invalidBidTxBz, err := s.EncCfg.TxConfig.TxEncoder()(invalidBidTx)
		s.Require().NoError(err)

		_, err = mevLaneHandler.ValidateBidTx(s.Ctx, invalidBidTx, &auctiontypes.BidInfo{
			Transactions: [][]byte{invalidBidTxBz},
		})
		s.Require().Error(err)
		s.Require().Contains(err.Error(), "failed to get bid info")
	})
}

type baseApp struct {
	ctx sdk.Context
}

// CommitMultiStore is utilized to retrieve the latest committed state.
func (ba *baseApp) CommitMultiStore() storetypes.CommitMultiStore {
	db := db.NewMemDB()
	return store.NewCommitMultiStore(db)
}

// CheckTx is baseapp's CheckTx method that checks the validity of a
// transaction.
func (baseApp) CheckTx(_ cometabci.RequestCheckTx) cometabci.ResponseCheckTx {
	return cometabci.ResponseCheckTx{}
}

// Logger is utilized to log errors.
func (ba *baseApp) Logger() log.Logger {
	return ba.ctx.Logger()
}

// LastBlockHeight is utilized to retrieve the latest block height.
func (ba *baseApp) LastBlockHeight() int64 {
	return ba.ctx.BlockHeight()
}

// GetConsensusParams is utilized to retrieve the consensus params.
func (baseApp) GetConsensusParams(ctx sdk.Context) *cmtproto.ConsensusParams {
	return &cmtproto.ConsensusParams{
		Block: &cmtproto.BlockParams{
			MaxBytes: 10000,
			MaxGas:   10000,
		},
		Evidence:  nil,
		Validator: nil,
		Version:   nil,
	}
}
