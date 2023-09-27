package base

import (
	"fmt"

	"cosmossdk.io/math"
	"github.com/cometbft/cometbft/libs/log"
	sdk "github.com/cosmos/cosmos-sdk/types"

	signer_extraction "github.com/skip-mev/block-sdk/adapters/signer_extraction_adapter"
	"github.com/skip-mev/block-sdk/block"
)

// LaneConfig defines the basic configurations needed for a lane.
type LaneConfig struct {
	Logger      log.Logger
	TxEncoder   sdk.TxEncoder
	TxDecoder   sdk.TxDecoder
	AnteHandler sdk.AnteHandler

	// SignerExtractor defines the interface used for extracting the expected signers of a transaction
	// from the transaction.
	SignerExtractor signer_extraction.Adapter

	// MaxBlockSpace defines the relative percentage of block space that can be
	// used by this lane. NOTE: If this is set to zero, then there is no limit
	// on the number of transactions that can be included in the block for this
	// lane (up to maxTxBytes as provided by the request). This is useful for the default lane.
	MaxBlockSpace math.LegacyDec

	// IgnoreList defines the list of lanes to ignore when processing transactions. This
	// is useful for when you want lanes to exist after the default lane. For example,
	// say there are two lanes: default and free. The free lane should be processed after
	// the default lane. In this case, the free lane should be added to the ignore list
	// of the default lane. Otherwise, the transactions that belong to the free lane
	// will be processed by the default lane (which accepts all transactions by default).
	IgnoreList []block.Lane

	// MaxTxs sets the maximum number of transactions allowed in the mempool with
	// the semantics:
	// - if MaxTx == 0, there is no cap on the number of transactions in the mempool
	// - if MaxTx > 0, the mempool will cap the number of transactions it stores,
	//   and will prioritize transactions by their priority and sender-nonce
	//   (sequence number) when evicting transactions.
	// - if MaxTx < 0, `Insert` is a no-op.
	MaxTxs int
}

// NewLaneConfig returns a new LaneConfig. This will be embedded in a lane.
func NewLaneConfig(
	logger log.Logger,
	txEncoder sdk.TxEncoder,
	txDecoder sdk.TxDecoder,
	anteHandler sdk.AnteHandler,
	signerExtractor signer_extraction.Adapter,
	maxBlockSpace math.LegacyDec,
) LaneConfig {
	return LaneConfig{
		Logger:          logger,
		TxEncoder:       txEncoder,
		TxDecoder:       txDecoder,
		AnteHandler:     anteHandler,
		MaxBlockSpace:   maxBlockSpace,
		SignerExtractor: signerExtractor,
	}
}

// ValidateBasic validates the lane configuration.
func (c *LaneConfig) ValidateBasic() error {
	if c.Logger == nil {
		return fmt.Errorf("logger cannot be nil")
	}

	if c.TxEncoder == nil {
		return fmt.Errorf("tx encoder cannot be nil")
	}

	if c.TxDecoder == nil {
		return fmt.Errorf("tx decoder cannot be nil")
	}

	if c.SignerExtractor == nil {
		return fmt.Errorf("signer extractor cannot be nil")
	}

	if c.MaxBlockSpace.IsNil() || c.MaxBlockSpace.IsNegative() || c.MaxBlockSpace.GT(math.LegacyOneDec()) {
		return fmt.Errorf("max block space must be set to a value between 0 and 1")
	}

	return nil
}
