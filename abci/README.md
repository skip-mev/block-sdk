# Block SDK Proposals

> 🤓 Learn and read all about how proposals are constructed and verified using
> the Block SDK

## 📖 Overview

The Block SDK is a framework for building smarter blocks. The Block SDK is built
harnessing the power of ABCI++ which is a new ABCI implementation that allows
for more complex and expressive applications to be built on top of the Cosmos SDK.
The process of building and verifiying proposals can be broken down into two
distinct parts: 

1. Preparing a proposal during `PrepareProposal`.
2. Processing a proposal during `ProcessProposal`.

The Block SDK provides a framework for building and verifying proposals by
segmenting a single block into multiple lanes. Each lane can be responsible for
proposing and verifying specific types of transaction. The Block SDK provides
a default implementation of a lane that can be used to build and verify proposals
similar to how they are built and verified in the Cosmos SDK today while also
providing a framework for building more complex lanes that can be used to build
and verify much more complex proposals.

## 🤔 How does it work

### 🔁 Transaction Lifecycle

The best way to understand how lanes work is to first understand the lifecycle 
of a transaction. A transaction begins its lifecycle when it is first signed and
broadcasted to a chain. After it is broadcasted to a validator, it will be checked
in `CheckTx` by the base application. If the transaction is valid, it will be
inserted into the applications mempool. 

The transaction then waits in the mempool until a new block needs to be proposed.
When a new block needs to be proposed, the application will call `PrepareProposal`
(which is a new ABCI++ addition) to request a new block from the current 
proposer. The proposer will look at what transactions currently waiting to 
be included in a block by looking at their mempool. The proposer will then 
iteratively select transactions until the block is full. The proposer will then
send the block to other validators in the network. 

When a validator receives a proposed block, the validator will first want to 
verify the contents of the block before signing off on it. The validator will 
call `ProcessProposal` to verify the contents of the block. If the block is 
valid, the validator will sign off on the block and broadcast their vote to the 
network. If the block is invalid, the validator will reject the block. Once a 
block is accepted by the network, it is committed and the transactions that 
were included in the block are removed from the validator's mempool (as they no
longer need to be considered).

### 🛣️ Lane Lifecycle

After a transaction is verified in `CheckTx`, it will attempt to be inserted 
into the `LanedMempool`. A `LanedMempool` is composed of several distinct `Lanes`
that have the ability to store their own transactions. The `LanedMempool` will 
insert the transaction into all lanes that will accept it. The criteria for 
whether a lane will accept a transaction is defined by the lane's 
`MatchHandler`. The default implementation of a `MatchHandler` will accept all transactions.


When a new block is proposed, the `PrepareProposalHandler` will iteratively call
`PrepareLane` on each lane (in the order in which they are defined in the
`LanedMempool`). The `PrepareLane` method is anaolgous to `PrepareProposal`. Calling
`PrepareLane` on a lane will trigger the lane to reap transactions from its mempool
and add them to the proposal (given they are valid respecting the verification rules
of the lane).

When proposals need to be verified in `ProcessProposal`, the `ProcessProposalHandler`
defined in `abci/abci.go` will call `ProcessLane` on each lane in the same order
as they were called in the `PrepareProposalHandler`. Each subsequent call to
`ProcessLane` will filter out transactions that belong to previous lanes. A given
lane's `ProcessLane` will only verify transactions that belong to that lane.

> **Scenario**
> 
> Let's say we have a `LanedMempool` composed of two lanes: `LaneA` and `LaneB`.
> `LaneA` is defined first in the `LanedMempool` and `LaneB` is defined second.
> `LaneA` contains transactions `Tx1` and `Tx2` and `LaneB` contains transactions
> `Tx3` and `Tx4`.


When a new block needs to be proposed, the `PrepareProposalHandler` will call
`PrepareLane` on `LaneA` first and `LaneB` second. When `PrepareLane` is called
on `LaneA`, `LaneA` will reap transactions from its mempool and add them to the
proposal. Same applies for `LaneB`. Say `LaneA` reaps transactions `Tx1` and `Tx2`
and `LaneB` reaps transactions `Tx3` and `Tx4`. This gives us a proposal composed
of the following:

* `Tx1`, `Tx2`, `Tx3`, `Tx4`

When the `ProcessProposalHandler` is called, it will call `ProcessLane` on `LaneA`
with the proposal composed of `Tx1`, `Tx2`, `Tx3`, and `Tx4`. `LaneA` will then
verify `Tx1` and `Tx2` and return the remaining transactions - `Tx3` and `Tx4`. 
The `ProcessProposalHandler` will then call `ProcessLane` on `LaneB` with the
remaining transactions - `Tx3` and `Tx4`. `LaneB` will then verify `Tx3` and `Tx4`
and return no remaining transactions.

## 🏗️ Setup

> **Note**
> 
> For a more in depth example of how to use the Block SDK, check out our
> example application in `block-sdk/tests/app/app.go`.

### 📦 Dependencies

The Block SDK is built on top of the Cosmos SDK. The Block SDK is currently
compatible with Cosmos SDK versions greater than or equal to `v0.47.0`.

### 📥 Installation

To install the Block SDK, run the following command:

```bash
go get github.com/skip-mev/block-sdk/abci
```

### 📚 Usage

First determine the set of lanes that you want to use in your application. The available
lanes can be found in our **Lane App Store** in `block-sdk/lanes`. In your base
application, you will need to create a `LanedMempool` composed of the lanes that
you want to use. You will also need to create a `PrepareProposalHandler` and a
`ProcessProposalHandler` that will be responsible for preparing and processing 
proposals respectively. 

```golang
// 1. Create the lanes.
//
// NOTE: The lanes are ordered by priority. The first lane is the highest priority
// lane and the last lane is the lowest priority lane. Top of block lane allows
// transactions to bid for inclusion at the top of the next block.
//
// For more information on how to utilize the LaneConfig please
// visit the README in block-sdk/block/base.
//
// MEV lane hosts an action at the top of the block.
mevConfig := constructor.LaneConfig{
    Logger:        app.Logger(),
    TxEncoder:     app.txConfig.TxEncoder(),
    TxDecoder:     app.txConfig.TxDecoder(),
    MaxBlockSpace: math.LegacyZeroDec(), 
    MaxTxs:        0,
}
mevLane := mev.NewMEVLane(
    mevConfig,
    mev.NewDefaultAuctionFactory(app.txConfig.TxDecoder()),
)

// Free lane allows transactions to be included in the next block for free.
freeConfig := constructor.LaneConfig{
    Logger:        app.Logger(),
    TxEncoder:     app.txConfig.TxEncoder(),
    TxDecoder:     app.txConfig.TxDecoder(),
    MaxBlockSpace: math.LegacyZeroDec(),
    MaxTxs:        0,
}
freeLane := free.NewFreeLane(
    freeConfig,
    constructor.DefaultTxPriority(),
    free.DefaultMatchHandler(),
)

// Default lane accepts all other transactions.
defaultConfig := constructor.LaneConfig{
    Logger:        app.Logger(),
    TxEncoder:     app.txConfig.TxEncoder(),
    TxDecoder:     app.txConfig.TxDecoder(),
    MaxBlockSpace: math.LegacyZeroDec(),
    MaxTxs:        0,
}
defaultLane := base.NewStandardLane(defaultConfig)

// Set the lanes into the mempool.
lanes := []block.Lane{
    mevLane,
    freeLane,
    defaultLane,
}
mempool := block.NewLanedMempool(app.Logger(), true, lanes...)
app.App.SetMempool(mempool)

...

anteHandler := NewAnteHandler(options)

// Set the lane ante handlers on the lanes.
for _, lane := range lanes {
    lane.SetAnteHandler(anteHandler)
}
app.App.SetAnteHandler(anteHandler)

// Set the abci handlers on base app
proposalHandler := abci.NewProposalHandler(
    app.Logger(),
    app.TxConfig().TxDecoder(),
    lanes,
)
app.App.SetPrepareProposal(proposalHandler.PrepareProposalHandler())
app.App.SetProcessProposal(proposalHandler.ProcessProposalHandler())
```