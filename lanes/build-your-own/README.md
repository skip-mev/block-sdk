# 🏗️ Build-Your-Own Lane Setup

## 📦 Dependencies

The Block SDK is built on top of the Cosmos SDK. The Block SDK is currently
compatible with Cosmos SDK versions greater than or equal to `v0.47.0`.

## 📥 Installation

To install the Block SDK, run the following command:

```bash
$ go install github.com/skip-mev/block-sdk
```

## 🤔 How to use it [30 min]

The Block SDK is designed to be modular and extensible. This means that 
developers can build their own lanes and customize the block building/verification
logic to fit their needs. This guide will walk through the process of building
a custom lane and configuring it in the application. Developers should 
extend the base lane (`block/base/lane.go`) to build their own lanes.

There are **five** components to building a custom lane using the base lane:

1. `Mempool` - The lane's mempool is responsible for storing transactions that 
have been verified and are waiting to be included in proposals.
2. `MatchHandler` - This is responsible for determining whether a transaction 
should belong to this lane.
3. [**OPTIONAL**] `PrepareLaneHandler` - Allows developers to define their own 
handler to customize the how transactions are verified and ordered before they 
are included in a proposal.
4. [**OPTIONAL**] `ProcessLaneHandler` - Allows developers to define their own 
handler for processing transactions that are included in block proposals.
5. `Configuration` - Configure high-level options for your lane.

### 1. 🗄️ Mempool

This is the data structure that is responsible for storing transactions as they 
are being verified and are waiting to be included in proposals. 
`block/base/mempool.go` provides an out-of-the-box implementation that should be
used as a starting point for building out the mempool and should cover most use 
cases. To utilize the mempool, you must implement a `TxPriority[C]` struct that 
does the following:

* Implements a `GetTxPriority` method that returns the priority (as defined
  by the type `[C]`) of a given transaction.
* Implements a `Compare` method that returns the relative priority of two
  transactions. If the first transaction has a higher priority, the method
  should return -1, if the second transaction has a higher priority the method
  should return 1, otherwise the method should return 0.
* Implements a `MinValue` method that returns the minimum priority value
  that a transaction can have.

The default implementation can be found in `block/base/mempool.go` - see `DefaultTxPriority`.

> Scenario
What if we wanted to prioritize transactions by the amount they have staked on 
a chain?

We could do the following:

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

#### Using a Custom TxPriority

To utilize this new priority configuration in a lane, all you have to then do 
is pass in the `TxPriority[C]` to the `NewMempool` function.

```golang
// Create the lane config (explained in more detail below)
laneCfg := NewLaneConfig(
    ...
    MaxTxs: 100,
    ...
)

// Pseudocode for creating the custom tx priority. This is what
// returns your custom tx priority.
priorityCfg := NewPriorityConfig(
    stakingKeeper,
    accountKeeper,
    ...
)


// define your mempool that orders transactions by on-chain stake
mempool := base.NewMempool[string](
    priorityCfg.CustomTxPriority(), // pass in the custom tx priority
    laneCfg.TxEncoder,
    laneCfg.MaxTxs,
)

// Initialize your lane with the mempool
lane := base.NewBaseLane(
    laneCfg,
    LaneName,
    mempool,
    base.DefaultMatchHandler(),
)
```

### 2. 🤝 MatchHandler

`MatchHandler` is utilized to determine if a transaction should be included in 
the lane. **This function can be a stateless or stateful check on the 
transaction!** The default implementation can be found in `block/base/handlers.go`.

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
func (h *Handler) CustomMatchHandler() base.MatchHandler {
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

        // The transaction can only be considered for inclusion if the amount
        // staked is greater than some predetermined threshold.
        return stakeAmount.GT(h.Threshold)
    }
}
```

#### Using a Custom MatchHandler

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
mempool := base.NewMempool[string](
    priorityCfg.CustomTxPriority(),
    cfg.TxEncoder,
    cfg.MaxTxs,
)

// Initialize your lane with the mempool
lane := base.NewBaseLane(
    cfg,
    LaneName,
    mempool, // Custom mempool
    handler.CustomMatchHandler(), // Custom match handler
)
```

### [OPTIONAL] Steps 3-5

The remaining steps walk through the process of creating custom block 
building/verification logic. The default implementation found in 
`block/base/handlers.go` should fit most use cases. Please reference that file 
for more details on the default implementation and whether it fits your use case.

Implementing custom block building/verification logic is a bit more involved 
than the previous steps and is a all or nothing approach. This means that if 
you implement any of the handlers, you must implement all of them in most cases.
 If you do not implement all of them, the lane may have unintended behavior.

### 3. 🛠️ PrepareLaneHandler

The `PrepareLaneHandler` is an optional field you can set on the base lane.
This handler is responsible for the transaction selection logic when a new proposal
is requested.

The handler should return the following for a given lane:

1. The transactions to be included in the block proposal.
2. The transactions to be removed from the lane's mempool.
3. An error if the lane is unable to prepare a block proposal.

When collecting transactions to include in the block proposal, the handler
must respect the lane limits - i.e. the maximum number of bytes and units
of gas that the lane can use in the block proposal. This is defined in the
`limit proposals.LaneLimits` struct.

```golang
type (
    // PrepareLaneHandler is responsible for preparing transactions to be 
    // included in the block from a given lane. Given a lane, this function 
    // should return the transactions to include in the block, the transactions
    // that must be removed from the lane, and an error if one occurred.
    PrepareLaneHandler func(
        ctx sdk.Context,
        proposal proposals.Proposal,
        limit proposals.LaneLimits,
    ) (txsToInclude, txsToRemove []sdk.Tx, err error)

	// LaneLimits defines the constraints for a partial proposal. Each lane must 
    // only propose transactions that satisfy these constraints. Otherwise the 
    // partial proposal update will be rejected.
	LaneLimits struct {
		// MaxTxBytes is the maximum number of bytes allowed in the partial proposal.
		MaxTxBytes int64
		// MaxGasLimit is the maximum gas limit allowed in the partial proposal.
		MaxGasLimit uint64
	}
)
```

The default implementation is simple. It will continue to select transactions 
from its mempool under the following criteria:

1. The transactions is not already included in the block proposal.
2. The transaction is valid and passes the AnteHandler check.
3. The transaction is not too large/gas intensive to be included in the block.

If a more involved selection process is required, you can implement your own 
`PrepareLaneHandler` and and set it after creating the base lane.

```golang
// Pseudocode for creating the custom prepare lane handler
// This assumes that the CustomLane inherits from the base
// lane.
customLane := NewCustomLane(
    cfg,
    mempool,
    handler.CustomMatchHandler(),
)

// Set the custom PrepareLaneHandler on the lane
customLane.SetPrepareLaneHandler(customlane.PrepareLaneHandler())
```

See `lanes/mev/abci.go` for an example of how to set up a custom `PrepareLaneHandler`.

### 4. 🆗 ProcessLaneHandler

The `ProcessLaneHandler` is an optional field you can set on the base lane. 
This handler is responsible for verifying the transactions in the block proposal
that belong to the lane.

```golang
// ProcessLaneHandler is responsible for processing transactions that are 
// included in a block and belong to a given lane. This handler must return an 
// error if the transactions are not correctly ordered, do not belong to this 
// lane, or any other relevant error.
ProcessLaneHandler func(ctx sdk.Context, partialProposal []sdk.Tx) error
```

The `partialProposal` is a slice of transactions that belong to the lane and
are included in the block proposal. The handler should return an error if the
transactions are not correctly ordered, do not belong to this lane, or any
other relevant error.

Given the description above, the default implementation is simple. It will 
continue to verify transactions in the block proposal under the following criteria:

1. All of the transactions included in `partialProposal` must belong to this
lane i.e. they must match the lane's `MatchHandler`.
2. All of the transactions must be ordered respecting the ordering rules of the
mempool i.e. the transactions must be ordered by their priority which is defined
by the `TxPriority[C]` struct.
3. All of the transactions must be valid and pass the AnteHandler check.

Similar to the setup of handlers above, if a more involved verification process 
is required, you can implement your own `ProcessLaneHandler` and and set it 
after creating the base lane.

```golang
// Pseudocode for creating the custom check order handler
// This assumes that the CustomLane inherits from the base
// lane.
customLane := NewCustomLane(
    cfg,
    mempool,
    handler.CustomMatchHandler(),
)

// Set the custom ProcessLaneHandler on the lane
customLane.SetProcessLaneHandler(customlane.ProcessLaneHandler())
```

See `lanes/mev/abci.go` for an example of how to set up a custom `ProcessLaneHandler`.

### 5. 📝 Lane Configuration

The lane config (`LaneConfig`) is a simple configuration object that defines 
the desired amount of block space the lane should utilize when building a 
proposal, an antehandler that is used to verify transactions as they are 
added/verified to/in a proposal, and more. By default, we recommend that user's 
pass in all of the base apps configurations (txDecoder, logger, etc.). A sample 
`LaneConfig` might look like the following:

```golang
config := base.LaneConfig{
    Logger: app.Logger(),
    TxDecoder: app.TxDecoder(),
    TxEncoder: app.TxEncoder(),
    AnteHandler: app.AnteHandler(),
    MaxTxs: 0,
    MaxBlockSpace: math.LegacyZeroDec(),
    IgnoreList: []block.Lane{},
    ModuleLaneFetcher: app.blocksdkKeeper,
}
```

The three most important parameters to set are the `AnteHandler`, `MaxTxs`, and `MaxBlockSpace`.

#### **AnteHandler**

With the default implementation, the `AnteHandler` is responsible for verifying 
transactions as they are being considered for a new proposal or are being 
processed in a proposed block. We recommend user's utilize the same antehandler 
chain that is used in the base app. If developers want a certain `AnteDecorator`
to be ignored if it qualifies for a given lane, they can do so by using the 
`NewIgnoreDecorator` defined in `block/ante.go`.

For example, a free lane might want to ignore the `DeductFeeDecorator` so that
its transactions are not charged any fees. Where ever the `AnteHandler` is 
defined, we could add the following to ignore the `DeductFeeDecorator`:

```golang
anteDecorators := []sdk.AnteDecorator{
    ante.NewSetUpContextDecorator(),
    ...,
    utils.NewIgnoreDecorator(
        ante.NewDeductFeeDecorator(
            options.BaseOptions.AccountKeeper,
            options.BaseOptions.BankKeeper,
            options.BaseOptions.FeegrantKeeper,
            options.BaseOptions.TxFeeChecker,
        ),
        options.FreeLane,
    ),
    ...,
}
```

Anytime a transaction that qualifies for the free lane is being processed, the 
`DeductFeeDecorator` will be ignored and no fees will be deducted!

#### **MaxTxs**

This sets the maximum number of transactions allowed in the mempool with the semantics:

* if `MaxTxs` == 0, there is no cap on the number of transactions in the mempool.
* if `MaxTxs` > 0, the mempool will cap the number of transactions it stores, 
and will prioritize transactions by their priority and sender-nonce 
(sequence number) when evicting transactions.
* if `MaxTxs` < 0, `Insert` is a no-op.

#### **MaxBlockSpace**

MaxBlockSpace is the maximum amount of block space that the lane will attempt 
to fill when building a proposal. This parameter may be useful for lanes that 
should be limited (such as a free or onboarding lane) in space usage. 
Setting this to 0 will allow the lane to fill the block with as many 
transactions as possible.

If a block proposal request has a `MaxTxBytes` of 1000 and the lane has a 
`MaxBlockSpace` of 0.5, the lane will attempt to fill the block with 500 bytes.

#### **[OPTIONAL] IgnoreList**

`IgnoreList` defines the list of lanes to ignore when processing transactions. 
For example, say there are two lanes: default and free. The free lane is 
processed after the default lane. In this case, the free lane should be added 
to the ignore list of the default lane. Otherwise, the transactions that belong 
to the free lane will be processed by the default lane (which accepts all 
transactions by default).


### Set up

Once you have created your custom lane, you can configure it in the application 
by doing the following:

1. Create a custom `LaneConfig` struct that defines the configuration of the lane.
2. Instantiate the lane with the custom `LaneConfig` struct alongside any other 
dependencies (mempool, match handler, etc.).
3. Instantiate a new `LanedMempool` with the custom lane.
4. Set the `LanedMempool` on the `BaseApp` instance.
5. Set up the proposal handlers of the Block SDK to use your lane.
6. That's it! You're done!
