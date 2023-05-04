
# POB Specification

## Abstract

The `x/builder` module is a Cosmos SDK module that allows Cosmos chains to host
top-of-block auctions directly in-protocol with auction revenue (MEV) being
redistributed according to the preferences of the chain. The `x/builder` module
introduces a new `MsgAuctionBid` message that allows users to submit a bid
alongside an ordered list of transactions, i.e. a **bundle**, that they want
executed at top-of-block before any other transactions are executed for that
block. The `x/builder` module works alongside the `AuctionMempool` such that:

* Auctions are held directly in the `AuctionMempool`, where a winner is determined
  when the proposer proposes a new block in `PrepareProposal`.
* `x/builder` provides the necessary validation of auction bids and subsequent
  state transitions to extract bids.

## Concepts

### Miner Extractable Value (MEV)

MEV refers to the potential profit that miners, or validators in a Proof-of-Stake
system, can make by strategically ordering, selecting, or even censoring
transactions in the blocks they produce. MEV can be classified into "good MEV"
and "bad MEV" based on the effects it has on the blockchain ecosystem and its
users. It's important to note that these classifications are subjective and may
vary depending on one's perspective.

**Good MEV** refers to the value that validators can extract while contributing
positively to the blockchain ecosystem. This typically includes activities that
enhance network efficiency, maintain fairness, and align incentives with the
intended use of the system. Examples of good MEV include:

* **Back-running**: Validators can place their own transactions immediately
  after a profitable transaction, capitalizing on the changes caused by the
  preceding transaction.
* **Arbitrage**: By exploiting price differences across decentralized exchanges
  or other DeFi platforms, validators help maintain more consistent price levels
  across the ecosystem, ultimately contributing to its stability.
* **Liquidations**: In DeFi platforms, when users' collateral falls below a
  specific threshold, validators can liquidate these positions, thereby maintaining
  the overall health of the platform and protecting its users from insolvency risks.

**Bad MEV** refers to the value that validators can extract through activities
that harm the blockchain ecosystem, lead to unfair advantages, or exploit users.
Examples of bad MEV include:

* **Front-running**: Validators can observe pending transactions in the mempool
  (the pool of unconfirmed transactions) and insert their own transactions ahead
  of them. This can be particularly profitable in decentralized finance (DeFi)
  applications, where a validator could front-run a large trade to take advantage
  of price movements.
* **Sandwich attacks**: Validators can surround a user's transaction with their
  own transactions, effectively manipulating the market price for their benefit.
* **Censorship**: Validators can selectively exclude certain transactions from
  blocks to benefit their own transactions or to extract higher fees from users.

MEV is a topic of concern in the blockchain community because it can lead to
unfair advantages for validators, reduced trust in the system, and a potential
concentration of power. Various approaches have been proposed to mitigate MEV,
such as proposer-builder separation (described below) and transparent and fair
transaction ordering mechanisms at the protocol-level (`POB`) to make MEV
extraction more incentive aligned with the users and blockchain ecosystem.

### Proposer Builder Separation (PBS)

Proposer-builder separation is a concept in the design of blockchain protocols,
specifically in the context of transaction ordering within a block. In traditional
blockchain systems, validators perform two main tasks: they create new blocks
(acting as proposers) and determine the ordering of transactions within those
blocks (acting as builders).


**Proposers**: They are responsible for creating and broadcasting new blocks,
just like in traditional blockchain systems. *However, they no longer determine
the ordering of transactions within those blocks*.

**Builders**: They have the exclusive role of determining the order of transactions
within a block - can be full or partial block. Builders submit their proposed
transaction orderings to an auction mechanism, which selects the winning template
based on predefined criteria, e.g. highest bid.

This dual role can lead to potential issues, such as front-running and other
manipulations that benefit the miners/builders themselves.

* *Increased complexity*: Introducing PBS adds an extra layer of complexity to
  the blockchain protocol. Designing, implementing, and maintaining an auction
  mechanism for transaction ordering requires additional resources and may
  introduce new vulnerabilities or points of failure in the system.
* *Centralization risks*: With PBS, there's a risk that a few dominant builders
  may emerge, leading to centralization of transaction ordering. This centralization
  could result in a lack of diversity in transaction ordering algorithms and an
  increased potential for collusion or manipulation by the dominant builders.
* *Incentive misalignments*: The bidding process may create perverse incentives
  for builders. For example, builders may be incentivized to include only high-fee
  transactions to maximize their profits, potentially leading to a neglect of
  lower-fee transactions. Additionally, builders may be incentivized to build
  blocks that include **bad-MEV** strategies because they are more profitable.

## Specification

### Mempool

As the lifeblood of blockchains, mempools serve as the intermediary space for
pending transactions, playing a vital role in transaction management, fee markets,
and network health. With ABCI++, mempools can be defined at the application layer
instead of the consensus layer (CometBFT). This means applications can define
their own mempools that have their own custom verification, block building, and
state transition logic. Adding on, these changes make it such that blocks are
built (`PrepareProposal`) and verified (`ProcessProposal`) directly in the
application layer.

The `x/builder` module implements an application-side mempool, `AuctionMempool`,
that implements the `sdk.Mempool` interface. The mempool is composed of two
primary indexes, a global index that contains all non-auction transactions and
an index that only contains auction transactions, i.e. transactions with a single
`MsgAuctionBid` message. Both indexes order transactions based on priority respecting
the sender's sequence number. The global index prioritizes transactions based on
`ctx.Priority()` and the auction index prioritizes transactions based on the
bid.

### Configuration

The `AuctionMempool` mempool implementation accepts a `AuctionFactory` 
interface that allows the mempool to be generic across many Cosmos SDK 
applications, such that it allows the ability for the application developer to 
define their business logic in terms of how to perform things such as the following:

* Getting tx signers
* Getting bundled tx signers
* Retrieving bid information


```go
// AuctionFactory defines the interface for processing auction transactions. 
// It is a wrapper around all of the functionality that each application chain
// must implement in order for auction processing to work.
type AuctionFactory interface {
  // WrapBundleTransaction defines a function that wraps a bundle transaction 
  // into a sdk.Tx. Since this is a potentially expensive operation, we allow
  // each application chain to define how they want to wrap the transaction
  // such that it is only called when necessary (i.e. when the transaction is
  // being considered in the proposal handlers).
  WrapBundleTransaction(tx []byte) (sdk.Tx, error)

  // GetAuctionBidInfo defines a function that returns the bid info from an 
  // auction transaction.
  GetAuctionBidInfo(tx sdk.Tx) (*AuctionBidInfo, error)
}
```

### PrepareProposal

After the proposer of the next block has been selected, the CometBFT client will
call `PrepareProposal` to build the next block. The block will be built in two
stages. First, it will host the auction and include the winning bidder's bundle
as the first set of transactions for the block, i.e. it will select the bid
transaction itself along with automatically including all the bundled transactions
in the specified order they appear in the bid's `transactions` field.

The auction currently supports only a single winner. Selecting the auction winner
involves a greedy search for a valid auction transaction starting from highest
paying bid, respecting user nonce, in the `AuctionMempool`. The `x/builder`'s
ante handler is responsible for verifying the auction transaction based on the
criteria described below (see **Ante Handler**).

Then, it will build the rest of the block by reaping and validating the transactions
in the global index. The second portion of block building iterates from highest
to lowest priority transactions in the global index and adds them to the proposal
if they are valid. If the proposer comes across a transaction that was already
included in the top of block, it will be ignored.

### ProcessProposal

After the proposer proposes a block of transactions for the next block, the
block will be verified by other nodes in the network in `ProcessProposal`. If
there is an auction transaction in the proposal, it must be the first transaction
in the proposal and all bundled transactions must follow the auction transaction
in the exact order we would expect them to be seen. If this fails, the proposal
is rejected. If this passes, the validator will then run `CheckTx` on all of the
transactions in the block in the order in which they were provided in the proposal.

### Ante Handler

When users want to bid for the rights for top-of-block execution they will submit
a normal `sdk.Tx` transaction with a single `MsgAuctionBid`. The ante handler is
responsible for verification of this transaction. The ante handler will verify that:

1. The auction transaction specifies a timeout height where the bid is no longer
   considered valid. Note, it is REQUIRED that all bid transactions include a
   height timeout.
2. The auction transaction includes less than `MaxBundleSize` transactions in
   its bundle.
3. The auction transaction includes only a SINGLE `MsgAuctionBid` message. We
   enforce that no other messages are included to prevent front-running.
4. Enforce that the user has sufficient funds to pay the bid they entered while
   covering all relevant auction fees.
5. Enforce that the transaction's min bid increment greater than the local highest
   bid in the mempool.
6. Enforce that the bundle of transactions the bidder provided does not front-run
   or sandwich (if enabled).

Note, the process of selecting auction winners occurs in a greedy manner. In
`PrepareProposal`, the `AuctionMempool` will iterate from largest to smallest
bidding transaction until it finds the first valid bid transaction.

### State

The `x/builder` module stores the following state objects:

```protobuf
message Params {
  option (amino.name) = "cosmos-sdk/x/builder/Params";

  // max_bundle_size is the maximum number of transactions that can be bundled
  // in a single bundle.
  uint32 max_bundle_size = 1;

  // escrow_account_address is the address of the account that will receive a
  // portion of the bid proceeds.
  string escrow_account_address = 2;

  // reserve_fee specifies the bid floor for the auction.
  cosmos.base.v1beta1.Coin reserve_fee = 3
      [ (gogoproto.nullable) = false, (amino.dont_omitempty) = true ];

  // min_buy_in_fee specifies the fee that the bidder must pay to enter the
  // auction.
  cosmos.base.v1beta1.Coin min_buy_in_fee = 4
      [ (gogoproto.nullable) = false, (amino.dont_omitempty) = true ];

  // min_bid_increment specifies the minimum amount that the next bid must be
  // greater than the previous bid.
  cosmos.base.v1beta1.Coin min_bid_increment = 5
      [ (gogoproto.nullable) = false, (amino.dont_omitempty) = true ];

  // front_running_protection specifies whether front running and sandwich
  // attack protection is enabled.
  bool front_running_protection = 6;

  // proposer_fee defines the portion of the winning bid that goes to the block
  // proposer that proposed the block.
  string proposer_fee = 7 [
    (gogoproto.nullable) = false,
    (gogoproto.customtype) = "github.com/cosmos/cosmos-sdk/types.Dec"
  ];
}
```

## Messages

### MsgAuctionBid

POB defines a new Cosmos SDK `Message`, `MsgAuctionBid`, that allows users to
create an auction bid and participate in a top-of-block auction. The `MsgAuctionBid`
message defines a bidder and a series of embedded transactions, i.e. the bundle.

```protobuf
message MsgAuctionBid {
  option (cosmos.msg.v1.signer) = "bidder";
  option (amino.name) = "pob/x/builder/MsgAuctionBid";

  option (gogoproto.equal) = false;

  // bidder is the address of the account that is submitting a bid to the
  // auction.
  string bidder = 1 [ (cosmos_proto.scalar) = "cosmos.AddressString" ];
  // bid is the amount of coins that the bidder is bidding to participate in the
  // auction.
  cosmos.base.v1beta1.Coin bid = 3
      [ (gogoproto.nullable) = false, (amino.dont_omitempty) = true ];
  // transactions are the bytes of the transactions that the bidder wants to
  // bundle together.
  repeated bytes transactions = 4;
}
```

Note, the `transactions` may or may not exist in a node's application mempool. If
a transaction containing a single `MsgAuctionBid` wins the auction, the block
proposal will automatically include the `MsgAuctionBid` transaction along with
injecting all the bundled transactions such that they are executed in the same
order after the `MsgAuctionBid` transaction.

When processing a `MsgAuctionBid`, the `x/builder` module will perform two primary
actions:

1. Ensure the bid is valid per the module's parameters and configuration.
2. Extract fee payments from the bidder's account and escrow them to the module's
   escrow account and the proposer that included the winning bid in the block
   proposal.

### MsgUpdateParams

The `MsgUpdateParams` message allows for an authority, typically the `x/gov`
module account, to update the `x/builder`'s parameters.

```protobuf
message MsgUpdateParams {
  option (cosmos.msg.v1.signer) = "authority";
  option (amino.name) = "pob/x/builder/MsgUpdateParams";

  option (gogoproto.equal) = false;

  // authority is the address of the account that is authorized to update the
  // x/builder module parameters.
  string authority = 1 [ (cosmos_proto.scalar) = "cosmos.AddressString" ];
  // params is the new parameters for the x/builder module.
  Params params = 2 [ (gogoproto.nullable) = false ];
}
```
