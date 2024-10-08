# How it Works

<!-- TODO: add images to this -->

### Summary

With the Block SDK, blocks are broken up into smaller partial blocks called `lanes`.

- Each `lane` has its own custom block building logic and stores distinct types of transactions.
- Each lane can only consume a portion of the block as defined on the `lane`'s configuration (`MaxBlockSpace`).
- When a block proposal is requested, a block will **fill** with transactions from each `lane`, iteratively, in the order in which the `lanes` are defined in the application.
- When a block proposal is processed, each `lane` will **verify** its portion of the block, iteratively, in the order in which the `lanes` are defined in the application.
- **Transactions in blocks MUST respect the ordering of lanes.**

### 🔁 Background: Transaction Lifecycle

Knowledge of the general transaction lifecycle is important to understand how `lanes` work.

- A transaction begins when it is signed and broadcasted to a node on a chain.
- It will be then be verified by the application on the node.
- If it is valid, it will be inserted into the node's `mempool`, which is a storage area for transactions before inclusion in a block.
- If the node happens to be a `validator`, and is proposing a block, the application will call `PrepareProposal` to create a new block proposal.
- The proposer will look at what transactions they have in their mempool, iteratively select transactions until the block is full, and share the proposal with other validators.
- When a different validator receives a proposal, the validator will verify its contents via `ProcessProposal` before signing it.
- If the proposal is valid, the validator will sign the proposal and broadcast their vote to the network.
- If the block is invalid, the validator will reject the proposal.
- Once a proposal is accepted by the network, it is committed as a block and the transactions that were included are removed from every validator's mempool.

### 🛣️ Lane Lifecycle

`Lanes` introduce new steps in the transaction lifecycle outlined above.

A `LanedMempool` is composed of several distinct `lanes` that store their own transactions. The `LanedMempool` will insert the transaction into all `lanes` that accept it

- After the base application accepts a transaction, the transaction will be checked to see if it can go into any `lanes`, as defined by the lane's `MatchHandler`.
- `Lane`'s can be configured to only accept transactions that match a certain criteria. For example, a `lane` could be configured to only accept transactions that are staking related (such as a free-transaction lane).
- When a new block is proposed, the `PrepareProposalHandler` of the application will iteratively call `PrepareLane` on each `lane` (in the order in which they are defined in the application). The `PrepareLane` method is similar to `PrepareProposal`.
- Calling `PrepareLane` on a `lane` will trigger the lane to reap transactions from its mempool and add them to the proposal (if they respect the verification rules of the `lane`).
- When proposals are verified in `ProcessProposal` by other validators, the `ProcessProposalHandler` defined in `abci/abci.go` will call `ProcessLane` on each `lane` in the same order as they were called in the `PrepareProposalHandler`.
- Each subsequent call to `ProcessLane` will filter out transactions that belong to previous lanes. **A given lane's ProcessLane will only verify transactions that belong to that lane.**

**Scenario**

Let's say we have a `LanedMempool` composed of two lanes: `LaneA` and `LaneB`.

`LaneA` is defined first in the `LanedMempool` and `LaneB` is defined second.

`LaneA` contains transactions Tx1 and Tx2 and `LaneB` contains transactions
Tx3 and Tx4.


When a new block needs to be proposed, the `PrepareProposalHandler` will call `PrepareLane` on `LaneA` first and `LaneB` second.

When `PrepareLane` is called on `LaneA`, `LaneA` will reap transactions from its mempool and add them to the proposal. The same applies for `LaneB`. Say `LaneA` reaps transactions Tx1 and Tx2 and `LaneB` reaps transactions Tx3 and Tx4. This gives us a proposal composed of the following:

- `Tx1`, `Tx2`, `Tx3`, `Tx4`

When the `ProcessProposalHandler` is called, it will call `ProcessLane` on `LaneA` with the proposal composed of Tx1, Tx2, Tx3, and Tx4. `LaneA` will then verify Tx1 and Tx2 and return the remaining transactions - Tx3 and Tx4. The `ProcessProposalHandler` will then call `ProcessLane` on `LaneB` with the remaining transactions - Tx3 and Tx4. `LaneB` will then verify Tx3 and Tx4 and return no remaining transactions.
