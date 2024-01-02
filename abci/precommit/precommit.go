package precommit

import (
	"cosmossdk.io/log"
	"github.com/cosmos/cosmos-sdk/types/module"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/skip-mev/block-sdk/block"
)

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

func (m *MempoolEvictionPreCommiter) PreCommit() sdk.Precommiter {
	return func(ctx sdk.Context) {
		// Precommiter application updates every commit
		err := m.mm.Precommit(ctx)
		if err != nil {
			panic(err)
		}

		// add eviction hook
		for iterator := m.mempl.Select(ctx, nil); iterator != nil; iterator = iterator.Next() {
			tx := iterator.Tx()
			_, err := m.anteHandler(ctx, tx, false)
			if err != nil {
				err := m.mempl.Remove(tx)
				if err != nil {
					panic(err)
				}
			}

		}
	}
}
