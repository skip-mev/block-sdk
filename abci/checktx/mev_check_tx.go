package checktx

import (
	"context"
	"fmt"

	cometabci "github.com/cometbft/cometbft/abci/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	"cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"

	"github.com/skip-mev/block-sdk/v2/block"
	mevlane "github.com/skip-mev/block-sdk/v2/lanes/mev"
	"github.com/skip-mev/block-sdk/v2/x/auction/types"
)

// MevCheckTxHandler is a wrapper around baseapp's CheckTx method that allows us to
// verify bid transactions against the latest committed state. All other transactions
// are executed normally using base app's CheckTx. This defines all of the
// dependencies that are required to verify a bid transaction.
type MEVCheckTxHandler struct {
	// baseApp is utilized to retrieve the latest committed state and to call
	// baseapp's CheckTx method.
	baseApp BaseApp

	// txDecoder is utilized to decode transactions to determine if they are
	// bid transactions.
	txDecoder sdk.TxDecoder

	// MEVLane is utilized to retrieve the bid info of a transaction and to
	// insert a bid transaction into the application-side mempool.
	mevLane MEVLaneI

	// anteHandler is utilized to verify the bid transaction against the latest
	// committed state.
	anteHandler sdk.AnteHandler

	// checkTxHandler is the wrapped CheckTx handler that is used to execute all non-bid txs
	checkTxHandler CheckTx
}

// MEVLaneI defines the interface for the mev auction lane. This interface
// is utilized by both the x/auction module and the checkTx handler.
type MEVLaneI interface {
	block.Lane
	mevlane.Factory
	GetTopAuctionTx(ctx context.Context) sdk.Tx
}

// BaseApp is an interface that allows us to call baseapp's CheckTx method
// as well as retrieve the latest committed state.
type BaseApp interface {
	// CommitMultiStore is utilized to retrieve the latest committed state.
	CommitMultiStore() storetypes.CommitMultiStore

	// Logger is utilized to log errors.
	Logger() log.Logger

	// LastBlockHeight is utilized to retrieve the latest block height.
	LastBlockHeight() int64

	// GetConsensusParams is utilized to retrieve the consensus params.
	GetConsensusParams(ctx sdk.Context) cmtproto.ConsensusParams

	// ChainID is utilized to retrieve the chain ID.
	ChainID() string
}

// NewCheckTxHandler constructs a new CheckTxHandler instance. This method fails if the given LanedMempool does not have a lane
// adhering to the MevLaneI interface
func NewMEVCheckTxHandler(
	baseApp BaseApp,
	txDecoder sdk.TxDecoder,
	mevLane MEVLaneI,
	anteHandler sdk.AnteHandler,
	checkTxHandler CheckTx,
) *MEVCheckTxHandler {
	return &MEVCheckTxHandler{
		baseApp:        baseApp,
		txDecoder:      txDecoder,
		mevLane:        mevLane,
		anteHandler:    anteHandler,
		checkTxHandler: checkTxHandler,
	}
}

// CheckTxHandler is a wrapper around baseapp's CheckTx method that allows us to
// verify bid transactions against the latest committed state. All other transactions
// are executed normally. We must verify each bid tx and all of its bundled transactions
// before we can insert it into the mempool against the latest commit state because
// otherwise the auction can be griefed. No state changes are applied to the state
// during this process.
func (handler *MEVCheckTxHandler) CheckTx() CheckTx {
	return func(req *cometabci.RequestCheckTx) (resp *cometabci.ResponseCheckTx, err error) {
		defer func() {
			if rec := recover(); rec != nil {
				handler.baseApp.Logger().Error(
					"panic in check tx handler",
					"err", rec,
				)

				err = fmt.Errorf("panic in check tx handler: %s", rec)
				resp = sdkerrors.ResponseCheckTxWithEvents(
					err,
					0,
					0,
					nil,
					false,
				)
			}
		}()

		tx, err := handler.txDecoder(req.Tx)
		if err != nil {
			handler.baseApp.Logger().Info(
				"failed to decode tx",
				"err", err,
			)

			return sdkerrors.ResponseCheckTxWithEvents(
				fmt.Errorf("failed to decode tx: %w", err),
				0,
				0,
				nil,
				false,
			), nil
		}

		// Attempt to get the bid info of the transaction.
		bidInfo, err := handler.mevLane.GetAuctionBidInfo(tx)
		if err != nil {
			handler.baseApp.Logger().Info(
				"failed to get auction bid info",
				"err", err,
			)

			return sdkerrors.ResponseCheckTxWithEvents(
				fmt.Errorf("failed to get auction bid info: %w", err),
				0,
				0,
				nil,
				false,
			), nil
		}

		// If this is not a bid transaction, we just execute it normally.
		if bidInfo == nil {
			resp, err := handler.checkTxHandler(req)
			if err != nil {
				handler.baseApp.Logger().Info(
					"failed to execute check tx",
					"err", err,
				)
			}

			return resp, err
		}

		// We attempt to get the latest committed state in order to verify transactions
		// as if they were to be executed at the top of the block. After verification, this
		// context will be discarded and will not apply any state changes.
		ctx := handler.GetContextForBidTx(req)

		// Verify the bid transaction.
		gasInfo, err := handler.ValidateBidTx(ctx, tx, bidInfo)
		if err != nil {
			handler.baseApp.Logger().Info(
				"invalid bid tx",
				"err", err,
				"height", ctx.BlockHeight(),
				"bid_height", bidInfo.Timeout,
				"bidder", bidInfo.Bidder,
				"bid", bidInfo.Bid,
				"is_recheck_tx", ctx.IsReCheckTx(),
			)

			// attempt to remove the bid from the MEVLane (if it exists)
			if handler.mevLane.Contains(tx) {
				if err := handler.mevLane.Remove(tx); err != nil {
					handler.baseApp.Logger().Error(
						"failed to remove bid transaction from mev-lane",
						"err", err,
					)
				}
			}

			return sdkerrors.ResponseCheckTxWithEvents(
				fmt.Errorf("invalid bid tx: %w", err),
				gasInfo.GasWanted,
				gasInfo.GasUsed,
				nil,
				false,
			), nil
		}

		handler.baseApp.Logger().Info(
			"valid bid tx",
			"height", ctx.BlockHeight(),
			"bid_height", bidInfo.Timeout,
			"bidder", bidInfo.Bidder,
			"bid", bidInfo.Bid,
			"inserting tx into mempool", true,
		)

		// If the bid transaction is valid, we know we can insert it into the mempool for consideration in the next block.
		if err := handler.mevLane.Insert(ctx, tx); err != nil {
			handler.baseApp.Logger().Info(
				"invalid bid tx; failed to insert bid transaction into mempool",
				"err", err,
			)

			return sdkerrors.ResponseCheckTxWithEvents(
				fmt.Errorf("invalid bid tx; failed to insert bid transaction into mempool: %w", err),
				gasInfo.GasWanted,
				gasInfo.GasUsed,
				nil,
				false,
			), nil
		}

		return &cometabci.ResponseCheckTx{
			Code:      cometabci.CodeTypeOK,
			GasWanted: int64(gasInfo.GasWanted),
			GasUsed:   int64(gasInfo.GasUsed),
		}, nil
	}
}

// ValidateBidTx is utilized to verify the bid transaction against the latest committed state.
func (handler *MEVCheckTxHandler) ValidateBidTx(ctx sdk.Context, bidTx sdk.Tx, bidInfo *types.BidInfo) (sdk.GasInfo, error) {
	// Verify the bid transaction.
	ctx, err := handler.anteHandler(ctx, bidTx, false)
	if err != nil {
		return sdk.GasInfo{}, fmt.Errorf("invalid bid tx; failed to execute ante handler: %w", err)
	}

	// Store the gas info and priority of the bid transaction before applying changes with other transactions.
	gasInfo := sdk.GasInfo{
		GasWanted: ctx.GasMeter().Limit(),
		GasUsed:   ctx.GasMeter().GasConsumed(),
	}

	// Verify all of the bundled transactions.
	for _, tx := range bidInfo.Transactions {
		bundledTx, err := handler.mevLane.WrapBundleTransaction(tx)
		if err != nil {
			return gasInfo, fmt.Errorf("invalid bid tx; failed to decode bundled tx: %w", err)
		}

		// bid txs cannot be included in bundled txs
		bidInfo, err := handler.mevLane.GetAuctionBidInfo(bundledTx)
		if err != nil {
			return gasInfo, fmt.Errorf("invalid bid tx; failed to get bid info: %w", err)
		}

		if bidInfo != nil {
			return gasInfo, fmt.Errorf("invalid bid tx; bundled tx cannot be a bid tx")
		}

		if ctx, err = handler.anteHandler(ctx, bundledTx, false); err != nil {
			return gasInfo, fmt.Errorf("invalid bid tx; failed to execute bundled transaction: %w", err)
		}
	}

	return gasInfo, nil
}

// GetContextForBidTx is returns the latest committed state and sets the context given
// the checkTx request.
func (handler *MEVCheckTxHandler) GetContextForBidTx(req *cometabci.RequestCheckTx) sdk.Context {
	// Retrieve the commit multi-store which is used to retrieve the latest committed state.
	ms := handler.baseApp.CommitMultiStore().CacheMultiStore()

	// Create a new context based off of the latest committed state.
	header := cmtproto.Header{
		Height:  handler.baseApp.LastBlockHeight(),
		ChainID: handler.baseApp.ChainID(),
	}
	ctx, _ := sdk.NewContext(ms, header, true, handler.baseApp.Logger()).CacheContext()

	// Set the context to the correct checking mode.
	switch req.Type {
	case cometabci.CheckTxType_New:
		ctx = ctx.WithIsCheckTx(true)
	case cometabci.CheckTxType_Recheck:
		ctx = ctx.WithIsReCheckTx(true)
	default:
		panic("unknown check tx type")
	}

	// Set the remaining important context values.
	ctx = ctx.
		WithTxBytes(req.Tx).
		WithEventManager(sdk.NewEventManager()).
		WithConsensusParams(handler.baseApp.GetConsensusParams(ctx))

	return ctx
}
