# NOTE: This file is forked from the Cosmos SDK Proto Builder at tag v0.47.0. With a single change to not specify the user for the 
#       container to make the Docker image compatible with the GitHub action CLI (see bullet #2 under 'Docker Container Filesystem') here:
#       https://docs.github.com/en/actions/using-github-hosted-runners/about-github-hosted-runners#docker-container-filesystem
#       
# Permalink to source file: https://github.com/cosmos/cosmos-sdk/blob/9acdbb94da5a4687495466f88d66265c732dcdd5/contrib/devtools/Dockerfile

## To test locally:
# docker build --pull --rm -f "contrib/devtools/Dockerfile" -t cosmossdk-proto:latest "contrib/devtools"
# docker run --rm -v $(pwd):/workspace --workdir /workspace cosmossdk-proto sh ./scripts/protocgen.sh

FROM bufbuild/buf:1.24.0 as BUILDER
FROM golang:1.20-alpine

RUN apk add --no-cache \
  nodejs \
  npm \
  git \
  make \
  clang-extra-tools \
  g++ \
  jq

RUN npm install -g swagger-combine

ENV GOLANG_PROTOBUF_VERSION=1.28.1 \
  GRPC_GATEWAY_VERSION=1.16.0

RUN go install github.com/cosmos/cosmos-proto/cmd/protoc-gen-go-pulsar@latest && \
  go install google.golang.org/protobuf/cmd/protoc-gen-go@v${GOLANG_PROTOBUF_VERSION} && \
  go install github.com/grpc-ecosystem/grpc-gateway/protoc-gen-grpc-gateway@v${GRPC_GATEWAY_VERSION} \
  github.com/grpc-ecosystem/grpc-gateway/protoc-gen-swagger@v${GRPC_GATEWAY_VERSION} && \
  go install cosmossdk.io/orm/cmd/protoc-gen-go-cosmos-orm@v1.0.0-beta.3 && \
  go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# install all gogo protobuf binaries
RUN git clone https://github.com/cosmos/gogoproto.git; \
  cd gogoproto; \
  go mod download; \
  make install

COPY --from=BUILDER /usr/local/bin /usr/local/bin
