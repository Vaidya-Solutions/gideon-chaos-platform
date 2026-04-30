# ==============================================================================
# Shared Lambda Service Makefile Rules
# ==============================================================================
# Grouped chaos platform Lambda Makefiles declare service metadata and any
# truly unique helper targets, then include this file for shared build /
# package / clean behavior owned by the grouped runtime boundary.

ifndef LAMBDA_SERVICE_NAME
$(error LAMBDA_SERVICE_NAME is required before including makefiles/lambda-service.mk)
endif

GOARCH ?= amd64
REPO_ROOT ?= $(abspath ..)
LAMBDA_BUILD_MODE ?= single
LAMBDA_BUILD_TARGET ?= .
LAMBDA_GO_BUILD_ARGS ?= -trimpath -ldflags='-s -w'
LAMBDA_EXTRA_CLEAN ?=
GO_CMD ?= env -u GOROOT -u GOTOOLCHAIN GOWORK=off go
GOLANGCI_LINT_VERSION ?= v2.11.4
GOLANGCI_LINT ?= $(GO_CMD) run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)
GOLANGCI_FAST_FLAG ?= --fast-only

BIN_ROOT := $(REPO_ROOT)/.artifacts/$(LAMBDA_SERVICE_NAME)/$(GOARCH)
PACKAGE_ROOT := $(BIN_ROOT)

BIN_DIR := $(BIN_ROOT)
PACKAGE_DIR := $(PACKAGE_ROOT)
BINARY_PATH := $(BIN_ROOT)/bootstrap
ZIP_PATH := $(PACKAGE_ROOT)/lambda.zip

.PHONY: build build-arm64 package package-arm64 clean lint lint-fast

build:
	@set -e; \
	if [ "$(LAMBDA_BUILD_MODE)" = "multi" ]; then \
		mkdir -p "$(BIN_ROOT)"; \
		for cmd in $(LAMBDA_COMMANDS); do \
			echo "Building $$cmd ($(GOARCH))..."; \
			GOOS=linux GOARCH=$(GOARCH) CGO_ENABLED=0 $(GO_CMD) build $(LAMBDA_GO_BUILD_ARGS) \
				-o "$(BIN_ROOT)/$$cmd/bootstrap" ./cmd/$$cmd; \
		done; \
		echo "✓ Built $(words $(LAMBDA_COMMANDS)) binaries"; \
	else \
		mkdir -p "$(BIN_DIR)"; \
		GOOS=linux GOARCH=$(GOARCH) CGO_ENABLED=0 $(GO_CMD) build $(LAMBDA_GO_BUILD_ARGS) \
			-o "$(BINARY_PATH)" $(LAMBDA_BUILD_TARGET); \
		echo "Built $(BINARY_PATH)"; \
	fi

build-arm64:
	@$(MAKE) build GOARCH=arm64

package: build
	@set -e; \
	if [ "$(LAMBDA_BUILD_MODE)" = "multi" ]; then \
		mkdir -p "$(PACKAGE_ROOT)"; \
		for cmd in $(LAMBDA_COMMANDS); do \
			mkdir -p "$(PACKAGE_ROOT)/$$cmd"; \
			(cd "$(BIN_ROOT)/$$cmd" && zip -q -j "$(PACKAGE_ROOT)/$$cmd/lambda.zip" bootstrap); \
		done; \
		echo "✓ Packaged $(words $(LAMBDA_COMMANDS)) Lambda zips"; \
	else \
		mkdir -p "$(PACKAGE_DIR)"; \
		cd "$(BIN_DIR)" && zip -q -j "$(ZIP_PATH)" bootstrap; \
		echo "Packaged $(ZIP_PATH)"; \
	fi

package-arm64:
	@$(MAKE) package GOARCH=arm64

clean:
	@rm -rf "$(REPO_ROOT)/.artifacts/$(LAMBDA_SERVICE_NAME)" \
		$(LAMBDA_EXTRA_CLEAN)

lint:
	@$(GOLANGCI_LINT) run ./...

lint-fast:
	@$(GOLANGCI_LINT) run $(GOLANGCI_FAST_FLAG) ./...