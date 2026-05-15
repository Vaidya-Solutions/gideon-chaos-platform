.PHONY: test fmt vet

# All Lambda entrypoints
LAMBDA_COMMANDS := steady-state start-experiment continue-execution evaluate-hypothesis

# Architecture configuration
# Default: x86_64 for local testing on developer machines
# Use `make build-arm64` or `make package-arm64` for AWS deployment (Graviton)
GOARCH ?= amd64

LAMBDA_SERVICE_NAME := lambda-chaos-machine
REPO_ROOT := $(abspath ..)
LAMBDA_BUILD_MODE := multi
LAMBDA_GO_BUILD_ARGS := -trimpath -tags lambda.norpc -ldflags='-s -w'
LAMBDA_EXTRA_CLEAN := dist coverage.out coverage.html

include ../makefiles/lambda-service.mk

test:
	go test -v -race -coverprofile=coverage.out ./...
	@go tool cover -func=coverage.out | tail -1

fmt:
	gofmt -w .

vet:
	go vet ./...
