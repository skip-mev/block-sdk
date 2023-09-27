package mev

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	signer_extraction "github.com/skip-mev/block-sdk/adapters/signer_extraction_adapter"
	"github.com/skip-mev/block-sdk/block/base"
	"github.com/skip-mev/block-sdk/x/auction/types"
)

type (
	// Factory defines the interface for processing auction transactions. It is
	// a wrapper around all of the functionality that each application chain must implement
	// in order for auction processing to work.
	Factory interface {
		// WrapBundleTransaction defines a function that wraps a bundle transaction into a sdk.Tx. Since
		// this is a potentially expensive operation, we allow each application chain to define how
		// they want to wrap the transaction such that it is only called when necessary (i.e. when the
		// transaction is being considered in the proposal handlers).
		WrapBundleTransaction(tx []byte) (sdk.Tx, error)

		// GetAuctionBidInfo defines a function that returns the bid info from an auction transaction.
		GetAuctionBidInfo(tx sdk.Tx) (*types.BidInfo, error)

		// MatchHandler defines a function that checks if a transaction matches the auction lane.
		MatchHandler() base.MatchHandler
	}

	// DefaultAuctionFactory defines a default implmentation for the auction factory interface for processing auction transactions.
	DefaultAuctionFactory struct {
		txDecoder       sdk.TxDecoder
		signerExtractor signer_extraction.Adapter
	}

	// TxWithTimeoutHeight is used to extract timeouts from sdk.Tx transactions. In the case where,
	// timeouts are explicitly set on the sdk.Tx, we can use this interface to extract the timeout.
	TxWithTimeoutHeight interface {
		sdk.Tx

		GetTimeoutHeight() uint64
	}
)

var _ Factory = (*DefaultAuctionFactory)(nil)

// NewDefaultAuctionFactory returns a default auction factory interface implementation.
func NewDefaultAuctionFactory(txDecoder sdk.TxDecoder, extractor signer_extraction.Adapter) Factory {
	return &DefaultAuctionFactory{
		txDecoder:       txDecoder,
		signerExtractor: extractor,
	}
}

// WrapBundleTransaction defines a default function that wraps a transaction
// that is included in the bundle into a sdk.Tx. In the default case, the transaction
// that is included in the bundle will be the raw bytes of an sdk.Tx so we can just
// decode it.
func (config *DefaultAuctionFactory) WrapBundleTransaction(tx []byte) (sdk.Tx, error) {
	return config.txDecoder(tx)
}

// GetAuctionBidInfo defines a default function that returns the auction bid info from
// an auction transaction. In the default case, the auction bid info is stored in the
// MsgAuctionBid message.
func (config *DefaultAuctionFactory) GetAuctionBidInfo(tx sdk.Tx) (*types.BidInfo, error) {
	msg, err := GetMsgAuctionBidFromTx(tx)
	if err != nil {
		return nil, err
	}

	if msg == nil {
		return nil, nil
	}

	bidder, err := sdk.AccAddressFromBech32(msg.Bidder)
	if err != nil {
		return nil, fmt.Errorf("invalid bidder address (%s): %w", msg.Bidder, err)
	}

	timeoutTx, ok := tx.(TxWithTimeoutHeight)
	if !ok {
		return nil, fmt.Errorf("cannot extract timeout; transaction does not implement TxWithTimeoutHeight")
	}

	signers, err := config.getBundleSigners(msg.Transactions)
	if err != nil {
		return nil, err
	}

	return &types.BidInfo{
		Bid:          msg.Bid,
		Bidder:       bidder,
		Transactions: msg.Transactions,
		Timeout:      timeoutTx.GetTimeoutHeight(),
		Signers:      signers,
	}, nil
}

func (config *DefaultAuctionFactory) MatchHandler() base.MatchHandler {
	return func(ctx sdk.Context, tx sdk.Tx) bool {
		bidInfo, err := config.GetAuctionBidInfo(tx)
		return bidInfo != nil && err == nil
	}
}

// getBundleSigners defines a default function that returns the signers of all transactions in
// a bundle. In the default case, each bundle transaction will be an sdk.Tx and the
// signers are the signers of each sdk.Msg in the transaction.
func (config *DefaultAuctionFactory) getBundleSigners(bundle [][]byte) ([]map[string]struct{}, error) {
	bundleSigners := make([]map[string]struct{}, 0)

	for _, tx := range bundle {
		sdkTx, err := config.txDecoder(tx)
		if err != nil {
			return nil, err
		}

		txSigners := make(map[string]struct{})

		signers, err := config.signerExtractor.GetSigners(sdkTx)
		if err != nil {
			return nil, err
		}

		for _, signer := range signers {
			txSigners[signer.Signer.String()] = struct{}{}
		}

		bundleSigners = append(bundleSigners, txSigners)
	}

	return bundleSigners, nil
}
