# 🎨 Lane Constructor

> 🏗️ Build your own lane in less than 30 minutes using the Lane Constructor

## 💡 Overview

The Lane Constructor is a generic implementation of a lane. It comes out of the 
box with default implementations for all the required interfaces. It is meant to
be used as a starting point for building your own lane. 

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

## How to use it

There are **three** critical
components to the Lane Constructor:

1. The lane configuration (`LaneConfig`) which determines the basic properties 
of the lane including the maximum block space that the lane can fill up.
2. The lane mempool (`LaneMempool`) which is responsible for storing 
transactions as they are being verified and are waiting to be included in proposals.
3. A `MatchHandler` which is responsible for determining whether a transaction should
be accepted to this lane.
4. [**OPTIONAL**] Users can optionally define their own `PrepareLaneHandler`, which
is responsible for reaping transactions from its mempool and adding them to a proposal.
This allows users to customize the order/how transactions are added to a proposal
if any custom block building logic is required.
5. [**OPTIONAL**] Users can optionally define their own `ProcessLaneHandler`, which
is responsible for processing transactions that are included in block proposals.
In the case where a custom `PrepareLaneHandler` is defined, a custom `ProcessLaneHandler`
will likely follow. This will allow a proposal to be verified against the custom
block building logic.
6. [**OPTIONAL**] Users can optionally define their own `CheckOrderHandler`, which
is responsible for determining whether transactions that are included in a proposal
and belong to a given lane are ordered correctly in a block proposal. This is useful
for lanes that require a specific ordering of transactions in a proposal.

### 1. Lane Config

The lane config (`LaneConfig`) is a simple configuration
object that defines the desired amount of block space the lane should
utilize when building a proposal, an antehandler that is used to verify
transactions as they are added/verified to/in a proposal, and more. By default,
we recommend that user's pass in all of the base apps configurations (txDecoder,
logger, etc.). A sample `LaneConfig` might look like the following:

```golang
config := block.LaneConfig{
    Logger: app.Logger(),
    TxDecoder: app.TxDecoder(),
    TxEncoder: app.TxEncoder(),
    AnteHandler: app.AnteHandler(),
    MaxTxs: 0,
    MaxBlockSpace: math.LegacyZeroDec(),
    IgnoreList: []block.Lane{},
}
```

The three most important parameters to set are the `AnteHandler`, `MaxTxs`, and
`MaxBlockSpace`.

### **AnteHandler**

With the default implementation, the `AnteHandler` is responsible for verifying
transactions as they are being considered for a new proposal or are being processed
in a proposal. 

### **MaxTxs**

This sets the maximum number of transactions allowed in the mempool with
the semantics:

* if `MaxTxs` == 0, there is no cap on the number of transactions in the mempool
* if `MaxTxs` > 0, the mempool will cap the number of transactions it stores,
    and will prioritize transactions by their priority and sender-nonce
    (sequence number) when evicting transactions.
* if `MaxTxs` < 0, `Insert` is a no-op.

### **MaxBlockSpace**

MaxBlockSpace is the maximum amount of block space that the lane will attempt to
fill when building a proposal. This parameter may be useful lanes that should be
limited (such as a free or onboarding lane) in space usage. Setting this to 0 
will allow the lane to fill the block with as many transactions as possible.

#### **[OPTIONAL] IgnoreList**

IgnoreList defines the list of lanes to ignore when processing transactions. 
This is useful for when you want lanes to exist after the default lane. For 
example, say there are two lanes: default and free. The free lane should be 
processed after the default lane. In this case, the free lane should be added 
to the ignore list of the default lane. Otherwise, the transactions that belong 
to the free lane will be processed by the default lane (which accepts all 
transactions by default).


### 2. LaneMempool

This is the data structure that is responsible for storing transactions
as they are being verified and are waiting to be included in proposals. `block/constructor/mempool.go`
provides an out-of-the-box implementation that should be used as a starting 
point for building out the mempool and should cover most use cases. To 
utilize the mempool, you must define a `TxPriority[C]` struct that does the
following:

- Implements a `GetTxPriority` method that returns the priority (as defined
by the type `[C]`) of a given transaction.
- Implements a `Compare` method that returns the relative priority of two
transactions. If the first transaction has a higher priority, the method
should return -1, if the second transaction has a higher priority, the method
should return 1, otherwise the method should return 0.
- Implements a `MinValue` method that returns the minimum priority value
that a transaction can have.

The default implementation can be found in `block/constructor/mempool.go`.

```golang
// DefaultTxPriority returns a default implementation of the TxPriority. It prioritizes
// transactions by their fee.
func DefaultTxPriority() TxPriority[string] {
    return TxPriority[string]{
        GetTxPriority: func(goCtx context.Context, tx sdk.Tx) string {
            feeTx, ok := tx.(sdk.FeeTx)
            if !ok {
                return ""
            }

            return feeTx.GetFee().String()
        },
        Compare: func(a, b string) int {
            aCoins, _ := sdk.ParseCoinsNormalized(a)
            bCoins, _ := sdk.ParseCoinsNormalized(b)

            switch {
            case aCoins == nil && bCoins == nil:
                return 0

            case aCoins == nil:
                return -1

            case bCoins == nil:
                return 1

            default:
                switch {
                case aCoins.IsAllGT(bCoins):
                    return 1

                case aCoins.IsAllLT(bCoins):
                    return -1

                default:
                    return 0
                }
            }
        },
        MinValue: "",
    }
}
```

### LaneHandlers
