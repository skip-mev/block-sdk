# 🏗️ MEV Lane Setup

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

1. This guide assumes you have already set up the [Block SDK (and the default lane)](https://docs.skip.money/chains/overview)
2. You will need to instantiate the `x/builder` module into your application. This
module is responsible for processing auction transactions and distributing revenue
to the auction house. The `x/builder` module is also responsible for ensuring the
validity of auction transactions. *The `x/builder` module should not exist on its
own. **This is the most intensive part of the set up process.**
3. Next, add the MEV lane into the `lane` object on your `app.go`. The first 
lane is the highest priority lane and the last lane is the lowest priority lane.
Since the MEV lane is meant to auction off the top of the block, **it should be 
the highest priority lane**. The default lane should follow.
4. You will also need to create a `PrepareProposalHandler` and a 
`ProcessProposalHandler` that will be responsible for preparing and processing 
proposals respectively. Configure the order of the lanes in the
`PrepareProposalHandler` and `ProcessProposalHandler` to match the order of the
lanes in the `LanedMempool`.

NOTE: This example walks through setting up the MEV and Default lanes.

1. Import the necessary dependencies into your application. This includes the
   Block SDK proposal handlers + mempool, keeper, builder types, and builder 
   module. This tutorial will go into more detail into each of the dependencies.

   ```go
   import (
    ...
    "github.com/skip-mev/block-sdk/abci"
    "github.com/skip-mev/block-sdk/lanes/mev"
    "github.com/skip-mev/block-sdk/lanes/base"
    buildermodule "github.com/skip-mev/block-sdk/x/builder"
    builderkeeper "github.com/skip-mev/block-sdk/x/builder/keeper"
    buildertypes "github.com/skip-mev/block-sdk/x/builder/types"
    builderante "github.com/skip-mev/block-sdk/x/builder/ante"
     ...
   )
   ```

2. Add your module to the the `AppModuleBasic` manager. This manager is in
   charge of setting up basic, non-dependent module elements such as codec
   registration and genesis verification. This will register the special
   `MsgAuctionBid` message. When users want to bid for top of block execution,
   they will submit a transaction - which we call an auction transaction - that
   includes a single `MsgAuctionBid`. We prevent any other messages from being
   included in auction transaction to prevent malicious behavior - such as front
   running or sandwiching.

   ```go
   var (
     ModuleBasics = module.NewBasicManager(
       ...
       buildermodule.AppModuleBasic{},
     )
     ...
   )
   ```

3. The builder `Keeper` is MEV lane's gateway to processing special `MsgAuctionBid`
   messages that allow users to participate in the top of block auction, distribute
   revenue to the auction house, and ensure the validity of auction transactions.

    a. First add the keeper to the app's struct definition. We also want to add 
    MEV lane's custom checkTx handler to the app's struct definition. This will 
    allow us to override the default checkTx handler to process bid transactions
    before they are inserted into the `LanedMempool`. NOTE: The custom handler 
    is required as otherwise the auction can be held hostage by a malicious
    users.

    ```go
    type App struct {
    ...
    // BuilderKeeper is the keeper that handles processing auction transactions
    BuilderKeeper         builderkeeper.Keeper

    // Custom checkTx handler
    checkTxHandler mev.CheckTx
    }
    ```

    b. Add the builder module to the list of module account permissions. This will
    instantiate the builder module account on genesis.

    ```go
    maccPerms = map[string][]string{
    builder.ModuleName: nil,
    ...
    }
    ```

    c. Instantiate the Block SDK's `LanedMempool` with the application's 
    desired lanes.

    ```go
    // 1. Create the lanes.
    //
    // NOTE: The lanes are ordered by priority. The first lane is the
    // highest priority
    // lane and the last lane is the lowest priority lane. Top of block 
    // lane allows transactions to bid for inclusion at the top of the next block.
    //
    // For more information on how to utilize the LaneConfig please
    // visit the README in docs.skip.money/chains/lanes/build-your-own-lane#-lane-config.
    //
    // MEV lane hosts an auction at the top of the block.
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

    // default lane accepts all other transactions.
    defaultConfig := base.LaneConfig{
        Logger:        app.Logger(),
        TxEncoder:     app.txConfig.TxEncoder(),
        TxDecoder:     app.txConfig.TxDecoder(),
        MaxBlockSpace: math.LegacyZeroDec(),
        MaxTxs:        0,
    }
    defaultLane := base.NewStandardLane(defaultConfig)

    // 2. Set up the relateive priority of lanes
    lanes := []block.Lane{
        mevLane,
        defaultLane,
    }
    mempool := block.NewLanedMempool(app.Logger(), true, lanes...)
    app.App.SetMempool(mempool)
    ```

    d. Add the `x/builder` module's `AuctionDecorator` to the ante-handler 
    chain. The `AuctionDecorator` is an AnteHandler decorator that enforces 
    various chain configurable MEV rules.

    ```go
    anteDecorators := []sdk.AnteDecorator{
        ante.NewSetUpContextDecorator(), 
        ...
        builderante.NewBuilderDecorator(
        options.BuilderKeeper, 
        options.TxEncoder, 
        options.TOBLane, 
        options.Mempool,
        ),
    }

    anteHandler := sdk.ChainAnteDecorators(anteDecorators...)
    app.SetAnteHandler(anteHandler)

    // Set the antehandlers on the lanes.
    //
    // NOTE: This step is required as otherwise the lanes will not be able to
    // process auction transactions.
    for _, lane := range lanes {
        lane.SetAnteHandler(anteHandler)
    }
    app.App.SetAnteHandler(anteHandler)
    ```

    e. Instantiate the builder keeper, store keys, and module manager. Note, be
    sure to do this after all the required keeper dependencies have been instantiated.

    ```go
    keys := storetypes.NewKVStoreKeys(
        buildertypes.StoreKey,
        ...
    )

    ...
    app.BuilderKeeper := builderkeeper.NewKeeper(
        appCodec,
        keys[buildertypes.StoreKey],
        app.AccountKeeper,
        app.BankKeeper,
        app.DistrKeeper,
        app.StakingKeeper,
        authtypes.NewModuleAddress(govv1.ModuleName).String(),
    )

    
    app.ModuleManager = module.NewManager(
        builder.NewAppModule(appCodec, app.BuilderKeeper),
        ...
    )
    ```

    f. Configure the proposal/checkTx handlers on base app.

    ```go
    // Create the proposal handler that will be used to build and validate blocks.
    proposalHandler := abci.NewProposalHandler(
        app.Logger(),
        app.txConfig.TxDecoder(),
        mempool,
    )
    app.App.SetPrepareProposal(proposalHandler.PrepareProposalHandler())
    app.App.SetProcessProposal(proposalHandler.ProcessProposalHandler())

    // Set the custom CheckTx handler on BaseApp.
    checkTxHandler := mev.NewCheckTxHandler(
        app.App,
        app.txConfig.TxDecoder(),
        mevLane,
        anteHandler,
    )
    app.SetCheckTx(checkTxHandler.CheckTx())

    // CheckTx will check the transaction with the provided checkTxHandler. 
    // We override the default handler so that we can verify transactions 
    // before they are inserted into the mempool. With the CheckTx, we can 
    // verify the bid transaction and all of the bundled transactions
    // before inserting the bid transaction into the mempool.
    func (app *TestApp) CheckTx(req *cometabci.RequestCheckTx) 
        (*cometabci.ResponseCheckTx, error) {
        return app.checkTxHandler(req)
    }

    // SetCheckTx sets the checkTxHandler for the app.
    func (app *TestApp) SetCheckTx(handler mev.CheckTx) {
        app.checkTxHandler = handler
    }
    ```

    g. Finally, update the app's `InitGenesis` order.

    ```go
    genesisModuleOrder := []string{
        buildertypes.ModuleName,
        ...,
    }
    ```

## Params

Note, before building or upgrading the application, make sure to initialize the
escrow address in the parameters of the module. The default parameters
initialize the escrow address to be the module account address.
