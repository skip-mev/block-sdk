module github.com/skip-mev/block-sdk/v2

go 1.22.4

require (
	cosmossdk.io/api v0.7.2
	cosmossdk.io/client/v2 v2.0.0-20230724130706-5442197d6bcd
	cosmossdk.io/core v0.11.0
	cosmossdk.io/depinject v1.0.0-alpha.4
	cosmossdk.io/errors v1.0.1
	cosmossdk.io/log v1.3.1
	cosmossdk.io/math v1.2.0
	cosmossdk.io/store v1.0.2
	cosmossdk.io/tools/confix v0.1.1
	cosmossdk.io/x/circuit v0.1.0
	cosmossdk.io/x/feegrant v0.1.0
	cosmossdk.io/x/tx v0.13.0
	cosmossdk.io/x/upgrade v0.1.1
	github.com/client9/misspell v0.3.4
	github.com/cometbft/cometbft v0.38.5
	github.com/cosmos/cosmos-db v1.0.0
	github.com/cosmos/cosmos-proto v1.0.0-beta.3
	github.com/cosmos/cosmos-sdk v0.50.3
	github.com/cosmos/gogoproto v1.4.11
	github.com/golang/protobuf v1.5.4
	github.com/golangci/golangci-lint v1.59.1
	github.com/gorilla/mux v1.8.1
	github.com/grpc-ecosystem/grpc-gateway v1.16.0
	github.com/huandu/skiplist v1.2.0
	github.com/skip-mev/chaintestutil v0.0.0-20231221145345-f208ee3b1383
	github.com/spf13/cobra v1.8.0
	github.com/spf13/pflag v1.0.5
	github.com/spf13/viper v1.18.2
	github.com/stretchr/testify v1.9.0
	golang.org/x/tools v0.22.0
	golang.org/x/vuln v1.1.2
	google.golang.org/genproto/googleapis/api v0.0.0-20240528184218-531527333157
	google.golang.org/grpc v1.65.0
	google.golang.org/protobuf v1.34.1
	mvdan.cc/gofumpt v0.6.0
)



replace (
	github.com/99designs/keyring => github.com/cosmos/keyring v1.2.0
	github.com/syndtr/goleveldb => github.com/syndtr/goleveldb v1.0.1-0.20210819022825-2ae1ddf74ef7
)
