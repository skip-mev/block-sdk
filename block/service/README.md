# Block SDK Mempool Service

The Block SDK mempool service is a service that allows you to query the current state of the application side mempool.

## Usage

The mempool service is a standard gRPC service that can be paired with http or grpc clients.

### HTTP Clients

To make requests to the mempool service using HTTP, you have to use the grpc-gateway defined on your application's server. This is usually hosted on port 1317.

### gRPC Clients

To query the mempool service using gRPC, you can use the Mempool `ServiceClient` defined in [types](./types/query.pb.go):

```golang
type serviceClient struct {
	cc grpc1.ClientConn
}

func NewServiceClient(cc grpc1.ClientConn) ServiceClient {
	return &serviceClient{cc}
}

func (c *serviceClient) GetTxDistribution(ctx context.Context, in *GetTxDistributionRequest, opts ...grpc.CallOption) (*GetTxDistributionResponse, error) {
	out := new(GetTxDistributionResponse)
	err := c.cc.Invoke(ctx, "/sdk.mempool.v1.Service/GetTxDistribution", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}
```

## Endpoints

### GetTxDistribution

GetTxDistribution returns the current distribution of transactions in the mempool. The response is a map of the lane name to the number of transactions in that lane.

```golang
type GetTxDistributionRequest struct {}

type GetTxDistributionResponse struct {
    Distribution map[string]uint64
}
```

### HTTP Requests

To query the mempool service using HTTP, you can use the following endpoint:

```bash
curl http://localhost:1317/block-sdk/mempool/v1/distribution
```
