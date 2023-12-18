package checktx

import (
	cometabci "github.com/cometbft/cometbft/abci/types"
)

type (
	// CheckTx is baseapp's CheckTx method that checks the validity of a
	// transaction.
	CheckTx func(req *cometabci.RequestCheckTx) (*cometabci.ResponseCheckTx, error)
)
