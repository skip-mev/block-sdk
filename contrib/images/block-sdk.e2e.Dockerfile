FROM golang:1.21-bullseye AS builder

WORKDIR /src/bsdk
COPY . .

RUN go mod tidy && make build-test-app

## Prepare the final clear binary
FROM ubuntu:rolling
EXPOSE 26656 26657 1317 9090 7171

<<<<<<<< HEAD:contrib/images/pob.integration.Dockerfile
COPY --from=builder /src/pob/build/* /usr/local/bin/
RUN apt-get update && apt-get install ca-certificates -yV
========
COPY --from=builder /src/bsdk/build/* /usr/local/bin/
RUN apt-get update && apt-get install ca-certificates -y
>>>>>>>> 7e279c5 (chore: rename `integration` to `e2e` (#291)):contrib/images/block-sdk.e2e.Dockerfile
