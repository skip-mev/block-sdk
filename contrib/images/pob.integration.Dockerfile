FROM golang:1.20-bullseye AS builder

WORKDIR /src/pob
COPY . .

RUN go mod tidy
RUN make build-test-app

## Prepare the final clear binary
FROM ubuntu:rolling
EXPOSE 26656 26657 1317 9090 7171

COPY --from=builder /src/pob/build/* /usr/local/bin/
RUN apt-get update && apt-get install ca-certificates -y
