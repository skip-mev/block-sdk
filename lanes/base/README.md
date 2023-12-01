# 🏗️ Default Lane Setup

## 📦 Dependencies

The Block SDK is built on top of the Cosmos SDK. The Block SDK is currently
compatible with Cosmos SDK versions greater than or equal to `v0.47.0`.

### Release Compatibility Matrix

| Block SDK Version | Cosmos SDK |
| :---------: | :--------: |
|   `v1.x.x`    |  `v0.47.x`   |
|   `v2.x.x`    |  `v0.50.x`   |

## 📥 Installation

To install the Block SDK, run the following command:

```bash
$ go install github.com/skip-mev/block-sdk
```

## 📚 Usage

> Note: Please visit [app.go](../../tests/app/lanes.go) to see a sample base app set up.

1. First determine the set of lanes that you want to use in your application. The
available lanes can be found in our 
[Lane App Store](https://docs.skip.money/chains/lanes/existing-lanes/default). 
This guide only sets up the `default lane`
2. In your base application, you will need to create a `LanedMempool` composed 
of the `lanes` you want to use.
3. Next, order the lanes by priority. The first lane is the highest priority lane
and the last lane is the lowest priority lane. **It is recommended that the last
lane is the default lane.**
4. You will also need to create a `PrepareProposalHandler` and a 
`ProcessProposalHandler` that will be responsible for preparing and processing 
proposals respectively. Configure the order of the lanes in the
`PrepareProposalHandler` and `ProcessProposalHandler` to match the order of the
lanes in the `LanedMempool`.
5. Configure your `app.go` to include the following:

```golang
import (
    "github.com/skip-mev/block-sdk/abci"
    "github.com/skip-mev/block-sdk/block/base"
    defaultlane "github.com/skip-mev/block-sdk/lanes/base"
)

...

func NewApp() {
    ...
    // 1. Create the lanes.
    //
    // NOTE: The lanes are ordered by priority. The first lane is the highest priority
    // lane and the last lane is the lowest priority lane.
    //
    // For more information on how to utilize the LaneConfig please
    // visit the README in docs.skip.money/chains/lanes/build-your-own-lane#-lane-config.
    //
    // Default lane accepts all transactions.
    defaultConfig := base.LaneConfig{
        Logger:        app.Logger(),
        TxEncoder:     app.txConfig.TxEncoder(),
        TxDecoder:     app.txConfig.TxDecoder(),
        MaxBlockSpace: math.LegacyZeroDec(),
        MaxTxs:        5000,
    }
    defaultLane := defaultlane.NewDefaultLane(defaultConfig, base.DefaultMatchHandler())

    // 2. Set up the relative priority of lanes
    lanes := []block.Lane{
        defaultLane,
    }
    mempool := block.NewLanedMempool(app.Logger(), lanes)
    app.App.SetMempool(mempool)

    ...

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
