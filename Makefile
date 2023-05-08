#!/usr/bin/make -f

export VERSION := $(shell echo $(shell git describe --always --match "v*") | sed 's/^v//')
export COMMIT := $(shell git log -1 --format='%H')
export COMETBFT_VERSION := $(shell go list -m github.com/cometbft/cometbft | sed 's:.* ::')

BIN_DIR ?= $(GOPATH)/bin
BUILD_DIR ?= $(CURDIR)/build
PROJECT_NAME = $(shell git remote get-url origin | xargs basename -s .git)
HTTPS_GIT := https://github.com/skip-mev/pob.git
DOCKER := $(shell which docker)

###############################################################################
###                                Test App                                 ###
###############################################################################

whitespace :=
whitespace += $(whitespace)
comma := ,
build_tags_comma_sep := $(subst $(whitespace),$(comma),$(build_tags))

ldflags = -X github.com/cosmos/cosmos-sdk/version.Name=testapp \
		  -X github.com/cosmos/cosmos-sdk/version.AppName=testappd \
		  -X github.com/cosmos/cosmos-sdk/version.Version=$(VERSION) \
		  -X github.com/cosmos/cosmos-sdk/version.Commit=$(COMMIT) \
		  -X "github.com/cosmos/cosmos-sdk/version.BuildTags=$(build_tags_comma_sep)" \
		  -X github.com/cometbft/cometbft/version.TMCoreSemVer=$(COMETBFT_VERSION)

# DB backend selection
ifeq (cleveldb,$(findstring cleveldb,$(COSMOS_BUILD_OPTIONS)))
  build_tags += gcc
endif
ifeq (badgerdb,$(findstring badgerdb,$(COSMOS_BUILD_OPTIONS)))
  build_tags += badgerdb
endif
# handle rocksdb
ifeq (rocksdb,$(findstring rocksdb,$(COSMOS_BUILD_OPTIONS)))
  CGO_ENABLED=1
  build_tags += rocksdb
endif
# handle boltdb
ifeq (boltdb,$(findstring boltdb,$(COSMOS_BUILD_OPTIONS)))
  build_tags += boltdb
endif

ifeq (,$(findstring nostrip,$(COSMOS_BUILD_OPTIONS)))
  ldflags += -w -s
endif

ldflags += $(LDFLAGS)
ldflags := $(strip $(ldflags))

build_tags += $(BUILD_TAGS)
build_tags := $(strip $(build_tags))

BUILD_FLAGS := -tags "$(build_tags)" -ldflags '$(ldflags)'

# check for nostrip option
ifeq (,$(findstring nostrip,$(COSMOS_BUILD_OPTIONS)))
  BUILD_FLAGS += -trimpath
endif

BUILD_TARGETS := build-test-app

build-test-app: BUILD_ARGS=-o $(BUILD_DIR)/

$(BUILD_TARGETS): $(BUILD_DIR)/
	cd $(CURDIR)/tests/app && go build -mod=readonly $(BUILD_FLAGS) $(BUILD_ARGS) ./...

$(BUILD_DIR)/:
	mkdir -p $(BUILD_DIR)/

.PHONY: build-test-app

###############################################################################
##                                  Docker                                   ##
###############################################################################

docker-build:
	@echo "Building E2E Docker image..."
	@DOCKER_BUILDKIT=1 docker build -t skip-mev/pob-e2e -f contrib/images/pob.e2e.Dockerfile .

###############################################################################
###                                  Tests                                  ###
###############################################################################

TEST_E2E_TAGS = e2e
TEST_E2E_DEPS = docker-build

test-e2e: $(TEST_E2E_DEPS)
	@echo "Running E2E tests..."
	@go test ./tests/e2e/... -mod=readonly -timeout 30m -race -v -tags='$(TEST_E2E_TAGS)'

test:
	@go test -v ./...

.PHONY: test test-e2e

###############################################################################
###                                Protobuf                                 ###
###############################################################################

protoVer=0.11.6
protoImageName=ghcr.io/cosmos/proto-builder:$(protoVer)
protoImage=$(DOCKER) run --rm -v $(CURDIR):/workspace --workdir /workspace $(protoImageName)

proto-all: proto-format proto-lint proto-gen

proto-gen:
	@echo "Generating Protobuf files"
	@$(protoImage) sh ./scripts/protocgen.sh

proto-pulsar-gen:
	@echo "Generating Dep-Inj Protobuf files"
	@$(protoImage) sh ./scripts/protocgen-pulsar.sh

proto-format:
	@$(protoImage) find ./ -name "*.proto" -exec clang-format -i {} \;

proto-lint:
	@$(protoImage) buf lint --error-format=json

proto-check-breaking:
	@$(protoImage) buf breaking --against $(HTTPS_GIT)#branch=main

proto-update-deps:
	@echo "Updating Protobuf dependencies"
	$(DOCKER) run --rm -v $(CURDIR)/proto:/workspace --workdir /workspace $(protoImageName) buf mod update

.PHONY: proto-all proto-gen proto-format proto-lint proto-check-breaking proto-update-deps

###############################################################################
###                                Linting                                  ###
###############################################################################

golangci_lint_cmd=golangci-lint
golangci_version=v1.51.2

lint:
	@echo "--> Running linter"
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@$(golangci_version)
	@golangci-lint run

lint-fix:
	@echo "--> Running linter"
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@$(golangci_version)
	@golangci-lint run --fix

lint-markdown:
	@echo "--> Running markdown linter"
	@markdownlint **/*.md

.PHONY: lint lint-fix lint-markdown
