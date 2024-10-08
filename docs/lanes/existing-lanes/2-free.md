# Free Lane

The `Free Lane` allows certain transactions to be included in a block without paying fees. This lane can be used to encourage certain behaviors on the chain, such as staking, governance, or others.

If you have not already, this assumes you have completed the [General Setup](../../0-integrate-the-sdk.md) guide first!

Please reach out to us on [**discord**](skip.build/discord) if you need help!


### 📖 Overview

The free lane closely follows the block building logic of the default lane, with exception for the following:

- Transactions can only be included in the free lane if they are considered free (as defined by the lane's `MatchHandler`). The default implementation matches transactions to the free lane iff the transaction is staking related (e.g. stake, re-delegate, etc.).
- By default, the ordering of transactions in the free lane is based on the transaction's fee amount (highest to lowest). However, this can be overridden to support ordering mechanisms that are not based on fee amount (e.g. ordering based on the user's on-chain stake amount).

The free lane implements the same `ABCI++` interface as the other lanes, and does the same verification logic as the [default lane](../../0-integrate-the-sdk.md). The free lane's `PrepareLane` handler will reap transactions from the lane up to the `MaxBlockSpace` limit, and the `ProcessLane` handler will ensure that the transactions are ordered based on their fee amount (by default) and pass the same checks done in `PrepareLane`.

### 📖 Set Up [10 mins]

**At a high level, to integrate the MEV Lane, chains must:**

1. Be using Cosmos SDK version or higher `v0.47.0`.
2. Import and configure the `Free Lane` (alongside any other desired lanes) into their base app.
3. Import and configure the Block SDK mempool into their base app.
4. Import and configure the Block SDK `Prepare` / `Process` proposal handlers into their base app.

# 🏗️ Free Lane Setup

## 📦 Dependencies

The Block SDK is built on top of the Cosmos SDK. The Block SDK is currently
compatible with Cosmos SDK versions greater than or equal to `v0.47.0`.

### Release Compatibility Matrix

| Block SDK Version | Cosmos SDK |
| :---------------: | :--------: |
|     `v1.x.x`      | `v0.47.x`  |
|     `v2.x.x`      | `v0.50.x`  |

## 📥 Installation

To install the Block SDK, run the following command:

```bash
$ go install github.com/skip-mev/block-sdk
```

## 📚 Usage

1. First determine the set of lanes that you want to use in your application.
   In your base application, you will need to create a `LanedMempool` composed of the
   lanes you want to use. _The free lane should not exist on its own. At minimum, it
   is recommended that the free lane is paired with the default lane._
2. Next, order the lanes by priority. The first lane is the highest priority lane
   and the last lane is the lowest priority lane.
3. Set up your `FeeDeductorDecorator` to ignore the free lane where ever you
   initialize your `AnteHandler`. This will ensure that the free lane is not
   subject to deducting transaction fees.
4. You will also need to create a `PrepareProposalHandler` and a
   `ProcessProposalHandler` that will be responsible for preparing and processing
   proposals respectively. Configure the order of the lanes in the
   `PrepareProposalHandler` and `ProcessProposalHandler` to match the order of the
   lanes in the `LanedMempool`.

NOTE: This example walks through setting up the Free and Default lanes.

```golang
import (
    "github.com/skip-mev/block-sdk/abci"
    "github.com/skip-mev/block-sdk/block/base"
    defaultlane "github.com/skip-mev/block-sdk/lanes/base"
    freelane "github.com/skip-mev/block-sdk/lanes/free"
)

...

func NewApp() {
    ...
    // 1. Create the lanes.
    //
    // NOTE: The lanes are ordered by priority. The first lane is the highest priority
    // lane and the last lane is the lowest priority lane. Top of block lane allows
    // transactions to bid for inclusion at the top of the next block.
    //
    // For more information on how to utilize the LaneConfig please
    // visit the README in docs.skip.money/chains/lanes/build-your-own-lane#-lane-config.
    //
    // Set up the configuration of the free lane and instantiate it.
    freeConfig := base.LaneConfig{
        Logger:        app.Logger(),
        TxEncoder:     app.txConfig.TxEncoder(),
        TxDecoder:     app.txConfig.TxDecoder(),
        MaxBlockSpace: math.LegacyZeroDec(),
        MaxTxs:        0,
    }
    freeLane := freelane.NewFreeLane(freeConfig, base.DefaultTxPriority(), freelane.DefaultMatchHandler())

    // Default lane accepts all transactions.
    defaultConfig := base.LaneConfig{
        Logger:        app.Logger(),
        TxEncoder:     app.txConfig.TxEncoder(),
        TxDecoder:     app.txConfig.TxDecoder(),
        MaxBlockSpace: math.LegacyZeroDec(),
        MaxTxs:        0,
    }
    defaultLane := defaultlane.NewDefaultLane(defaultConfig)

    // 2. Set up the relative priority of lanes
    lanes := []block.Lane{
        freeLane,
        defaultLane,
    }
    mempool := block.NewLanedMempool(app.Logger(), true, lanes...)
    app.App.SetMempool(mempool)

    ...

    // 3. Set up the ante handler.
    //
    // This will allow any transaction that matches the to the free lane to
    // be processed without paying any fees.
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

    // 4. Set the abci handlers on base app
    proposalHandler := abci.NewProposalHandler(
        app.Logger(),
        app.TxConfig().TxDecoder(),
        mempool,
    )
    app.App.SetPrepareProposal(proposalHandler.PrepareProposalHandler())
    app.App.SetProcessProposal(proposalHandler.ProcessProposalHandler())

    ...
}
```
