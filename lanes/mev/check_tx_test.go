package mev_test

import (
	"fmt"

	"cosmossdk.io/log"
	"cosmossdk.io/math"
	"cosmossdk.io/store"
	storetypes "cosmossdk.io/store/types"
	cometabci "github.com/cometbft/cometbft/abci/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	db "github.com/cosmos/cosmos-db"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/skip-mev/block-sdk/block"
	"github.com/skip-mev/block-sdk/lanes/mev"
	"github.com/skip-mev/block-sdk/testutils"
	blocksdktypes "github.com/skip-mev/block-sdk/x/blocksdk/types"
)

func (s *MEVTestSuite) TestCheckTx() {
	bidTx, _, err := testutils.CreateAuctionTx(
		s.encCfg.TxConfig,
		s.accounts[0],
		sdk.NewCoin(s.gasTokenDenom, math.NewInt(100)),
		0,
		0,
		nil,
		100,
	)
	s.Require().NoError(err)

	// create a tx that should not be inserted in the mev-lane
	bidTx2, _, err := testutils.CreateAuctionTx(
		s.encCfg.TxConfig,
		s.accounts[0],
		sdk.NewCoin(s.gasTokenDenom, math.NewInt(100)),
		1,
		0,
		nil,
		100,
	)
	s.Require().NoError(err)

	txs := map[sdk.Tx]bool{
		bidTx: true,
	}

	mevLane := s.initLane(math.LegacyOneDec(), txs)
	mempool, err := block.NewLanedMempool(s.ctx.Logger(), []block.Lane{mevLane}, moduleLaneFetcher{
		mevLane,
	})
	s.Require().NoError(err)

	handler := mev.NewCheckTxHandler(
		&baseApp{
			s.ctx,
		},
		s.encCfg.TxConfig.TxDecoder(),
		mevLane,
		mempool,
		s.setUpAnteHandler(txs),
	).CheckTx()

	// test that a bid can be successfully inserted to mev-lane on CheckTx
	s.Run("test bid insertion on CheckTx", func() {
		txBz, err := s.encCfg.TxConfig.TxEncoder()(bidTx)
		s.Require().NoError(err)

		// check tx
		res, err := handler(&cometabci.RequestCheckTx{Tx: txBz, Type: cometabci.CheckTxType_New})
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
		txBz, err := s.encCfg.TxConfig.TxEncoder()(bidTx2)
		s.Require().NoError(err)

		res, err := handler(&cometabci.RequestCheckTx{Tx: txBz, Type: cometabci.CheckTxType_Recheck})
		s.Require().NoError(err)

		s.Require().Equal(uint32(1), res.Code)
	})
}

type baseApp struct {
	ctx sdk.Context
}

// CommitMultiStore is utilized to retrieve the latest committed state.
func (ba *baseApp) CommitMultiStore() storetypes.CommitMultiStore {
	db := db.NewMemDB()
	return store.NewCommitMultiStore(db, ba.ctx.Logger(), nil)
}

// CheckTx is baseapp's CheckTx method that checks the validity of a
// transaction.
func (baseApp) CheckTx(_ *cometabci.RequestCheckTx) (*cometabci.ResponseCheckTx, error) {
	return nil, fmt.Errorf("not implemented")
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
func (baseApp) GetConsensusParams(ctx sdk.Context) cmtproto.ConsensusParams {
	return ctx.ConsensusParams()
}

// ChainID is utilized to retrieve the chain ID.
func (ba *baseApp) ChainID() string {
	return ba.ctx.ChainID()
}

type moduleLaneFetcher struct {
	lane *mev.MEVLane
}

func (mlf moduleLaneFetcher) GetLane(sdk.Context, string) (lane blocksdktypes.Lane, err error) {
	return blocksdktypes.Lane{}, nil
}

func (mlf moduleLaneFetcher) GetLanes(sdk.Context) []blocksdktypes.Lane {
	return nil
}
