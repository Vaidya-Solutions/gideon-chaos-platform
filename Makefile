# Copyright (c) Vaidya Solutions
# SPDX-License-Identifier: BSD-3-Clause
#
# services/gideon-chaos-platform/Makefile
# @file services/gideon-chaos-platform/Makefile
# @description Wrapper-only commands for the grouped Gideon chaos platform boundary.
# @update-policy Update when grouped wrapper targets change; do not duplicate runtime build logic here.

.DEFAULT_GOAL := help

.PHONY: help check-layout test lint package-arm64 verify-all clean

CHAOS_PLATFORM_RUNTIME_SERVICES := \
	lambda-chaos-machine

help: ## Show grouped chaos platform wrapper commands
	@awk 'BEGIN {FS = ":.*## "} /^[a-zA-Z0-9_-]+:.*## / {printf "%-16s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

check-layout:
	@set -e; \
	for service in $(CHAOS_PLATFORM_RUNTIME_SERVICES); do \
		if [ ! -d "$$service" ]; then \
			echo "Expected $$service/ under services/gideon-chaos-platform before grouped commands can run."; \
			echo "Move the current lambda-chaos-machine runtime into the grouped root first."; \
			exit 1; \
		fi; \
	done

test: check-layout ## Run unit tests for grouped chaos platform runtimes
	@set -e; \
	for service in $(CHAOS_PLATFORM_RUNTIME_SERVICES); do \
		echo "Testing $$service..."; \
		$(MAKE) -C "$$service" test; \
	done

lint: check-layout ## Run linting for grouped chaos platform runtimes
	@set -e; \
	for service in $(CHAOS_PLATFORM_RUNTIME_SERVICES); do \
		echo "Linting $$service..."; \
		$(MAKE) -C "$$service" lint; \
	done

package-arm64: check-layout ## Package ARM64 artifacts for grouped chaos platform runtimes
	@set -e; \
	for service in $(CHAOS_PLATFORM_RUNTIME_SERVICES); do \
		echo "Packaging ARM64 artifacts for $$service..."; \
		$(MAKE) -C "$$service" package-arm64; \
	done

verify-all: ## Run lint, tests, and ARM64 packaging for grouped chaos platform runtimes
	@$(MAKE) lint
	@$(MAKE) test
	@$(MAKE) package-arm64

clean: check-layout ## Remove grouped chaos platform local build artifacts
	@set -e; \
	for service in $(CHAOS_PLATFORM_RUNTIME_SERVICES); do \
		echo "Cleaning $$service..."; \
		$(MAKE) -C "$$service" clean; \
	done
