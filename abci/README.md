# Block SDK Proposal Construction & Verification

## Overview

This readme describes how the Block SDK constructs and verifies proposals. To get a high level overview of the Block SDK, please see the [Block SDK Overview](../README.md).

At a high level, the Block SDK is a set of Cosmos SDK and ABCI++ primitives that allow chains to fully customize blocks to specific use cases. It turns your chain's blocks into a **`highway`** consisting of individual **`lanes`** with their own special functionality. The mempool is no longer a single black box, but rather a set of lanes that can be customized to fit your application's needs. Each lane is meant to deal with specific types of transactions, and can be configured to fit your application's needs.

Proposal construction / verification is done via the `PrepareProposal` and `ProcessProposal` handlers defined in [`abci.go`](./abci.go), respectively. Each block proposal built by the Block SDK enforces that a block is comprised of contiguous sections of transactions that belong to a single lane.

For example, if your application has 3 lanes, `A`, `B`, and `C`, and the order of lanes is `A -> B -> C`, then the proposal will be built as follows:

```golang
blockProposal := {
    Tx1, (Lane A)
    Tx2, (Lane A)
    Tx3, (Lane A)
    Tx4, (Lane B)
    Tx5, (Lane B)
    Tx6, (Lane B)
    Tx7, (Lane C)
    Tx8, (Lane C)
    Tx9, (Lane C)
}
```

## Proposal Construction

The `PrepareProposal` handler is called by the ABCI++ application when a new block is requested by the network for the given proposer. At runtime, the `PrepareProposal` handler will do the following steps:

1. Determine the order of lanes it wants to construct the proposal from. If the application is configured to utilize the Block SDK module, it will fetch the governance configured order of lanes and use that. Otherwise, it will use the order defined in your [`app.go`](../tests/app/app.go) file (see our test app as an example).
2. After determining the order, it will chain together all of the lane's `PrepareLane` methods into a single `PrepareProposal` method.
3. Each lane will select transactions from its mempool, verify them according to its own rules, and return a list of valid transactions to add and a list of transactions to remove i.e. see [`PrepareLane`](../block/base/abci.go).
4. The transactions will only be added to the current proposal being built iff the transactions are under the block gas and size limit for that lane. If the transactions are over the limit, they will NOT be added to the proposal and the next lane will be called.
5. Once all lanes have been called, a final proposal is returned with all of the valid transactions from each lane.

If any of the lanes fail to `PrepareLane`, the next lane will be called and the proposal will be built from the remaining lanes. This is a fail-safe mechanism to ensure that the proposal is always built, even if one of the lanes fails to prepare. Additionally, state is mutated iff the lane is successful in preparing its portion of the proposal.

To customize how much block-space a given lane consumes, you have to configure the `MaxBlockSpace` variable in your lane configuration object (`LaneConfig`). Please visit [`lanes.go`](../tests/app/lanes.go) for an example. This variable is a map of lane name to the maximum block space that lane can consume. Note that if the Block SDK module is utilized, the `MaxBlockSpace` variable will be overwritten by the governance configured value.

## Proposal Verification

The `ProcessProposal` handler is called by the ABCI++ application when a new block has been proposed by the proposer and needs to be verified by the network. At runtime, the `ProcessProposal` handler will do the following steps:

1. Determine the order of lanes it wants to verify the proposal with. If the application is configured to utilize the Block SDK module, it will fetch the governance configured order of lanes and use that. Otherwise, it will use the order defined in your [`app.go`](../tests/app/app.go) file (see our test app as an example).
2. After determining the order, it will chain together all of the lane's `ProcessLane` methods into a single `ProcessProposal` method.
3. Given that the proposal contains contiguous sections of transactions from a given lane, each lane will verify the transactions that belong to it and return the remaining transactions to verify for the next lane - see [`ProcessLane`](../block/base/abci.go).
4. After determining and verifiying the transactions that belong to it, the lane will attempt to update the current proposal - so as to replicate the exact same steps done in `PrepareProposal`. If the lane is unable to update the proposal, it will return an error and the proposal will be rejected.
5. Once all lanes have been called and no transactions are left to verify, the proposal outputted by `ProcessProposal` should be the same as the proposal outputted by `PrepareProposal`!