# Default Lane

> The Default Lane is the most general and least restrictive lane. The Default
> Lane accepts all transactions that are not accepted by the other lanes, is 
> generally the lowest priority lane, and consumes all blockspace that is not 
> consumed by the other lanes.

## 📖 Overview

The default lane should be used to accept all transactions that are not accepted
by the other lanes.

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
$ go install github.com/skip-mev/block-sdk
```

### 📚 Usage

1. First determine the set of lanes that you want to use in your application. The
available lanes can be found in our **Lane App Store** in `block-sdk/lanes`. In
your base application, you will need to create a `LanedMempool` composed of the
lanes that you want to use.
2. Next, order the lanes by priority. The first lane is the highest priority lane
and the last lane is the lowest priority lane. **It is recommended that the last
lane is the default lane.**
3. You will also need to create a `PrepareProposalHandler` and a 
`ProcessProposalHandler` that will be responsible for preparing and processing 
proposals respectively. Configure the order of the lanes in the
`PrepareProposalHandler` and `ProcessProposalHandler` to match the order of the
lanes in the `LanedMempool`.

NOTE: This example walks through setting up the MEV, Free, and Default Lanes. To
only utilize the default lane, ignore the MEV and Free Lane setup.

```golang
import (
    "github.com/skip-mev/block-sdk/abci"
    "github.com/skip-mev/block-sdk/block/base"
    defaultlane "github.com/skip-mev/block-sdk/lanes/base"
)

...
```

```golang
func NewApp() {
    ...
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
    mevConfig := base.LaneConfig{
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
    freeConfig := base.LaneConfig{
        Logger:        app.Logger(),
        TxEncoder:     app.txConfig.TxEncoder(),
        TxDecoder:     app.txConfig.TxDecoder(),
        MaxBlockSpace: math.LegacyZeroDec(),
        MaxTxs:        0,
    }
    freeLane := free.NewFreeLane(
        freeConfig,
        base.DefaultTxPriority(),
        free.DefaultMatchHandler(),
    )

    // Default lane accepts all other transactions.
    defaultConfig := base.LaneConfig{
        Logger:        app.Logger(),
        TxEncoder:     app.txConfig.TxEncoder(),
        TxDecoder:     app.txConfig.TxDecoder(),
        MaxBlockSpace: math.LegacyZeroDec(),
        MaxTxs:        0,
    }
    defaultLane := defaultlane.NewDefaultLane(defaultConfig)

    // 2. Set up the relateive priority of lanes
    lanes := []block.Lane{
        mevLane,
        freeLane,
        defaultLane,
    }
    mempool := block.NewLanedMempool(app.Logger(), true, lanes...)
    app.App.SetMempool(mempool)

    ...

    // 3. Set up the ante handler.
    anteDecorators := []sdk.AnteDecorator{
		ante.NewSetUpContextDecorator(),
        ...
		utils.NewIgnoreDecorator( // free lane specific set up
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
    for _, lane := range lanes {
        lane.SetAnteHandler(anteHandler)
    }
    app.App.SetAnteHandler(anteHandler)

    // 4. Set the abci handlers on base app
    proposalHandler := abci.NewProposalHandler(
        app.Logger(),
        app.TxConfig().TxDecoder(),
        lanes,
    )
    app.App.SetPrepareProposal(proposalHandler.PrepareProposalHandler())
    app.App.SetProcessProposal(proposalHandler.ProcessProposalHandler())

    ...
}
```
