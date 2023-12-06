# Base Lane

## Overview

> The base lane is purposefully built to be a simple lane that can be extended (inherited) by other lanes. It provides the basic functionality that is required by all lanes with the option to override any of the methods.

The base lane implements the lane interface and provides a few critical methods that allow application developers to create a lane that has custom transaction ordering and execution logic. The most important things you need to know in order to build a custom lane are:

* `MatchHandler`: This method is responsible for determining if a given transaction should be accepted by the lane.
* `PrepareLaneHandler`: This method is responsible for reaping transactions from the mempool, validating them, re-ordering (if necessary), and returning them to be included in a block proposal.
* `ProcessLaneHandler`: This method is responsible verifying the matched transactions that were included in a block proposal.
* `LaneMempool`: This allows developers to have the freedom to implement their own mempools with custom transaction ordering logic.
* `LaneConfig`: This allows developers to customize how the lane behaves in terms of max block space, max transaction count, and more.

## MatchHandler

MatchHandler is utilized to determine if a transaction should be included in the lane. This function can be a stateless or stateful check on the transaction. The function signature is as follows:

```go
MatchHandler func(ctx sdk.Context, tx sdk.Tx) bool
```

To create a custom lane with a custom `MatchHandler`, you must implement this function and pass it into the constructor for the base lane. For example, the [free lane](../../lanes/free/lane.go) inherits all the base lane functionality but overrides the `MatchHandler` to only accept staking related transactions.

```go
// DefaultMatchHandler returns the default match handler for the free lane. The
// default implementation matches transactions that are staking related. In particular,
// any transaction that is a MsgDelegate, MsgBeginRedelegate, or MsgCancelUnbondingDelegation.
func DefaultMatchHandler() base.MatchHandler {
	return func(ctx sdk.Context, tx sdk.Tx) bool {
		for _, msg := range tx.GetMsgs() {
			switch msg.(type) {
			case *types.MsgDelegate:
				return true
			case *types.MsgBeginRedelegate:
				return true
			case *types.MsgCancelUnbondingDelegation:
				return true
			}
		}

		return false
	}
}
```

The default `MatchHandler` is implemented in the [base lane](./match.go) and matches all transactions.

## PrepareLaneHandler

The `PrepareLaneHandler` is responsible for reaping transactions from the mempool, validating them, re-ordering (if necessary), and returning them to be included in a block proposal. If any of the transactions were invalid, it should return them alongside the transactions it wants to include in the proposal. The invalid transactions will subsequently be removed from the lane's mempool. The function signature is as follows:

```go
PrepareLaneHandler func(
    ctx sdk.Context,
    proposal proposals.Proposal,
    limit proposals.LaneLimits,
) (txsToInclude []sdk.Tx, txsToRemove []sdk.Tx, err error)
```

To create a custom lane with a custom `PrepareLaneHandler`, you must implement this function and set it on the lane after it has been created. Please visit the [MEV lane's](../../lanes/mev/abci.go) `PrepareLaneHandler` for an example of how to implement this function.

The default `PrepareLaneHandler` is implemented in the [base lane](./proposals.go). It reaps transactions from the mempool, validates them, ensures that the lane's block space limit is not exceeded, and returns the transactions to be included in the block and the ones that need to be removed.

## ProcessLaneHandler

The `ProcessLaneHandler` is responsible for verifying the transactions that belong to a given lane that were included in a block proposal and returning those that did not to the next lane. The function signature is as follows:

```go
ProcessLaneHandler func(ctx sdk.Context, partialProposal []sdk.Tx) (
    txsFromLane []sdk.Tx,
    remainingTxs []sdk.Tx,
    err error,
)
```

Note that block proposals built using the Block SDK contain contiguous sections of transactions in the block that belong to a given lane, to read more about how proposals are constructed relative to other lanes, please visit the [abci section](../../abci/README.md). As such, a given lane will recieve some transactions in (partialProposal) that belong to it and some that do not. The transactions that belong to it must be contiguous from the start, and the transactions that do not belong to it must be contiguous from the end. The lane must return the transactions that belong to it and the transactions that do not belong to it. The transactions that do not belong to it will be passed to the next lane in the proposal. The default `ProcessLaneHandler` is implemented in the [base lane](./proposals.go). It verifies the transactions that belong to the lane and returns them alongside the transactions that do not belong to the lane.

Please visit the [MEV lane's](../../lanes/mev/abci.go) `ProcessLaneHandler` for an example of how to implement a custom handler.

## LaneMempool

The lane mempool is the data structure that is responsible for storing transactions that belong to a given lane, before they are included in a block proposal. The lane mempool input's a `TxPriority` object that allows developers to customize how they want to order transactions within their mempool. Additionally, it also accepts a signer extrator adapter that allows for custom signature schemes to be used (although the default covers Cosmos SDK transactions). To read more about the signer extractor adapter, please visit the [signer extractor section](../../adapters/signer_extraction_adapter/README.md). 

### TxPriority

The `TxPriority` object is responsible for ordering transactions within the mempool. The definition of the `TxPriority` object is as follows:

```go
// TxPriority defines a type that is used to retrieve and compare transaction
// priorities. Priorities must be comparable.
TxPriority[C comparable] struct {
    // GetTxPriority returns the priority of the transaction. A priority must be
    // comparable via Compare.
    GetTxPriority func(ctx context.Context, tx sdk.Tx) C

    // CompareTxPriority compares two transaction priorities. The result should be
    // 0 if a == b, -1 if a < b, and +1 if a > b.
    Compare func(a, b C) int

    // MinValue defines the minimum priority value, e.g. MinInt64. This value is
    // used when instantiating a new iterator and comparing weights.
    MinValue C
}
```

The default implementation can be found in the [base lane](./mempool.go). It orders transactions by their gas price in descending order. The `TxPriority` object is passed into the lane mempool constructor. Please visit the [MEV lane's](../../lanes/mev/mempool.go) `TxPriority` for an example of how to implement a custom `TxPriority`.

## LaneConfig

The lane config is the object that is responsible for configuring the lane. It allows developers to customize how the lane behaves in terms of max block space, max transaction count, and more. The definition of the `LaneConfig` object is as follows:

```go
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

	// MaxTxs sets the maximum number of transactions allowed in the mempool with
	// the semantics:
	// - if MaxTx == 0, there is no cap on the number of transactions in the mempool
	// - if MaxTx > 0, the mempool will cap the number of transactions it stores,
	//   and will prioritize transactions by their priority and sender-nonce
	//   (sequence number) when evicting transactions.
	// - if MaxTx < 0, `Insert` is a no-op.
	MaxTxs int
}
```

Each lane must define its own custom `LaneConfig` in order to be properly instantiated. Please visit [`app.go`](../../tests/app/app.go) for an example of how to implement a custom `LaneConfig`.



