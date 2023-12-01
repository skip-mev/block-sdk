# Block SDK Proposal Construction & Verification

## Overview

This readme describes how the Block SDK constructs and verifies proposals. To get a high level overview of the Block SDK, please see the [Block SDK Overview](../README.md).

The Block SDK is a set of Cosmos SDK and ABCI++ primitives that allow chains to fully customize blocks for specific use cases. It turns your chain's blocks into a **`highway`** consisting of individual **`lanes`** with their own special functionality. The mempool is no longer a single black box, but rather a set of lanes that can be customized to fit your application's needs. Each lane is meant to deal with specific a set of transactions, and can be configured to fit your application's needs.

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

The `PrepareProposal` handler is called by the ABCI++ application when a new block is requested by the network for the given proposer. At runtime, the `PrepareProposal` handler will do the following:

1. Determine the order of lanes it wants to construct the proposal from. If the application is configured to utilize the Block SDK module, it will fetch the governance configured order of lanes and use that. Otherwise, it will use the order defined in your [`app.go`](../tests/app/app.go) file (see our test app as an example).
2. After determining the order, it will chain together all of the lane's `PrepareLane` methods into a single `PrepareProposal` method.
3. Each lane will select transactions from its mempool, verify them according to its own rules, and return a list of valid transactions to add and a list of transactions to remove i.e. see [`PrepareLane`](../block/base/abci.go).
4. The transactions will only be added to the current proposal being built _iff_ the transactions are under the block gas and size limit for that lane. If the transactions are over the limit, they will NOT be added to the proposal and the next lane will be called.
5. A final proposal is returned with all of the valid transactions from each lane.

If any of the lanes fail during `PrepareLane`, the next lane will be called and the proposal will be built from the remaining lanes. This is a fail-safe mechanism to ensure that the proposal is always built, even if one of the lanes fails to prepare. Additionally, state is mutated _iff_ the lane is successful in preparing its portion of the proposal.

To customize how much block-space a given lane consumes, you have to configure the `MaxBlockSpace` variable in your lane configuration object (`LaneConfig`). Please visit [`lanes.go`](../tests/app/lanes.go) for an example. This variable is a map of lane name to the maximum block space that lane can consume. Note that if the Block SDK module is utilized, the `MaxBlockSpace` variable will be overwritten by the governance configured value.

### Proposal Construction Example

> Let's say your application has 3 lanes, `A`, `B`, and `C`, and the order of lanes is `A -> B -> C`. 
> 
> * Lane `A` has a `MaxBlockSpace` of 1000 bytes and 500 gas limit.
> * Lane `B` has a `MaxBlockSpace` of 2000 bytes and 1000 gas limit.
> * Lane `C` has a `MaxBlockSpace` of 3000 bytes and 1500 gas limit.

Lane `A` currently contains 4 transactions:

* Tx1: 100 bytes, 100 gas
* Tx2: 800 bytes, 300 gas
* Tx3: 200 bytes, 100 gas
* Tx4: 100 bytes, 100 gas

Lane `B` currently contains 4 transactions:

* Tx5: 1000 bytes, 500 gas
* Tx6: 1200 bytes, 600 gas
* Tx7: 1500 bytes, 300 gas
* Tx8: 1000 bytes, 400 gas

Lane `C` currently contains 4 transactions:

* Tx9: 1000 bytes, 500 gas
* Tx10: 1200 bytes, 600 gas
* Tx11: 1500 bytes, 300 gas
* Tx12: 100 bytes, 400 gas

Assuming all transactions are valid according to their respective lanes, the proposal will be built as follows (this is pseudo-code):

```golang
// Somewhere in abci.go a new proposal is created:
blockProposal := proposals.NewProposal(...)
...
// First lane to be called is lane A, this will return the following transactions to add after PrepareLane is called:
partialProposalFromLaneA := {
    Tx1, // 100 bytes, 100 gas
    Tx2, // 800 bytes, 300 gas
    Tx3, // 200 bytes, 100 gas
}
...
// First Lane A will update the proposal.
if err := blockProposal.Update(partialProposalFromLaneA); err != nil {
    return err
}
...
// Next, lane B will be called with the following transactions to add after PrepareLane is called:
//
// Note: Here, Tx6 is excluded because it and Tx5's gas would exceed the lane gas limit. Tx7 is similarly excluded because it and Tx5's size would exceed the lane block size limit. Note that Tx5 will always be included first because it is ordered first and it is valid given the lane's constraints.
partialProposalFromLaneB := {
    Tx5, // 1000 bytes, 500 gas
    Tx8, // 1000 bytes, 400 gas
}
...
// Next, lane B will update the proposal.
if err := blockProposal.Update(partialProposalFromLaneB); err != nil {
    return err
}
...
// Finally, lane C will be called with the following transactions to add after PrepareLane is called:
partialProposalFromLaneC := {
    Tx9, // 1000 bytes, 500 gas
    Tx10, // 1200 bytes, 600 gas
    Tx12, // 100 bytes, 400 gas
}
...
// Finally, lane C will update the proposal.
if err := blockProposal.Update(partialProposalFromLaneC); err != nil {
    return err
}
...
// The final proposal will be:
blockProposal := {
    Tx1, // 100 bytes, 100 gas
    Tx2, // 800 bytes, 300 gas
    Tx3, // 200 bytes, 100 gas
    Tx5, // 1000 bytes, 500 gas
    Tx8, // 1000 bytes, 400 gas
    Tx9, // 1000 bytes, 500 gas
    Tx10, // 1200 bytes, 600 gas
    Tx12, // 100 bytes, 400 gas
}
```

If any transactions are invalid, they will not be included in the lane's partial proposal.

## Proposal Verification

The `ProcessProposal` handler is called by the ABCI++ application when a new block has been proposed by the proposer and needs to be verified by the network. At runtime, the `ProcessProposal` handler will do the following:

1. Determine the order of lanes it wants to verify the proposal with. If the application is configured to utilize the Block SDK module, it will fetch the governance configured order of lanes and use that. Otherwise, it will use the order defined in your [`app.go`](../tests/app/app.go) file (see our test app as an example).
2. After determining the order, it will chain together all of the lane's `ProcessLane` methods into a single `ProcessProposal` method.
3. Given that the proposal contains contiguous sections of transactions from a given lane, each lane will verify the transactions that belong to it and return the remaining transactions to verify for the next lane - see [`ProcessLane`](../block/base/abci.go).
4. After determining and verifiying the transactions that belong to it, the lane will attempt to update the current proposal - so as to replicate the exact same steps done in `PrepareProposal`. If the lane is unable to update the proposal, it will return an error and the proposal will be rejected.
5. Once all lanes have been called and no transactions are left to verify, the proposal outputted by `ProcessProposal` should be the same as the proposal outputted by `PrepareProposal`!

If any of the lanes fail during `ProcessLane`, the entire proposal is rejected. There ensures that there is always parity between the proposal built and the proposal verified.

### Proposal Verification Example

Following the example above, let's say we recieve the same proposal from the network:

```golang
blockProposal := {
    Tx1, // 100 bytes, 100 gas
    Tx2, // 800 bytes, 300 gas
    Tx3, // 200 bytes, 100 gas
    Tx5, // 1000 bytes, 500 gas
    Tx8, // 1000 bytes, 400 gas
    Tx9, // 1000 bytes, 500 gas
    Tx10, // 1200 bytes, 600 gas
    Tx12, // 100 bytes, 400 gas
}
```

The proposal will be verified as follows (this is pseudo-code):

```golang
// Somewhere in abci.go a new proposal is created:
blockProposal := proposals.NewProposal(...)
...
// First lane to be called is lane A, this will return the following transactions that it verified and the remaining transactions to verify after calling ProcessLane:
verifiedTransactionsFromLaneA, remainingTransactions := {
    Tx1, // 100 bytes, 100 gas
    Tx2, // 800 bytes, 300 gas
    Tx3, // 200 bytes, 100 gas
}, {
    Tx5, // 1000 bytes, 500 gas
    Tx8, // 1000 bytes, 400 gas
    Tx9, // 1000 bytes, 500 gas
    Tx10, // 1200 bytes, 600 gas
    Tx12, // 100 bytes, 400 gas
}
...
// First Lane A will update the proposal.
if err := blockProposal.Update(verifiedTransactionsFromLaneA); err != nil {
    return err
}
...
// Next, lane B will be called with the following transactions to verify and the remaining transactions to verify after calling ProcessLane:
verifiedTransactionsFromLaneB, remainingTransactions := {
    Tx5, // 1000 bytes, 500 gas
    Tx8, // 1000 bytes, 400 gas
}, {
    Tx9, // 1000 bytes, 500 gas
    Tx10, // 1200 bytes, 600 gas
    Tx12, // 100 bytes, 400 gas
}
...
// Next, lane B will update the proposal.
if err := blockProposal.Update(verifiedTransactionsFromLaneB); err != nil {
    return err
}
...
// Finally, lane C will be called with the following transactions to verify and the remaining transactions to verify after calling ProcessLane:
verifiedTransactionsFromLaneC, remainingTransactions := {
    Tx9, // 1000 bytes, 500 gas
    Tx10, // 1200 bytes, 600 gas
    Tx12, // 100 bytes, 400 gas
}, {}
...
// Finally, lane C will update the proposal.
if err := blockProposal.Update(verifiedTransactionsFromLaneC); err != nil {
    return err
}
...
// The final proposal will be:
blockProposal := {
    Tx1, // 100 bytes, 100 gas
    Tx2, // 800 bytes, 300 gas
    Tx3, // 200 bytes, 100 gas
    Tx5, // 1000 bytes, 500 gas
    Tx8, // 1000 bytes, 400 gas
    Tx9, // 1000 bytes, 500 gas
    Tx10, // 1200 bytes, 600 gas
    Tx12, // 100 bytes, 400 gas
}
```

As we can see, in the process of verifying a proposal, the proposal is updated to reflect the exact same steps done in `PrepareProposal`.

