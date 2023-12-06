# Block SDK Mempool & Lanes

## Overview

> This document describes how the Block SDK mempool and lanes operate at a high level. To learn more about how to construct lanes, please visit the [build my own lane readme](../lanes/build-your-own/README.md) and/or the [base lane documentation](./base/README.md). To read about how proposals are construct, visit the [abci readme](../abci/README.md).

Mempools are traditionally used to temporarily store transactions before they are added to a block. The Block SDK mempool is no different. However, instead of treating each transaction the same, the Block SDK allows for developers to create `Lanes` that permit transactions to be ordered differently based on the properties of the transaction itself.

What was once a single monolithic data structure, is now a collection of sub-mempools that can be configured to order transactions in a way that makes sense for the application.

## Lanes

Lanes are utilized to allow developers to create custom transaction order, validation, and execution logic. Each lane is responsible for maintaining its own mempool - ordering transactions as it desires only for the transactions it wants to accept. For example, a lane may only accept transactions that are staking related, such as the free lane. The free lane may then order the transactions based on the user's on-chain stake.

When proposals are constructed, the transactions from a given lane are selected based on highest to lowest priority, validated according to the lane's verfication logic, and included in the proposal.

Each lane must implement the `Lane` interface, although it is highly recommended that developers extend the [base lane](./base/README.md) to create new lanes.

```go
// LaneMempool defines the interface a lane's mempool should implement. The basic API
// is the same as the sdk.Mempool, but it also includes a Compare function that is used
// to determine the relative priority of two transactions belonging in the same lane.
type LaneMempool interface {
	sdkmempool.Mempool

	// Compare determines the relative priority of two transactions belonging in the same lane. Compare
	// will return -1 if this transaction has a lower priority than the other transaction, 0 if they have
	// the same priority, and 1 if this transaction has a higher priority than the other transaction.
	Compare(ctx sdk.Context, this, other sdk.Tx) (int, error)

	// Contains returns true if the transaction is contained in the mempool.
	Contains(tx sdk.Tx) bool

	// Priority returns the priority of a transaction that belongs to this lane.
	Priority(ctx sdk.Context, tx sdk.Tx) any
}

// Lane defines an interface used for matching transactions to lanes, storing transactions,
// and constructing partial blocks.
type Lane interface {
	LaneMempool

	// PrepareLane builds a portion of the block. It inputs the current context, proposal, and a
	// function to call the next lane in the chain. This handler should update the context as needed
	// and add transactions to the proposal. Note, the lane should only add transactions up to the
	// max block space for the lane.
	PrepareLane(
		ctx sdk.Context,
		proposal proposals.Proposal,
		next PrepareLanesHandler,
	) (proposals.Proposal, error)

	// ProcessLane verifies this lane's portion of a proposed block. It inputs the current context,
	// proposal, transactions that belong to this lane, and a function to call the next lane in the
	// chain. This handler should update the context as needed and add transactions to the proposal.
	// The entire process lane chain should end up constructing the same proposal as the prepare lane
	// chain.
	ProcessLane(
		ctx sdk.Context,
		proposal proposals.Proposal,
		txs []sdk.Tx,
		next ProcessLanesHandler,
	) (proposals.Proposal, error)

	// GetMaxBlockSpace returns the max block space for the lane as a relative percentage.
	GetMaxBlockSpace() math.LegacyDec

	// SetMaxBlockSpace sets the max block space for the lane as a relative percentage.
	SetMaxBlockSpace(math.LegacyDec)

	// Name returns the name of the lane.
	Name() string

	// Match determines if a transaction belongs to this lane.
	Match(ctx sdk.Context, tx sdk.Tx) bool

	// GetTxInfo returns various information about the transaction that
	// belongs to the lane including its priority, signer's, sequence number,
	// size and more.
	GetTxInfo(ctx sdk.Context, tx sdk.Tx) (utils.TxWithInfo, error)
}
```

## Lane Priorities

Each lane has a priority that is used to determine the order in which lanes are processed. The higher the priority, the earlier the lane is processed. For example, if we have three lanes - MEV, free, and default - proposals will be constructed in the following order:

1. MEV
2. Free
3. Default

Proposals will then be verified in the same order. Please see the [readme above](../abci/README.md) for more information on how proposals are constructed using lanes.

The ordering of lane's priorities is determined based on the order passed into the constructor of the Block SDK mempool i.e. `LanedMempool`.

## Block SDK mempool

The `LanedMempool` is a wrapper on top of the collection of lanes. It is solely responsible for adding transactions to the appropriate lanes. Transactions are always inserted / removed to the first lane that accepts / matches the transactions. **Transactions should only match to one lane.**. **In the case where a transaction can match to multiple lanes, the transaction will be inserted into the lane that has the highest priority.**

To read more about the underlying implementation of the Block SDK mempool, please see the implementation [here](./mempool.go).
