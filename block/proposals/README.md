# Proposals

## Overview

> The proposal type - `proposals.Proposal` - is utilized to represent a block proposal. It contains information about the total gas utilization, block size, number of transactions, raw transactions, and much more. It is recommended that you read the [proposal construction and verification](../../abci/README.md) section before continuing.

## Proposal

After a given lane executes its `PrepareLaneHandler` or `ProcessLaneHandler`, it will return a set of transactions that need to be added to the current proposal that is being constructed. To update the proposal, `Update` is called with the lane that needs to add transactions to the proposal as well as the transactions that need to be added.

Proposals are updated _iff_:

1. The total gas utilization of the partial proposal (i.e. the transactions it wants to add) are under the limits allocated for the lane and are less than the maximum gas utilization of the proposal.
2. The total size in bytes of the partial proposal is under the limits allocated for the lane and is less than the maximum size of the proposal.
3. The transactions have not already been added to the proposal.
4. The lane has not already attempted to add transactions to the proposal.

If any of these conditions fail, the proposal will not be updated and the transactions will not be added to the proposal. The lane will be marked as having attempted to add transactions to the proposal.

The proposal is responsible for determining the `LaneLimits` for a given lane. The `LaneLimits` are the maximum gas utilization and size in bytes that a given lane can utilize in a block proposal. This is a function of the max gas utilization and size defined by the application, the current gas utilization and size of the proposal, and the `MaxBlockSpace` allocated to the lane as defined by its `LaneConfig`. To read more about how `LaneConfigs` are defined, please visit the [lane config section](../base/README.md#laneconfig) or see an example implementation in [`app.go`](../../tests/app/app.go).

