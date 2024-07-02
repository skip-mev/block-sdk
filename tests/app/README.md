# Block SDK Enabled Test App (10 min)

## Overview

This readme describes how to build a test app that uses the Block SDK. This assumes that you have already installed the Block SDK and have a working development environment. To install the Block SDK, please run the following:

```bash
go get github.com/skip-mev/block-sdk
```

## Building the Test App

There are fix critical steps to building a test app that uses the Block SDK:

1. Set up the signer extractor.
2. Create the lane configurations for each individual lane i.e. `LaneConfig`.
3. Configure the match handlers for each lane i.e. `MatchHandler`.
4. Creating the Block SDK mempool i.e. `LanedMempool`.
5. Setting the antehandlers - used for transaction validation - for each lane.
6. Setting the proposal handlers - used for block creation and verification - for the application to utilize the Block SDK's Prepare and Process Proposal handlers.

**IMPORTANT NOTE:** It is recommended that applications use the **`NewDefaultProposalHandler`** when constructing their Prepare and Process Proposal handlers. This is because the priority nonce mempool - the underlying storage device of transactions - has non-deterministic ordering of transactions on chains with multiple fees. The use of the `NewDefaultProposalHandler` ensures that the ordering of transactions on proposal construction follows the ordering logic of the lanes, which is deterministic, and that process proposal optimistically assumes that the ordering of transactions is correct.

### 1. Signer Extractor

The signer extractor is responsible for extracting signers and relevant information about who is signing the transaction. We recommend using the default implementation provided by the Block SDK. 

```go
signerAdapter := signerextraction.NewDefaultAdapter()
```

### 2. Lane Configurations

This controls how many transactions can be stored by each lane, how much block space is allocated to each lane, how to extract transacation information such as signers, fees, and more. Each lane should have a separate `LaneConfig` object.

For example, in [`lanes.go`](./lanes.go) we see the following:

```go
mevConfig := base.LaneConfig{
	Logger:          app.Logger(),
	TxEncoder:       app.txConfig.TxEncoder(),
	TxDecoder:       app.txConfig.TxDecoder(),
	MaxBlockSpace:   math.LegacyMustNewDecFromStr("0.2"),
	SignerExtractor: signerAdapter,
	MaxTxs:          1000,
}
```

Following the example above:

* `Logger`: This is the logger that will be utilized by the lane when outputting information as blocks are being processed and constructed. 
* `TxEncoder`: This is the encoder that will be used to encode transactions.
* `TxDecoder`: This is the decoder that will be used to decode transactions.
* `MaxBlockSpace`: This is the maximum amount of block space that can be allocated to this lane. In this case, we allocate 20% of the block space to this lane.
* `SignerExtractor`: This is the signer extractor that will be used to extract signers from transactions. In this case, we utilize the default signer extractor provided by the Block SDK. **This is the recommended approach.**
* `MaxTxs`: This is the maximum number of transactions that can be stored in this lane. In this case, we allow up to 1000 transactions to be stored in this lane at any given time.

### Match Handlers

Match handlers are responsible for matching transactions to lanes. Each lane should have a unique match handler. By default, we recommend that the default lane be the last lane in your application. This is because the default lane matches all transactions that do not match to any of the other lanes. If you want to have a lane after the default lane, please see the section below.

#### (OPTIONAL) Having Lanes after the Default Lane

If you want to have lanes after the default lane, you will need to utilize the `base.NewMatchHandler` function. This function allows you to construct a match handler that can ignore other lane's match handlers.

For example, if we wanted the free and MEV lanes to be processed after the default lane - default, MEV, free - we can do the following:

```go
// Create the final match handler for the default lane.
defaultMatchHandler := base.NewMatchHandler(
	base.DefaultMatchHandler(),
	factory.MatchHandler(),
	freelane.DefaultMatchHandler(),
)
```

Following the example, we can see the following:

* `base.DefaultMatchHandler()`: This is the default match handler provided by the Block SDK. This matches all transactions to the lane.
* `factory.MatchHandler()`: This is the MEV lane's match handler. This is passed as a parameter to the `base.NewMatchHandler` function - which means that all transactions that match to the MEV lane will be ignored by the default match handler.
* `freelane.DefaultMatchHandler()`: This is the default match handler for the free lane. This is passed as a parameter to the `base.NewMatchHandler` function - which means that all transactions that match to the free lane will be ignored by the default match handler.

**This will allow the default match handler to only match transactions that do not match to the MEV lane or the free lane.**

### Block SDK Mempool

After constructing the lanes, we can create the Block SDK mempool - `LanedMempool`. This object is responsible for managing the lanes and processing transactions. 

```go
// STEP 1: Create the Block SDK lanes.
mevLane, freeLane, defaultLane := CreateLanes(app)

// STEP 2: Construct a mempool based off the lanes.
mempool, err := block.NewLanedMempool(
	app.Logger(),
	[]block.Lane{mevLane, freeLane, defaultLane},
)
if err != nil {
	panic(err)
}

// STEP 3: Set the mempool on the app.
app.App.SetMempool(mempool)
```

Note that we pass the lanes to the `block.NewLanedMempool` function. **The order of the lanes is important.** Proposals will be constructed based on the order of lanes passed to the `block.NewLanedMempool` function. In the example above, the MEV lane will be processed first, followed by the free lane, and finally the default lane.

### AnteHandlers

`AnteHandlers` are responsible for validating transactions. We recommend that developers utilize the same antehandler chain that is used by the application. In the example test app, we construct the `AnteHandler` with `NewBSDKAnteHandler`. In the case where the certain ante decorators should ignore certain lanes, we can wrap a `Decorator` with the `block.NewIgnoreDecorator` function as seen in `ante.go`.

After constructing the `AnteHandler`, we can set it on the application and on the lanes.

```go
// STEP 4: Create a global ante handler that will be called on each transaction when
// proposals are being built and verified. Note that this step must be done before
// setting the ante handler on the lanes.
handlerOptions := ante.HandlerOptions{
	AccountKeeper:   app.AccountKeeper,
	BankKeeper:      app.BankKeeper,
	FeegrantKeeper:  app.FeeGrantKeeper,
	SigGasConsumer:  ante.DefaultSigVerificationGasConsumer,
	SignModeHandler: app.txConfig.SignModeHandler(),
}
options := BSDKHandlerOptions{
	BaseOptions:   handlerOptions,
	auctionkeeper: app.auctionkeeper,
	TxDecoder:     app.txConfig.TxDecoder(),
	TxEncoder:     app.txConfig.TxEncoder(),
	FreeLane:      freeLane,
	MEVLane:       mevLane,
}
anteHandler := NewBSDKAnteHandler(options)
app.App.SetAnteHandler(anteHandler)

// Set the AnteHandlers on the lanes.
mevLane.SetAnteHandler(anteHandler)
freeLane.SetAnteHandler(anteHandler)
defaultLane.SetAnteHandler(anteHandler)
```

### Proposal Handlers

The proposal handlers - `PrepareProposal` and `ProcessProposal` - are responsible for building and verifying block proposals. To add it to your application, follow the example below:

```go
// Step 5: Create the proposal handler and set it on the app.
proposalHandler := abci.NewProposalHandler(
	app.Logger(),
	app.TxConfig().TxDecoder(),
	app.TxConfig().TxEncoder(),
	mempool,
)
app.App.SetPrepareProposal(proposalHandler.PrepareProposalHandler())
app.App.SetProcessProposal(proposalHandler.ProcessProposalHandler())
```

## Conclusion

Adding the Block SDK to your application is a simple 6 step process. If you have any questions, please feel free to reach out to the [Skip team](https://skip.money/contact). We are happy to help!
