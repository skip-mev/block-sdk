package keeper

import (
	"context"
	"github.com/skip-mev/block-sdk/x/blocksdk/types"
)

var _ types.QueryServer = QueryServer{}

// QueryServer defines the auction module's gRPC querier service.
type QueryServer struct {
	keeper Keeper
}

// NewQueryServer creates a new gRPC query server for the auction module.
func NewQueryServer(keeper Keeper) *QueryServer {
	return &QueryServer{keeper: keeper}
}

func (q QueryServer) Lane(c context.Context, _ *types.QueryLaneRequest) (*types.QueryLaneResponse, error) {
	// TODO

	return nil, nil
}

func (q QueryServer) Lanes(c context.Context, _ *types.QueryAllLanesRequest) (*types.QueryAllLanesResponse, error) {
	// TODO

	return nil, nil
}
