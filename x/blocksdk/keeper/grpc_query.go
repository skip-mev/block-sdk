package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/skip-mev/block-sdk/v2/x/blocksdk/types"
)

var _ types.QueryServer = QueryServer{}

// QueryServer defines the blocksdk module's gRPC querier service.
type QueryServer struct {
	keeper Keeper
}

// NewQueryServer creates a new gRPC query server for the blocksdk module.
func NewQueryServer(keeper Keeper) *QueryServer {
	return &QueryServer{keeper: keeper}
}

// Lane implements the service to query a Lane by its ID.
func (q QueryServer) Lane(goCtx context.Context, query *types.QueryLaneRequest) (*types.QueryLaneResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	lane, err := q.keeper.GetLane(ctx, query.Id)
	if err != nil {
		return nil, err
	}

	return &types.QueryLaneResponse{Lane: lane}, nil
}

// Lanes implements the service to query all Lanes in the stores.
func (q QueryServer) Lanes(goCtx context.Context, _ *types.QueryLanesRequest) (*types.QueryLanesResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	lanes := q.keeper.GetLanes(ctx)

	return &types.QueryLanesResponse{Lanes: lanes}, nil
}
