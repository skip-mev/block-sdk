package precommit

import (
	"cosmossdk.io/log"
	"github.com/cosmos/cosmos-sdk/types/module"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/skip-mev/block-sdk/v2/block"
)

// MempoolEvictionPreCommiter is a PreCommit function that evicts txs that invalid from the app-side mempool.
type MempoolEvictionPreCommiter struct {
	// logger
	logger log.Logger

	// app side mempool interface
	mempl block.Mempool

	// module manager
	mm *module.Manager

	// ante handler
	anteHandler sdk.AnteHandler
}

// NewMempoolEvictionPreCommiter returns a new MempoolEvictionPreCommiter handler.
func NewMempoolEvictionPreCommiter(
	logger log.Logger,
	mempl block.Mempool,
	mm *module.Manager,
	anteHandler sdk.AnteHandler,
) MempoolEvictionPreCommiter {
	return MempoolEvictionPreCommiter{
		logger:      logger,
		mempl:       mempl,
		mm:          mm,
		anteHandler: anteHandler,
	}
}

// PreCommit returns a PreCommit handler that wraps the default PreCommit handler and evicts invalid txs from the
// app-side mempool
func (m *MempoolEvictionPreCommiter) PreCommit() sdk.Precommiter {
	return func(ctx sdk.Context) {
		defer func() {
			if rec := recover(); rec != nil {
				m.logger.Error(
					"panic in precommiter",
					"err", rec)
			}
		}()

		// call precommit per module
		err := m.mm.Precommit(ctx)
		if err != nil {
			panic(err)
		}

		lanes, err := m.mempl.Registry(ctx)
		if err != nil {
			panic(err)
		}

		// add eviction hook
		for _, lane := range lanes {
			for iterator := lane.Select(ctx, nil); iterator != nil; iterator = iterator.Next() {
				tx := iterator.Tx()

				// run ante handler on all tx
				_, err := m.anteHandler(ctx, tx, false)
				if err != nil {
					m.logger.Debug("error running antehandler on tx in precommit: removing from mempool",
						"tx", tx,
						"error", err,
					)
					err := m.mempl.Remove(tx)
					if err != nil {
						panic(err)
					}
				}

				// check if tx matches still
				if !lane.Match(ctx, tx) {
					m.logger.Debug("tx does not match lane: removing from mempool",
						"tx", tx,
					)
					err := m.mempl.Remove(tx)
					if err != nil {
						panic(err)
					}
				}
			}
		}
	}
}
