# Integrate the Block SDK

The Block SDK is **open-source software** licensed under MIT. It is free to use, and has existing plug-and-play Lanes that work immediately!

Visit the GitHub repo [here](https://github.com/skip-mev/block-sdk).

We strive to be responsive to questions and issues within 1-2 weeks - please open a GitHub issue or join us on [**discord**](skip.build/discord). Note, we are not currently providing hands-on support for new integrations.


## ⚙️ Architecture [15 mins]

This is a high-level overview of the architecture, please reference [this page](2-how-it-works.md) or the [`Block-SDK` repo](https://github.com/skip-mev/block-sdk) for detailed and up to date info. For those eager to code, feel free to skip this and start down the page at **Set Up**!


### How Were Blocks Constructed pre-Block-SDK?

There are 3 relevant stages of consensus (these are all ABCI++ methods)

- **PrepareProposal**
  - In this step, the consensus-engine (CometBFT, etc.) gives the application all of the transactions it has seen thus far.
  - The app looks over these, performs some app-specific logic, and then gives them back to the consensus-engine. The consensus-engine then creates and broadcasts a proposal containing the transactions sent back from the app.
- **ProcessProposal**
  - In this step, all validators check that the transactions in the proposal are valid, and that the proposal (as a whole) satisfies validity conditions determined by the application
    - If the proposal fails, validators will not vote on the block, and the network will be forced to another round of consensus
    - if the proposal passes, valdiators vote on the block, and the block will become canonical (barring unforeseen events)

### **Application Mempools**

In `v0.47.0` of the cosmos-sdk, **app-side mempools** were added to the SDK. With app-side mempools, validators no longer need to rely on the consensus-engine to keep track of and order all available transactions. Now applications can define their own mempool implementations, that

1. Store all pending (not finalized in a block) transactions
2. Order the set of pending transactions

#### **How does block-building change?**

Now in **PrepareProposal** instead of getting transactions from the consensus-engine, validators can pull transactions from their application-state aware mempools, and prioritize those transactions instead of the consensus-engine's transactions.

**Why is this better?**

- Mempools that are not app-state aware will not have the ability to make state-aware ordering rules. Like

1. All staker transactions are placed at the top of the block
2. All IBC `LightClientUpdate` messages are placed at the top of the block
3. Anything you can think of!!

- The consensus engine's mempool is generally in-efficient.
  - The consensus-engine's mempool does not know when to remove transactions from its own mempool
  - The consensus-engine spends most of its time re-broadcasting transactions between peers, hogging network bandwidth

## Block-SDK!!

The `Block-SDK` defines its own custom implementation of an **app-side mempool**, a `LaneMempool`. The `LaneMempool` is composed of `Lanes`, and handles transaction ingress, ordering, and cleaning.

**transaction ingress**

- The `LanedMempool` constructor defines an ordering of lanes. When a transaction is received by the app, it iterates through all lanes in order and inserts the transaction into the first `Lane` that it belongs in.
  **ordering**
- Each `Lane` of the `LanedMempool` maintains its own ordering of transactions. When the `LanedMempool` routes a transaction to its corresponding `Lane` the `Lane` then inserts the transaction at its designated position with respect to all other transactions in the lane

### PrepareProposal

When the application is instructed to `PrepareProposal` it iterates through its `Lane`s in order, and calls each `Lane`'s `PrepareLane` method. The `Lane.PrepareLane` method collects transactions from a `Lane` and appends those transactions to the set of transactions from previous `Lane`'s `PrepareLane` calls. In other words, each block-proposal is now a collection of the transactions from the `LanedMempool`'s constituent lanes.

### ProcessProposal

When the application receives a proposal, and calls `ProcessProposal`, the app delegates the validation to the `LaneMempool.ProcessLanes` method. Remember, the proposal is composed of transactions from the sub-lanes of the `LaneMempool`, as such, the `LaneMempool` can route each `Lane`'s contribution to the Proposal to that `Lane` for validation. The proposal passes iff all `Lane`'s contributions are valid.

#### ⚠️ NOTE ⚠️

A block constructed from a `LaneMempool`'s `PrepareLanes` method must always pass that `LaneMempool`'s `ProcessLanes` method, otherwise, the chain will fail to produce blocks!! These functions are consensus critical, so practice caution when implementing them!!


## 📖 Set Up [20 mins]

To get set up, we're going to implement the `Default Lane`, which is the **most general and least restrictive** that accepts all transactions. This will cause **no changes** to your chain functionality, but will prepare you to add `lanes` with more functionality afterwards!

The default lane mirrors how CometBFT creates proposals today.

- It does a basic check to ensure that the transaction is valid.
- Orders the transactions based on tx fee amount (highest to lowest).
- The `PrepareLane` handler will reap transactions from the lane up to the `MaxBlockSpace` limit
- The `ProcessLane` handler will ensure that the transactions are ordered based on their fee amount and pass the same checks done in `PrepareLane`.

<!-- TODO: create script -->

# 🏗️ Default Lane Setup

## 📦 Dependencies

The Block SDK is built on top of the Cosmos SDK. The Block SDK is currently
compatible with Cosmos SDK versions greater than or equal to `v0.47.0`.

### Release Compatibility Matrix

| Block SDK Version | Cosmos SDK |
| :---------------: | :--------: |
|     `v1.x.x`      | `v0.47.x`  |
|     `v2.x.x`      | `v0.50.x`  |

## 📥 Adding the Block SDK to Your Project

```bash
$ go get github.com/skip-mev/block-sdk
```

## 📚 Usage

1. First determine the set of lanes that you want to use in your application. This guide only sets up the `default lane`

```golang
import (
    "github.com/skip-mev/block-sdk/abci"
    "github.com/skip-mev/block-sdk/block/base"
    defaultlane "github.com/skip-mev/block-sdk/lanes/base"
)

    // 1. Create the lanes.
    //
    // NOTE: The lanes are ordered by priority. The first lane is the highest priority
    // lane and the last lane is the lowest priority lane. Top of block lane allows
    // transactions to bid for inclusion at the top of the next block.
    //
    // For more information on how to utilize the LaneConfig please
    // visit the README in docs.skip.money/chains/lanes/build-your-own-lane#-lane-config.
    //
    // Default lane accepts all transactions.

func NewApp() {
    ...
    defaultConfig := base.LaneConfig{
        Logger:        app.Logger(),
        TxEncoder:     app.txConfig.TxEncoder(),
        TxDecoder:     app.txConfig.TxDecoder(),
        MaxBlockSpace: math.LegacyZeroDec(),
        MaxTxs:        0,
    }
    defaultLane := defaultlane.NewDefaultLane(defaultConfig)
    // TODO(you): Add more Lanes!!!
```

2. In your base application, you will need to create a `LanedMempool` composed
   of the `lanes` you want to use.

```golang
        // 2. Set up the relative priority of lanes
        lanes := []block.Lane{
            defaultLane,
        }
        mempool := block.NewLanedMempool(app.Logger(), true, lanes...)
        app.App.SetMempool(mempool)
```

3. Next, order the lanes by priority. The first lane is the highest priority lane
   and the last lane is the lowest priority lane. **It is recommended that the last
   lane is the default lane.**

```golang
    // 3. Set up the ante handler.
    anteDecorators := []sdk.AnteDecorator{
		ante.NewSetUpContextDecorator(),
        ...
		utils.NewIgnoreDecorator(
			ante.NewDeductFeeDecorator(
				options.BaseOptions.AccountKeeper,
				options.BaseOptions.BankKeeper,
				options.BaseOptions.FeegrantKeeper,
				options.BaseOptions.TxFeeChecker,
			),
			options.FreeLane,
		),
        ...
	}

    anteHandler := sdk.ChainAnteDecorators(anteDecorators...)

    // Set the lane ante handlers on the lanes.
    //
    // NOTE: This step is very important. Without the antehandlers, lanes will not
    // be able to verify transactions.
    for _, lane := range lanes {
        lane.SetAnteHandler(anteHandler)
    }
    app.App.SetAnteHandler(anteHandler)
```

4. You will also need to create a `PrepareProposalHandler` and a
   `ProcessProposalHandler` that will be responsible for preparing and processing
   proposals respectively. Configure the order of the lanes in the
   `PrepareProposalHandler` and `ProcessProposalHandler` to match the order of the
   lanes in the `LanedMempool`.

```golang
    // 4. Set the abci handlers on base app
    // Create the LanedMempool's ProposalHandler
    proposalHandler := abci.NewProposalHandler(
        app.Logger(),
        app.TxConfig().TxDecoder(),
        mempool,
    )

    // set the Prepare / ProcessProposal Handlers on the app to be the `LanedMempool`'s
    app.App.SetPrepareProposal(proposalHandler.PrepareProposalHandler())
    app.App.SetProcessProposal(proposalHandler.ProcessProposalHandler())
```

### 💅 Next step: implement other `lanes`

See the [Mev Lane](lanes/existing-lanes/1-mev.md) and select the `lanes` you want, or [Build Your Own](lanes/1-build-your-own-lane.md).
