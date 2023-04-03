package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/skip-mev/pob/x/builder/types"
)

var _ types.QueryServer = QueryServer{}

// QueryServer defines the builder module's gRPC querier service.
type QueryServer struct {
	keeper Keeper
}

// NewQueryServer creates a new gRPC query server for the builder module.
func NewQueryServer(keeper Keeper) *QueryServer {
	return &QueryServer{keeper: keeper}
}

// Params queries all parameters of the builder module.
func (q QueryServer) Params(c context.Context, _ *types.QueryParamsRequest) (*types.QueryParamsResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)

	params, err := q.keeper.GetParams(ctx)
	if err != nil {
		return nil, err
	}

	return &types.QueryParamsResponse{Params: params}, nil
}
