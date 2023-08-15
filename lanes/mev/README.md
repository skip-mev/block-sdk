# MEV Lane

> The MEV Lane hosts top of block auctions in protocol and verifiably builds 
> blocks with top-of-block block space reserved for auction winners, with 
> auction revenue being redistributed to chains.

## 📖 Overview

Blockspace is valuable, and MEV bots find arbitrage opportunities to capture 
value. The Block SDK provides a fair auction for these opportunities via the 
x/auction module inside the Block SDK so that protocols are rewarded while 
ensuring that users are not front-run or sandwiched in the process. 

The Block SDK uses the app-side mempool, PrepareLane / ProcessLane, and CheckTx 
to create an MEV marketplace inside the protocol. It introduces a new message 
type, called a MsgAuctionBid, that allows the submitter to execute multiple 
transactions at the top of the block atomically 
(atomically = directly next to each other).

## Install

```shell
$ go install github.com/skip-mev/block-sdk
```

## Setup

> This set up guide will walk you through the process of setting up a POB 
> application. In particular, we will configure an application with the 
> following features:
>
>* MEV lane (auction lane). This will create an MEV lane where users can bid to 
> have their transactions executed at the top of the block.
>* Free lane. This will create a free lane where users can submit transactions 
> that will be executed for free (no fees).
>* Default lane. This will create a default lane where users can submit 
> transactions that will be executed with the default app logic.
>* Builder module that pairs with the auction lane to process auction 
> transactions and distribute revenue to the auction house.

1. Import the necessary dependencies into your application. This includes the
   Block SDK proposal handlers + mempool, keeper, builder types, and builder 
   module. This tutorial will go into more detail into each of the dependencies.

   ```go
   import (
    ...
    "github.com/skip-mev/pob/block-sdk"
    "github.com/skip-mev/pob/block-sdk/abci"
    "github.com/skip-mev/pob/block-sdk/lanes/auction"
    "github.com/skip-mev/pob/block-sdk/lanes/base"
    "github.com/skip-mev/pob/block-sdk/lanes/free"
    buildermodule "github.com/skip-mev/block-sdk/x/builder"
    builderkeeper "github.com/skip-mev/block-sdk/x/builder/keeper"
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

3. The builder `Keeper` is POB's gateway to processing special `MsgAuctionBid`
   messages that allow users to participate in the top of block auction, distribute
   revenue to the auction house, and ensure the validity of auction transactions.

   a. First add the keeper to the app's struct definition. We also want to add 
   MEV lane's custom checkTx handler to the app's struct definition. This will 
   allow us to override the default checkTx handler to process bid transactions 
   before they are inserted into the mempool. NOTE: The custom handler is 
   required as otherwise the auction can be held hostage by a malicious
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

    c. Instantiate the blockbuster mempool with the application's desired lanes.

      ```go
        // 1. Create the lanes.
        //
        // NOTE: The lanes are ordered by priority. The first lane is the 
        // highest priority lane and the last lane is the lowest priority lane. 
        // Top of block lane allows transactions to bid for inclusion at the 
        // top of the next block.
        //
        // For more information on how to utilize the LaneConfig please
        // visit the README in block-sdk/block/base.
        //
        // MEV lane hosts an auction at the top of the block.
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

        // Standard lane accepts all other transactions.
        defaultConfig := constructor.LaneConfig{
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
            freeLane,
            defaultLane,
        }
        mempool := block.NewLanedMempool(app.Logger(), true, lanes...)
        app.App.SetMempool(mempool)
      ```

    d. Instantiate the antehandler chain for the application with awareness of the
    blockbuster mempool. This will allow the application to verify the validity
    of a transaction respecting the desired logic of a given lane. In this walkthrough,
    we want the `FeeDecorator` to be ignored for all transactions that should 
    belong to the free lane. Additionally, we want to add the `x/builder` 
    module's `AuctionDecorator` to the ante-handler chain. The `AuctionDecorator`
    is an AnteHandler decorator that enforces various chain configurable MEV rules.

      ```go
        import (
            ...
            "github.com/skip-mev/pob/blockbuster"
            "github.com/skip-mev/pob/blockbuster/utils"
            builderante "github.com/skip-mev/pob/x/builder/ante"
            ...
        )

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

    e. Configure the proposal/checkTx handlers on base app.

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
    checkTxHandler := abci.NewCheckTxHandler(
      app.App,
      app.txConfig.TxDecoder(),
      tobLane,
      anteHandler,
      app.ChainID(),
    )
    app.SetCheckTx(checkTxHandler.CheckTx())
    ...


    func (app *TestApp) CheckTx(req cometabci.RequestCheckTx) 
        cometabci.ResponseCheckTx {
      return app.checkTxHandler(req)
    }

    // SetCheckTx sets the checkTxHandler for the app.
    func (app *TestApp) SetCheckTx(handler abci.CheckTx) {
      app.checkTxHandler = handler
    }
    ```

    f. Finally, update the app's `InitGenesis` order and ante-handler chain.

    ```go
    genesisModuleOrder := []string{
      buildertypes.ModuleName,
      ...,
    }
    ```

## Params

Note, before building or upgrading the application, make sure to initialize the
escrow address for POB in the parameters of the module. The default parameters
initialize the escrow address to be the module account address.
