#!/usr/bin/make -f

export VERSION := $(shell echo $(shell git describe --always --match "v*") | sed 's/^v//')
# export TM_VERSION := $(shell go list -m github.com/cometbft/cometbft | sed 's:.* ::')
export COMMIT := $(shell git log -1 --format='%H')

BIN_DIR ?= $(GOPATH)/bin
BUILD_DIR ?= $(CURDIR)/build
PROJECT_NAME = $(shell git remote get-url origin | xargs basename -s .git)
HTTPS_GIT := https://github.com/skip-mev/pob.git
DOCKER := $(shell which docker)

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

proto-format:
	@$(protoImage) find ./ -name "*.proto" -exec clang-format -i {} \;

proto-lint:
	@$(protoImage) buf lint --error-format=json

proto-check-breaking:
	@$(protoImage) buf breaking --against $(HTTPS_GIT)#branch=main

# TODO: Update/remove after v0.37.x tag of CometBFT.
# TM_URL              = https://raw.githubusercontent.com/cometbft/cometbft/387422ac220d/proto/tendermint

# TM_CRYPTO_TYPES     = proto/tendermint/crypto
# TM_ABCI_TYPES       = proto/tendermint/abci
# TM_TYPES            = proto/tendermint/types
# TM_VERSION          = proto/tendermint/version
# TM_LIBS             = proto/tendermint/libs/bits
# TM_P2P              = proto/tendermint/p2p

proto-update-deps:
	@echo "Updating Protobuf dependencies"

	# @mkdir -p $(TM_ABCI_TYPES)
	# @curl -sSL $(TM_URL)/abci/types.proto > $(TM_ABCI_TYPES)/types.proto

	# @mkdir -p $(TM_VERSION)
	# @curl -sSL $(TM_URL)/version/types.proto > $(TM_VERSION)/types.proto

	# @mkdir -p $(TM_TYPES)
	# @curl -sSL $(TM_URL)/types/types.proto > $(TM_TYPES)/types.proto
	# @curl -sSL $(TM_URL)/types/evidence.proto > $(TM_TYPES)/evidence.proto
	# @curl -sSL $(TM_URL)/types/params.proto > $(TM_TYPES)/params.proto
	# @curl -sSL $(TM_URL)/types/validator.proto > $(TM_TYPES)/validator.proto
	# @curl -sSL $(TM_URL)/types/block.proto > $(TM_TYPES)/block.proto

	# @mkdir -p $(TM_CRYPTO_TYPES)
	# @curl -sSL $(TM_URL)/crypto/proof.proto > $(TM_CRYPTO_TYPES)/proof.proto
	# @curl -sSL $(TM_URL)/crypto/keys.proto > $(TM_CRYPTO_TYPES)/keys.proto

	# @mkdir -p $(TM_LIBS)
	# @curl -sSL $(TM_URL)/libs/bits/types.proto > $(TM_LIBS)/types.proto

	# @mkdir -p $(TM_P2P)
	# @curl -sSL $(TM_URL)/p2p/types.proto > $(TM_P2P)/types.proto

	$(DOCKER) run --rm -v $(CURDIR)/proto:/workspace --workdir /workspace $(protoImageName) buf mod update

.PHONY: proto-all proto-gen proto-format proto-lint proto-check-breaking proto-update-deps
