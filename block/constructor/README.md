# 🎨 Lane Constructor

> 🏗️ Build your own lane in less than 10 minutes using the Lane Constructor

## 💡 Overview

The Lane Constructor is a generic implementation of a lane. It comes out of the 
box with default implementations for all the required interfaces. It is meant to
be used as a starting point for building your own lane. 

## 🤔 How to use it

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
5. [**OPTIONAL**] Users can optionally define their own `CheckOrderHandler`, which
is responsible for determining whether transactions that are included in a proposal
and belong to a given lane are ordered correctly in a block proposal. This is useful
for lanes that require a specific ordering of transactions in a proposal.
6. [**OPTIONAL**] Users can optionally define their own `ProcessLaneHandler`, which
is responsible for processing transactions that are included in block proposals.
In the case where a custom `PrepareLaneHandler` is defined, a custom `ProcessLaneHandler`
will likely follow. This will allow a proposal to be verified against the custom
block building logic.

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
utilize the mempool, you must implement a `TxPriority[C]` struct that does the
following:

- Implements a `GetTxPriority` method that returns the priority (as defined
by the type `[C]`) of a given transaction.
- Implements a `Compare` method that returns the relative priority of two
transactions. If the first transaction has a higher priority, the method
should return -1, if the second transaction has a higher priority, the method
should return 1, otherwise the method should return 0.
- Implements a `MinValue` method that returns the minimum priority value
that a transaction can have.

The default implementation can be found in `block/constructor/mempool.go`. What if 
we wanted to prioritize transactions by the amount they have staked on chain? Well
we could do something like the following:

```golang
// CustomTxPriority returns a TxPriority that prioritizes transactions by the
// amount they have staked on chain. This means that transactions with a higher
// amount staked will be prioritized over transactions with a lower amount staked.
func (p *CustomTxPriority) CustomTxPriority() TxPriority[string] {
    return TxPriority[string]{
        GetTxPriority: func(ctx context.Context, tx sdk.Tx) string {
            // Get the signer of the transaction.
            signer := p.getTransactionSigner(tx)

            // Get the total amount staked by the signer on chain.
            // This is abstracted away in the example, but you can
            // implement this using the staking keeper.
            totalStake, err := p.getTotalStake(ctx, signer)
            if err != nil {
                return ""
            }

            return totalStake.String()
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

To utilize this mempool in a lane, all you have to then do is pass in the
`TxPriority[C]` to the `NewLaneMempool` function.

```golang
// Pseudocode for creating the custom tx priority
priorityCfg := NewPriorityConfig(
    stakingKeeper,
    accountKeeper,
    ...
)


// define your mempool that orders transactions by on chain stake
mempool := constructor.NewMempool[string](
    priorityCfg.CustomTxPriority(),
    cfg.TxEncoder,
    cfg.MaxTxs,
)

// Initialize your lane with the mempool
lane := constructor.NewLaneConstructor(
    cfg,
    LaneName,
    mempool,
    constructor.DefaultMatchHandler(),
)
```

### 3. MatchHandler

> 🔒 `MatchHandler` Invarients
> 
> The handler assumes that the transactions passed into the function are already
> ordered respecting the lane's ordering rules and respecting the ordering rules
> of the mempool relative to the lanes it has. This means that the transactions
> should already be in contiguous order.

MatchHandler is utilized to determine if a transaction should be included in 
the lane. This function can be a stateless or stateful check on the transaction.
The default implementation can be found in `block/constructor/handlers.go`.

The match handler can be as custom as desired. Following the example above, if 
we wanted to make a lane that only accepts transactions if they have a large 
amount staked, we could do the following:

```golang
// CustomMatchHandler returns a custom implementation of the MatchHandler. It
// matches transactions that have a large amount staked. These transactions 
// will then be charged no fees at execution time.
//
// NOTE: This is a stateful check on the transaction. The details of how to
// implement this are abstracted away in the example, but you can implement
// this using the staking keeper.
func (h *Handler) CustomMatchHandler() block.MatchHandler {
    return func(ctx sdk.Context, tx sdk.Tx) bool {
        if !h.IsStakingTx(tx) {
            return false
        }

        signer, err := getTxSigner(tx)
        if err != nil {
            return false
        }

        stakedAmount, err := h.GetStakedAmount(signer)
        if err != nil {
            return false
        }

        return stakeAmount.GT(h.Threshold)
    }
}
```

If we wanted to create the lane using the custom match handler along with the 
custom mempool, we could do the following:

```golang
// Pseudocode for creating the custom match handler
handler := NewHandler(
    stakingKeeper,
    accountKeeper,
    ...
)

// define your mempool that orders transactions by on chain stake
mempool := constructor.NewMempool[string](
    priorityCfg.CustomTxPriority(),
    cfg.TxEncoder,
    cfg.MaxTxs,
)

// Initialize your lane with the mempool
lane := constructor.NewLaneConstructor(
    cfg,
    LaneName,
    mempool,
    handler.CustomMatchHandler(),
)
```

### Notes on Steps 4-6

Although not required, if you implement any single custom handler, whether it's
the `PrepareLaneHandler`, `ProcessLaneHandler`, or `CheckOrderHandler`, you must
implement all of them. This is because the default implementation of the lane
constructor will call all of these handlers. If you do not implement all of them,
the lane may have unintended behavior.

### 4. [OPTIONAL] PrepareLaneHandler

> 🔒 `PrepareLaneHandler` Invarients
> 
> Transactions should be reaped respecting the priority mechanism of the lane. 
> By default this is the TxPriority object used to initialize the lane's mempool.

The `PrepareLaneHandler` is an optional field you can set on the lane constructor.
This handler is responsible for the transaction selection logic when a new proposal
is requested. The default implementation should fit most use cases and can be found
in `block/constructor/handlers.go`.

The handler should return the following for a given lane:

1. The transactions to be included in the block proposal.
2. The transactions to be removed from the mempool.
3. An error if the lane is unable to prepare a block proposal.

```golang
// PrepareLaneHandler is responsible for preparing transactions to be included 
// in the block from a given lane. Given a lane, this function should return 
// the transactions to include in the block, the transactions that must be 
// removed from the lane, and an error if one occurred.
PrepareLaneHandler func(
    ctx sdk.Context,
    proposal BlockProposal,
    maxTxBytes int64,
) (txsToInclude [][]byte, txsToRemove []sdk.Tx, err error)
```

The default implementation is simple. It will continue to select transactions
from its mempool under the following criteria:

1. The transactions is not already included in the block proposal.
2. The transaction is valid and passes the AnteHandler check.
3. The transaction is not too large to be included in the block.

If a more involved selection process is required, you can implement your own
`PrepareLaneHandler` and and set it after creating the lane constructor.

```golang
customLane := constructor.NewLaneConstructor(
    cfg,
    LaneName,
    mempool,
    handler.CustomMatchHandler(),
)

customLane.SetPrepareLaneHandler(customlane.PrepareLaneHandler())
```

### 5. [OPTIONAL] CheckOrderHandler

> 🔒 `CheckOrderHandler` Invarients
> 
> The CheckOrderHandler must ensure that transactions included in block proposals
> only include transactions that are in contiguous order respecting the lane's
> ordering rules and respecting the ordering rules of the mempool relative to the
> lanes it has. This means that all transactions that belong to the same lane, must
> be right next to each other in the block proposal. Additionally, the relative priority
> of each transaction belonging to the lane must be respected.

The `CheckOrderHandler` is an optional field you can set on the lane constructor. 


### 6. [OPTIONAL] ProcessLaneHandler

> 🔒 `ProcessLaneHandler` Invarients
> 
> The handler assumes that the transactions passed into the function are already
> ordered respecting the lane's ordering rules and respecting the ordering rules
> of the mempool relative to the lanes it has. This means that the transactions
> should already be in contiguous order.

The `ProcessLaneHandler` is an optional field you can set on the lane constructor.
This handler is responsible for verifying the transactions in the block proposal
that belong to the lane. The default implementation should fit most use cases and
can be found in `block/constructor/handlers.go`.


```golang
// ProcessLaneHandler is responsible for processing transactions that are 
// included in a block and belong to a given lane. ProcessLaneHandler is 
// executed after CheckOrderHandler so the transactions passed into this 
// function SHOULD already be in order respecting the ordering rules of the 
// lane and respecting the ordering rules of mempool relative to the lanes it has.
ProcessLaneHandler func(ctx sdk.Context, txs []sdk.Tx) ([]sdk.Tx, error)
```

Given the invarients above, the default implementation is simple. It will
continue to verify transactions in the block proposal under the following
criteria:

1. If a transaction matches to this lane, verify it and continue. If it is not
valid, return an error.
2. If a transaction does not match to this lane, return the remaining transactions.