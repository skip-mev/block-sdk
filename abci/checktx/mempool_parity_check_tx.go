package checktx

import (
	"fmt"

	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"

	"cosmossdk.io/log"

	cmtabci "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	"github.com/skip-mev/block-sdk/v2/block"
)

// MempoolParityCheckTx is a CheckTx function that evicts txs that are not in the app-side mempool
// on ReCheckTx. This handler is used to enforce parity in the app-side / comet mempools.
type MempoolParityCheckTx struct {
	// logger
	logger log.Logger

	// app side mempool interface
	mempl block.Mempool

	// tx-decoder
	txDecoder sdk.TxDecoder

	// checkTxHandler to wrap
	checkTxHandler CheckTx

	// baseApp is utilized to retrieve the latest committed state and to call
	// baseapp's CheckTx method.
	baseApp BaseApp
}

// NewMempoolParityCheckTx returns a new MempoolParityCheckTx handler.
func NewMempoolParityCheckTx(
	logger log.Logger,
	mempl block.Mempool,
	txDecoder sdk.TxDecoder,
	checkTxHandler CheckTx,
	baseApp BaseApp,
) MempoolParityCheckTx {
	return MempoolParityCheckTx{
		logger:         logger,
		mempl:          mempl,
		txDecoder:      txDecoder,
		checkTxHandler: checkTxHandler,
		baseApp:        baseApp,
	}
}

// CheckTx returns a CheckTx handler that wraps a given CheckTx handler and evicts txs that are not
// in the app-side mempool on ReCheckTx.
func (m MempoolParityCheckTx) CheckTx() CheckTx {
	return func(req *cmtabci.RequestCheckTx) (*cmtabci.ResponseCheckTx, error) {
		// decode tx
		tx, err := m.txDecoder(req.Tx)
		if err != nil {
			return sdkerrors.ResponseCheckTxWithEvents(
				fmt.Errorf("failed to decode tx: %w", err),
				0,
				0,
				nil,
				false,
			), nil
		}

		isReCheck := req.Type == cmtabci.CheckTxType_Recheck
		txInMempool := m.mempl.Contains(tx)

		// if the mode is ReCheck and the app's mempool does not contain the given tx, we fail
		// immediately, to purge the tx from the comet mempool.
		if isReCheck && !txInMempool {
			m.logger.Debug(
				"tx from comet mempool not found in app-side mempool",
				"tx", tx,
			)

			return sdkerrors.ResponseCheckTxWithEvents(
				fmt.Errorf("tx from comet mempool not found in app-side mempool"),
				0,
				0,
				nil,
				false,
			), nil
		}

		// run the checkTxHandler
		res, checkTxError := m.checkTxHandler(req)

		// if re-check fails for a transaction, we'll need to explicitly purge the tx from
		// the app-side mempool
		if isInvalidCheckTxExecution(res, checkTxError) && isReCheck {
			// check if the tx exists first
			if txInMempool {
				// remove the tx
				if err := m.mempl.Remove(tx); err != nil {
					m.logger.Debug(
						"failed to remove tx from app-side mempool when purging for re-check failure",
						"removal-err", err,
						"check-tx-err", checkTxError,
					)
				}
			}
		}

		sdkCtx := m.GetContextForTx(req)
		lane, err := m.matchLane(sdkCtx, tx)
		if err != nil {
			m.logger.Debug("failed to match lane", "lane", lane, "err", err)
			return sdkerrors.ResponseCheckTxWithEvents(
				err,
				0,
				0,
				nil,
				false,
			), nil
		}

		consensusParams := sdkCtx.ConsensusParams()
		laneSize := lane.GetMaxBlockSpace().MulInt64(consensusParams.GetBlock().GetMaxBytes()).TruncateInt64()

		txSize := int64(len(req.Tx))
		if txSize > laneSize {
			m.logger.Debug(
				"tx size exceeds max block bytes",
				"tx", tx,
				"tx size", txSize,
				"max bytes", laneSize,
			)

			return sdkerrors.ResponseCheckTxWithEvents(
				fmt.Errorf("tx size exceeds max block bytes"),
				0,
				0,
				nil,
				false,
			), nil
		}

		return res, checkTxError
	}
}

// matchLane returns a Lane if the given tx matches the Lane.
func (m MempoolParityCheckTx) matchLane(ctx sdk.Context, tx sdk.Tx) (block.Lane, error) {
	var lane block.Lane
	// find corresponding lane for this tx
	for _, l := range m.mempl.Registry() {
		if l.Match(ctx, tx) {
			lane = l
			break
		}
	}

	if lane == nil {
		m.logger.Debug(
			"failed match tx to lane",
			"tx", tx,
		)

		return nil, fmt.Errorf("failed match tx to lane")
	}

	return lane, nil
}

func isInvalidCheckTxExecution(resp *cmtabci.ResponseCheckTx, checkTxErr error) bool {
	return resp == nil || resp.Code != 0 || checkTxErr != nil
}

// GetContextForTx is returns the latest committed state and sets the context given
// the checkTx request.
func (m MempoolParityCheckTx) GetContextForTx(req *cmtabci.RequestCheckTx) sdk.Context {
	// Retrieve the commit multi-store which is used to retrieve the latest committed state.
	ms := m.baseApp.CommitMultiStore().CacheMultiStore()

	// Create a new context based off of the latest committed state.
	header := cmtproto.Header{
		Height:  m.baseApp.LastBlockHeight(),
		ChainID: m.baseApp.ChainID(),
	}
	ctx, _ := sdk.NewContext(ms, header, true, m.baseApp.Logger()).CacheContext()

	// Set the context to the correct checking mode.
	switch req.Type {
	case cmtabci.CheckTxType_New:
		ctx = ctx.WithIsCheckTx(true)
	case cmtabci.CheckTxType_Recheck:
		ctx = ctx.WithIsReCheckTx(true)
	default:
		panic("unknown check tx type")
	}

	// Set the remaining important context values.
	ctx = ctx.
		WithTxBytes(req.Tx).
		WithEventManager(sdk.NewEventManager()).
		WithConsensusParams(m.baseApp.GetConsensusParams(ctx))

	return ctx
}
