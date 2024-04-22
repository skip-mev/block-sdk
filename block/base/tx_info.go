package base

import (
	"encoding/hex"
	"fmt"
	"strings"

	comettypes "github.com/cometbft/cometbft/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/skip-mev/block-sdk/v2/block/utils"
)

// GetTxInfo returns various information about the transaction that
// belongs to the lane including its priority, signer's, sequence number,
// size and more.
func (l *BaseLane) GetTxInfo(ctx sdk.Context, tx sdk.Tx) (utils.TxWithInfo, error) {
	txBytes, err := l.cfg.TxEncoder(tx)
	if err != nil {
		return utils.TxWithInfo{}, fmt.Errorf("failed to encode transaction: %w", err)
	}

	// TODO: Add an adapter to lanes so that this can be flexible to support EVM, etc.
	gasTx, ok := tx.(sdk.FeeTx)
	if !ok {
		return utils.TxWithInfo{}, fmt.Errorf("failed to cast transaction to gas tx")
	}

	signers, err := l.cfg.SignerExtractor.GetSigners(tx)
	if err != nil {
		return utils.TxWithInfo{}, err
	}

	return utils.TxWithInfo{
		Hash:     strings.ToUpper(hex.EncodeToString(comettypes.Tx(txBytes).Hash())),
		Size:     int64(len(txBytes)),
		GasLimit: gasTx.GetGas(),
		TxBytes:  txBytes,
		Priority: l.LaneMempool.Priority(ctx, tx),
		Signers:  signers,
	}, nil
}
