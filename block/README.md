## 🤔 How does it work

### Transaction Lifecycle

The best way to understand how lanes work is to first understand the lifecycle 
of a transaction. When a transaction is submitted to the chain, it will be checked
in `CheckTx` by the base application. If the transaction is valid, it will be
inserted into the applications mempool. The transaction then waits in the mempool
until a new block needs to be proposed. When a new block needs to be proposed,
the application will call `PrepareProposal` (which is a new ABCI++ addition) to
request a new block from the current proposer. The proposer will look at what the
transactions currently waiting to be included in a block in their mempool and 
will iterative select transactions until the block is full. The proposer will then
send the block to other validators in the network. When a validator receives a 
proposed block, the validator will first want to verify the contents of the block
before signing off on it. The validator will call `ProcessProposal` to verify the
contents of the block. If the block is valid, the validator will sign off on the
block and broadcast their vote to the network. If the block is invalid, the validator
will reject the block. Once a block is accepted by the network, it is committed
and the transactions that were included in the block are removed from the mempool.

### Lane Lifecycle

The Lane Constructor implements the `Lane` interface. After transactions are 
check in `CheckTx`, they will be added to this lane's mempool (data structure
responsible for storing transactions). When a new block is proposed, `PrepareLane`
will be called by the `PrepareProposalHandler` defined in `abci/abci.go`. This 
will trigger the lane to reap transactions from its mempool and add them to the
proposal. By default, transactions are added to proposals in the order that they
are reaped from the mempool. Transactions will only be added to a proposal
if they are valid according to the lane's verification logic. The default implementation
determines whether a transaction is valid by running the transaction through the
lane's `AnteHandler`. If any transactions are invalid, they will be removed from
lane's mempool from further consideration.

When proposals need to be verified in `ProcessProposal`, the `ProcessProposalHandler`
defined in `abci/abci.go` will call `ProcessLane` on each lane. This will trigger
the lane to process all transactions that are included in the proposal. Lane's 
should only verify transactions that belong to their lane. The default implementation
of `ProcessLane` will first check that transactions that should belong to the 
current lane are ordered correctly in the proposal. If they are not, the proposal
will be rejected. If they are, the lane will run the transactions through its `ProcessLaneHandler`
which is responsible for verifying the transactions against the lane's verification
logic. If any transactions are invalid, the proposal will be rejected. 