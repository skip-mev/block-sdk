package keeper

import (
	"context"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/skip-mev/block-sdk/x/blocksdk/types"
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

func (q QueryServer) Lane(goCtx context.Context, query *types.QueryLaneRequest) (*types.QueryLaneResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	lane, err := q.keeper.GetLane(ctx, query.Id)
	if err != nil {
		return nil, err
	}

	return &types.QueryLaneResponse{Lane: lane}, nil
}

func (q QueryServer) Lanes(goCtx context.Context, _ *types.QueryAllLanesRequest) (*types.QueryAllLanesResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	lanes := q.keeper.GetLanes(ctx)

	return &types.QueryAllLanesResponse{Lanes: lanes}, nil
}
