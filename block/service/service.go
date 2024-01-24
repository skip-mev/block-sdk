package service

import (
	"context"

	gogogrpc "github.com/cosmos/gogoproto/grpc"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"

	"github.com/skip-mev/block-sdk/v2/block"
	"github.com/skip-mev/block-sdk/v2/block/service/types"
)

var _ types.ServiceServer = (*QueryService)(nil)

// QueryService defines the service used by the gRPC query server to query the
// Block SDK mempool.
type QueryService struct {
	types.UnimplementedServiceServer

	// mempool is the mempool instance to query.
	mempool block.Mempool
}

// NewQueryService creates a new QueryService instance.
func NewQueryService(mempool block.Mempool) *QueryService {
	return &QueryService{
		mempool: mempool,
	}
}

// GetTxDistribution returns the current distribution of transactions in the
// mempool.
func (s *QueryService) GetTxDistribution(
	_ context.Context,
	_ *types.GetTxDistributionRequest,
) (*types.GetTxDistributionResponse, error) {
	distribution := s.mempool.GetTxDistribution()
	return &types.GetTxDistributionResponse{Distribution: distribution}, nil
}

// RegisterMempoolService registers the Block SDK mempool queries on the gRPC server.

func RegisterMempoolService(
	server gogogrpc.Server,
	mempool block.Mempool,
) {
	types.RegisterServiceServer(server, NewQueryService(mempool))
}

// RegisterGRPCGatewayRoutes mounts the Block SDK mempool service's GRPC-gateway routes on the
// given Mux.
func RegisterGRPCGatewayRoutes(clientConn gogogrpc.ClientConn, mux *runtime.ServeMux) {
	_ = types.RegisterServiceHandlerClient(context.Background(), mux, types.NewServiceClient(clientConn))
}
