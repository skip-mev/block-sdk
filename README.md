<h1 align="center">Protocol-Owned Builder</h1>

<!-- markdownlint-disable MD013 -->
<!-- markdownlint-disable MD041 -->
[![Project Status: Active – The project has reached a stable, usable state and is being actively developed.](https://www.repostatus.org/badges/latest/active.svg)](https://www.repostatus.org/#wip)
[![GoDoc](https://img.shields.io/badge/godoc-reference-blue?style=flat-square&logo=go)](https://godoc.org/github.com/skip-mev/pob)
[![Go Report Card](https://goreportcard.com/badge/github.com/skip-mev/pob?style=flat-square)](https://goreportcard.com/report/github.com/skip-mev/pob)
[![Version](https://img.shields.io/github/tag/skip-mev/pob.svg?style=flat-square)](https://github.com/skip-mev/pob/releases/latest)
[![License: Apache-2.0](https://img.shields.io/github/license/skip-mev/pob.svg?style=flat-square)](https://github.com/skip-mev/pob/blob/main/LICENSE)
[![Lines Of Code](https://img.shields.io/tokei/lines/github/skip-mev/pob?style=flat-square)](https://github.com/skip-mev/pob)

Skip Protocol's Protocol-Owned Builder (POB) is a set of Cosmos SDK and ABCI++
primitives that provide application developers the ability to define how their
apps construct and validate blocks on-chain in a transparent, enforceable way,
such as giving complete control to the protocol to recapture, control, and
redistribute MEV.

Skip's POB provides developers with a set of a few core primitives:

* `BlockBuster`: BlockBuster is a generalized block-building and mempool SDK
  that allows developers to define how their applications construct and validate blocks
  on-chain in a transparent, enforceable way. At its core, BlockBuster is an app-side mempool + set 
  of proposal handlers (`PrepareProposal`/`ProcessProposal`) that allow developers to configure 
  modular lanes of transactions in their blocks with distinct validation/ordering logic. For more
  information, see the [BlockBuster README](/blockbuster/README.md).
* `x/builder`: This Cosmos SDK module gives applications the ability to process
  MEV bundled transactions in addition to having the ability to define how searchers
  and block proposers are rewarded. In addition, the module defines a `AuctionDecorator`,
  which is an AnteHandler decorator that enforces various chain configurable MEV
  rules.

## Releases

### Release Compatibility Matrix

| POB Version | Cosmos SDK |
| :---------: | :--------: |
|   v1.x.x    |  v0.47.x   |
|   v1.x.x    |  v0.48.x   |
|   v1.x.x    |  v0.49.x   |
|   v1.x.x    |  v0.50.x   |

## Install

```shell
$ go install github.com/skip-mev/pob
```

## Setup

>This set up guide will walk you through the process of setting up a POB application. In particular, we will configure an application with the following features:
>
>* Top of block lane (auction lane). This will create an auction lane where users can bid to have their
>    transactions executed at the top of the block.
>* Free lane. This will create a free lane where users can submit transactions that will be executed
>     for free (no fees).
>* Default lane. This will create a default lane where users can submit transactions that will be executed
>     with the default app logic.
>* Builder module that pairs with the auction lane to process auction transactions and distribute revenue
>     to the auction house.
>
> To build your own custom BlockBuster Lane, please see the [BlockBuster README](/blockbuster/README.md).

1. Import the necessary dependencies into your application. This includes the
   blockbuster proposal handlers +mempool, keeper, builder types, and builder module. This
   tutorial will go into more detail into each of the dependencies.

   ```go
   import (
    ...
    "github.com/skip-mev/pob/blockbuster"
    "github.com/skip-mev/pob/blockbuster/abci"
    "github.com/skip-mev/pob/blockbuster/lanes/auction"
    "github.com/skip-mev/pob/blockbuster/lanes/base"
    "github.com/skip-mev/pob/blockbuster/lanes/free"
    buildermodule "github.com/skip-mev/pob/x/builder"
    builderkeeper "github.com/skip-mev/pob/x/builder/keeper"
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

   a. First add the keeper to the app's struct definition. We also want to add POB's custom
   checkTx handler to the app's struct definition. This will allow us to override the 
   default checkTx handler to process bid transactions before they are inserted into the mempool.
   NOTE: The custom handler is required as otherwise the auction can be held hostage by a malicious
   users.

      ```go
      type App struct {
        ...
        // BuilderKeeper is the keeper that handles processing auction transactions
        BuilderKeeper         builderkeeper.Keeper

        // Custom checkTx handler
        checkTxHandler abci.CheckTx
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
        // Set the blockbuster mempool into the app.
        // Create the lanes.
        //
        // NOTE: The lanes are ordered by priority. The first lane is the highest priority
        // lane and the last lane is the lowest priority lane.
        // Top of block lane allows transactions to bid for inclusion at the top of the next block.
        //
        // blockbuster.BaseLaneConfig is utilized for basic encoding/decoding of transactions. 
        tobConfig := blockbuster.BaseLaneConfig{
          Logger:        app.Logger(),
          TxEncoder:     app.txConfig.TxEncoder(),
          TxDecoder:     app.txConfig.TxDecoder(),
          // the desired portion of total block space to be reserved for the lane. a value of 0
          // indicates that the lane can use all available block space.
          MaxBlockSpace: sdk.ZeroDec(),
        }
        tobLane := auction.NewTOBLane(
          tobConfig,
          // the maximum number of transactions that the mempool can store. a value of 0 indicates
          // that the mempool can store an unlimited number of transactions.
          0,
          // AuctionFactory is responsible for determining what is an auction bid transaction and
          // how to extract the bid information from the transaction. There is a default implementation
          // that can be used or application developers can implement their own.
          auction.NewDefaultAuctionFactory(app.txConfig.TxDecoder()),
        )

        // Free lane allows transactions to be included in the next block for free.
        freeConfig := blockbuster.BaseLaneConfig{
          Logger:        app.Logger(),
          TxEncoder:     app.txConfig.TxEncoder(),
          TxDecoder:     app.txConfig.TxDecoder(),
          MaxBlockSpace: sdk.ZeroDec(),
          // IgnoreList is a list of lanes that if a transaction should be included in, it will be
          // ignored by the lane. For example, if a transaction should belong to the tob lane, it
          // will be ignored by the free lane.
          IgnoreList: []blockbuster.Lane{
            tobLane,
          },
        }
        freeLane := free.NewFreeLane(
          freeConfig,
          free.NewDefaultFreeFactory(app.txConfig.TxDecoder()),
        )

        // Default lane accepts all other transactions.
        defaultConfig := blockbuster.BaseLaneConfig{
          Logger:        app.Logger(),
          TxEncoder:     app.txConfig.TxEncoder(),
          TxDecoder:     app.txConfig.TxDecoder(),
          MaxBlockSpace: sdk.ZeroDec(),
          IgnoreList: []blockbuster.Lane{
            tobLane,
            freeLane,
          },
        }
        defaultLane := base.NewDefaultLane(defaultConfig)

        // Set the lanes into the mempool.
        lanes := []blockbuster.Lane{
          tobLane,
          freeLane,
          defaultLane,
        }
        mempool := blockbuster.NewMempool(lanes...)
        app.App.SetMempool(mempool)
      ```

    d. Instantiate the antehandler chain for the application with awareness of the
    blockbuster mempool. This will allow the application to verify the validity
    of a transaction respecting the desired logic of a given lane. In this walkthrough,
    we want the `FeeDecorator` to be ignored for all transactions that should belong to the 
    free lane. Additionally, we want to add the `x/builder` module's `AuctionDecorator` to the
    ante-handler chain. The `AuctionDecorator` is an AnteHandler decorator that enforces various
    chain configurable MEV rules.

      ```go
        import (
            ...
            "github.com/skip-mev/pob/blockbuster"
            "github.com/skip-mev/pob/blockbuster/utils"
            builderante "github.com/skip-mev/pob/x/builder/ante"
            ...
        )

        anteDecorators := []sdk.AnteDecorator{
          ante.NewSetUpContextDecorator(), // outermost AnteDecorator. SetUpContext must be called first
          ...
          // The IgnoreDecorator allows for certain decorators to be ignored for certain transactions. In 
          // this case, we want to ignore the FeeDecorator for all transactions that should belong to the
          // free lane.
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
          builderante.NewBuilderDecorator(options.BuilderKeeper, options.TxEncoder, options.TOBLane, options.Mempool),
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

    e. With Cosmos SDK version 0.47.0, the process of building blocks has been
    updated and moved from the consensus layer, CometBFT, to the application layer.
    When a new block is requested, the proposer for that height will utilize the
    `PrepareProposal` handler to build a block while the `ProcessProposal` handler
    will verify the contents of the block proposal by all validators. The
    combination of the `BlockBuster` mempool + `PrepareProposal`/`ProcessProposal`
    handlers allows the application to verifiably build valid blocks with
    top-of-block block space reserved for auctions and partial block for free transactions. 
    Additionally, we override the `BaseApp`'s `CheckTx` handler with our own custom 
    `CheckTx` handler that will be responsible for checking the validity of transactions. 
    We override the `CheckTx` handler so that we can verify auction transactions before they are
    inserted into the mempool. With the POB `CheckTx`, we can verify the auction
    transaction and all of the bundled transactions before inserting the auction
    transaction into the mempool. This is important because we otherwise there may be
    discrepencies between the auction transaction and the bundled transactions
    are validated in `CheckTx` and `PrepareProposal` such that the auction can be 
    griefed. All other transactions will be executed with base app's `CheckTx`.

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

    // CheckTx will check the transaction with the provided checkTxHandler. We override the default
    // handler so that we can verify bid transactions before they are inserted into the mempool.
    // With the POB CheckTx, we can verify the bid transaction and all of the bundled transactions
    // before inserting the bid transaction into the mempool.
    func (app *TestApp) CheckTx(req cometabci.RequestCheckTx) cometabci.ResponseCheckTx {
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
initialize the escrow address to be the module account address. The escrow address 
will be the address that is receiving a portion of auction house revenue alongside the proposer (or custom rewards providers).
