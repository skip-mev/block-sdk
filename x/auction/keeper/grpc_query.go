package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/skip-mev/block-sdk/v2/x/auction/types"
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

// Params queries all parameters of the auction module.
func (q QueryServer) Params(c context.Context, _ *types.QueryParamsRequest) (*types.QueryParamsResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)

	params, err := q.keeper.GetParams(ctx)
	if err != nil {
		return nil, err
	}

	escrowAddress := sdk.AccAddress(params.EscrowAccountAddress)
	return &types.QueryParamsResponse{Params: params, EscrowAddressString: escrowAddress.String()}, nil
}
