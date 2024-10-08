# Overview

### 🤔What is the Block SDK?

**the Block SDK is a toolkit for building customized blocks**

The Block SDK is a set of Cosmos SDK and ABCI++ primitives that allow chains to fully customize blocks to specific use cases. It turns your chain's blocks into a **`highway`** consisting of individual **`lanes`** with their own special functionality.


Skip has built out a number of plug-and-play `lanes` on the SDK that your protocol can use, including in-protocol MEV recapture and Oracles! Additionally, the Block SDK can be extended to add **your own custom `lanes`** to configure your blocks to exactly fit your application needs.

🚦 **Blocks are like highways**

Let's say you're the designer of a 4 lane highway. You'd want a paid lane, for fast drivers who'd like to be separated from other lanes. You'd like a lane for large vehicles, you can configure this lane to be wider, require more space between vehicles, etc. The other two lanes are for the rest of traffic. The beauty here, is that as the owner of the highway, you get to decide what vehicles (transactions) you'll allow, and how they can behave (ordering)!!


#### If you've been here before

##### [Integrate Block-SDK](0-integrate-the-sdk.md)

##### [Building your own Lane](lanes/1-build-your-own-lane.md)

##### [Searcher docs for MEV Lane](3-searcher-docs.md)

### ❌ Problems: Blocks are not Customizable

Most Cosmos chains today utilize traditional block construction - which is too limited.

- Traditional block building is susceptible to MEV-related issues, such as front-running and sandwich attacks, since proposers have monopolistic rights on ordering and no verification of good behavior. MEV that is created cannot be redistributed to the protocol.
- Traditional block building uses a one-size-fits-all approach, which can result in inefficient transaction processing for specific applications or use cases and sub-optimal fee markets.
- Transactions tailored for specific applications may need custom prioritization, ordering or validation rules that the mempool is otherwise unaware of because transactions within a block are currently in-differentiable when a blockchain might want them to be.

### ✅ Solution: The Block SDK

You can think of the Block SDK as a **transaction `highway` system**, where each
`lane` on the highway serves a specific purpose and has its own set of rules and
traffic flow.

In the Block SDK, each `lane` has its own set of rules and transaction flow management systems.

- A `lane` is what we might traditionally consider to be a standard mempool
  where transaction **_validation_**, **_ordering_** and **_prioritization_** for
  contained transactions are shared.
- `lanes` implement a **standard interface** that allows each individual `lane` to
  propose and validate a portion of a block.
- `lanes` are ordered with each other, configurable by developers. All `lanes`
  together define the desired block structure of a chain.

### ✨ Block SDK Use Cases

A block with separate `lanes` can be used for:

1. **MEV mitigation**: a top of block lane could be designed to create an in-protocol top-of-block [auction](lanes/existing-lanes/1-mev.md) to recapture MEV in a transparent and governable way.
2. **Free/reduced fee txs**: transactions with certain properties (e.g. from trusted accounts or performing encouraged actions) could leverage a free lane to facilitate _good_ behavior.
3. **Dedicated oracle space** Oracles could be included before other kinds of transactions to ensure that price updates occur first, and are not able to be sandwiched or manipulated.
4. **Orderflow auctions**: an OFA lane could be constructed such that order flow providers can have their submitted transactions bundled with specific backrunners, to guarantee MEV rewards are attributed back to users
5. **Enhanced and customizable privacy**: privacy-enhancing features could be introduced, such as threshold encrypted lanes, to protect user data and maintain privacy for specific use cases.
6. **Fee market improvements**: one or many fee markets - such as EIP-1559 - could be easily adopted for different lanes (potentially custom for certain dApps). Each smart contract/exchange could have its own fee market or auction for transaction ordering.
7. **Congestion management**: segmentation of transactions to lanes can help mitigate network congestion by capping usage of certain applications and tailoring fee markets.

### 🎆 Chains Currently Using the Block-SDK

#### Mainnets

| Chain Name  | Chain-ID        | Block-SDK Version |
| ----------- | --------------- | ----------------- |
| Juno        | `juno-1`        | `v1.0.2`          |
| Persistence | `persistence-1` | `v1.0.2`          |
| Initia      | `NA`            | `v1.0.2`          |
| Prism       | `NA`            | `v1.0.2`          |
| Terra       | `phoenix-1`     | `v1.0.2`          |

#### Testnets

| Chain Name | Chain-ID | Block-SDK Version |
| ---------- | -------- | ----------------- |
| Juno       | `uni-6`  | `v1.0.2`          |
