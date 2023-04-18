<!-- markdownlint-disable MD013 -->
<h1 align="center">Protocol-Owned Builder</h1>

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

* `x/builder`: This Cosmos SDK module gives applications the ability to process
  MEV bundled transactions in addition to having the ability to define how searchers
  and block proposers are rewarded. In addition, the module defines a `AuctionDecorator`,
  which is an AnteHandler decorator that enforces various chain configurable MEV
  rules.
* `ProposalHandler`: This ABCI++ handler defines `PrepareProposal` and `ProcessProposal`
  methods that give applications the ability to perform top-of-block auctions,
  which enables recapturing, redistributing and control over MEV. These methods
  are responsible for block proposal construction and validation.
* `AuctionMempool`: An MEV-aware mempool that enables searchers to submit bundled
  transactions to the mempool and have them bundled into blocks via a top-of-block
  auction. Searchers include a bid in their bundled transactions and the highest
  bid wins the auction. Application devs have control over levers that control
  aspects such as the bid floor and minimum bid increment.

## Releases

### Release Compatibility Matrix

| POB Version | Cosmos SDK |
| :---------: | :--------: |
|   v1.x.x    |  v0.47.x   |

## Install

```shell
$ go install github.com/skip-mev/pob
```

## Setup

1. Import the necessary dependencies into your application. This includes the
   proposal handlers, mempool, keeper, builder types, and builder module. This
   tutorial will go into more detail into each of the dependencies.

   ```go
   import (
     proposalhandler "github.com/skip-mev/pob/abci"
     "github.com/skip-mev/pob/mempool"
     "github.com/skip-mev/pob/x/builder"
     builderkeeper "github.com/skip-mev/pob/x/builder/keeper"
     buildertypes "github.com/skip-mev/pob/x/builder/types"
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
       builder.AppModuleBasic{},
     )
     ...
   )
   ```

3. The builder `Keeper` is POB's gateway to processing special `MsgAuctionBid`
   messages that allow users to participate in the top of block auction, distribute
   revenue to the auction house, and ensure the validity of auction transactions.

   a. First add the keeper to the app's struct definition.

      ```go
      type App struct {
        BuilderKeeper builderkeeper.Keeper
        ...
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

    c. Instantiate the builder keeper, store keys, and module manager. Note, be
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

    d. Searchers bid to have their bundles executed at the top of the block
    using `MsgAuctionBid` messages. While the builder `Keeper` is capable of
    tracking valid bids, it is unable to correctly sequence the auction
    transactions alongside the normal transactions without having access to the
    application’s mempool. As such, we have to instantiate POB’s custom
    `AuctionMempool` - a modified version of the SDK’s priority sender-nonce
    mempool - into the application. Note, this should be done after `BaseApp` is
    instantiated.

    ```go
    mempool := mempool.NewAuctionMempool(txConfig.TxDecoder(), GetMaxMempoolSize())
    bApp.SetMempool(mempool)
    ```

    e. With Cosmos SDK version 0.47.0, the process of building blocks has been
    updated and moved from the consensus layer, CometBFT, to the application layer.
    When a new block is requested, the proposer for that height will utilize the
    `PrepareProposal` handler to build a block while the `ProcessProposal` handler
    will verify the contents of the block proposal by all validators. The
    combination of the `AuctionMempool`, `PrepareProposal` and `ProcessProposal`
    handlers allows the application to verifiably build valid blocks with
    top-of-block block space reserved for auctions.

    ```go
    handler := proposalhandler.NewProposalHandler(
      mempool, 
      bApp.Logger(), 
      bApp,
      txConfig.TxEncoder(),
      txConfig.TxDecoder(),
    )
    bApp.SetPrepareProposal(handler.PrepareProposalHandler())
    bApp.SetProcessProposal(handler.ProcessProposalHandler())
    ```

    f. Finally, update the app's `InitGenesis` order and ante-handler chain.

    ```go
    genesisModuleOrder := []string{
      buildertypes.ModuleName,
      ...,
    }

    anteDecorators := []sdk.AnteDecorator{
      auction.NewAuctionDecorator(
        app.BuilderKeeper,
        txConfig.TxDecoder(),
        txConfig.TxEncoder(),
        mempool,
      ),
      ...,
    }
    ```

Note, before building or upgrading the application, make sure to initialize the
escrow address for POB in the parameters of the module. The default parameters
do not initialize an escrow address as that should be determined by governance.
The escrow address will be the address that is receiving a portion of auction
house revenue alongside the proposer (if enabled).
