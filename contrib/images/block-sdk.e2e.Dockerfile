FROM golang:1.22-bullseye AS builder

WORKDIR /src/bsdk
COPY . .

RUN go mod tidy && make build-test-app

## Prepare the final clear binary
FROM ubuntu:rolling
EXPOSE 26656 26657 1317 9090 7171

COPY --from=builder /src/bsdk/build/* /usr/local/bin/
RUN apt-get update && apt-get install ca-certificates -y
