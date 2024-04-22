package checktx

import (
	"fmt"

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
}

// NewMempoolParityCheckTx returns a new MempoolParityCheckTx handler.
func NewMempoolParityCheckTx(logger log.Logger, mempl block.Mempool, txDecoder sdk.TxDecoder, checkTxHandler CheckTx) MempoolParityCheckTx {
	return MempoolParityCheckTx{
		logger:         logger,
		mempl:          mempl,
		txDecoder:      txDecoder,
		checkTxHandler: checkTxHandler,
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

		// if the mode is ReCheck and the app's mempool does not contain the given tx, we fail
		// immediately, to purge the tx from the comet mempool.
		if isReCheck && !m.mempl.Contains(tx) {
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
			if m.mempl.Contains(tx) {
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

		return res, checkTxError
	}
}

func isInvalidCheckTxExecution(resp *cmtabci.ResponseCheckTx, checkTxErr error) bool {
	return resp == nil || resp.Code != 0 || checkTxErr != nil
}
