# Signer Extraction Adapter (SEA)

## Overview

The Signer Extraction Adapter is utilized to retrieve the signature information of a given transaction. This is purposefully built to allow application developers to retrieve signer information in the case where the default Cosmos SDK signature information is not applicable. 

## Utilization within the Block SDK

Each lane can configure it's own Signer Extraction Adapter (SEA). However, for almost all cases each lane will have the same SEA. The SEA is utilized to retrieve the address of the signer and nonce of the transaction. It's utilized by each lane's mempool to retrieve signer information as transactions are being inserted and for logging purposes as a proposal is being created / verified.

## Configuration

To extend and implement a new SEA, the following interface must be implemented:

```go
// Adapter is an interface used to determine how the signers of a transaction should be extracted
// from the transaction.
type Adapter interface {
	GetSigners(sdk.Tx) ([]SignerData, error)
}
```

The `GetSigners` method is responsible for extracting the signer information from the transaction. The `SignerData` struct is defined as follows:

```go
type SignerData struct {
	Signer   sdk.AccAddress
	Sequence uint64
}
```

