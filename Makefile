# Formae GHA Plugin Makefile

# Plugin metadata - extracted from formae-plugin.pkl
PLUGIN_NAME := $(shell pkl eval -x 'name' formae-plugin.pkl 2>/dev/null || echo "gha")
PLUGIN_VERSION := $(shell pkl eval -x 'version' formae-plugin.pkl 2>/dev/null || echo "0.0.0")
PLUGIN_NAMESPACE := $(shell pkl eval -x 'namespace' formae-plugin.pkl 2>/dev/null || echo "GHA")

GO := go
GOFLAGS := -trimpath
BINARY := $(PLUGIN_NAME)

PLUGIN_BASE_DIR := $(HOME)/.pel/formae/plugins
INSTALL_DIR := $(PLUGIN_BASE_DIR)/$(PLUGIN_NAME)/v$(PLUGIN_VERSION)

.PHONY: all build test test-unit test-integration lint lint-reuse add-license schema-version verify-schema gen-pkl clean install install-dev help clean-environment conformance-test conformance-test-crud conformance-test-discovery conformance-test-crud-run conformance-test-discovery-run

all: build

## schema-version: Write schema/pkl/VERSION from the plugin manifest
schema-version:
	@mkdir -p schema/pkl && echo "$(PLUGIN_VERSION)" > schema/pkl/VERSION

## build: Build the plugin binary
build: schema-version
	$(GO) build $(GOFLAGS) -o bin/$(BINARY) .

## test: Run all tests
test:
	$(GO) test -v ./...

## test-unit: Run unit tests only
test-unit:
	$(GO) test -v -tags=unit ./...

## test-integration: Run integration tests (requires GITHUB_TOKEN, GHA_TEST_OWNER, GHA_TEST_REPO)
test-integration:
	$(GO) test -v -tags=integration -timeout 5m ./...

## lint: Run golangci-lint
lint:
	golangci-lint run

## lint-reuse: Check REUSE license compliance
lint-reuse:
	./scripts/lint_reuse.sh

## add-license: Add license headers to source files (idempotent)
add-license:
	./scripts/add_license.sh

## verify-schema: Validate Pkl schema files
verify-schema: schema-version
	$(GO) run github.com/platform-engineering-labs/formae/pkg/plugin/testutil/cmd/verify-schema --namespace $(PLUGIN_NAMESPACE) ./schema/pkl

## gen-pkl: Resolve all Pkl project dependencies
gen-pkl:
	pkl project resolve schema/pkl
	pkl project resolve testdata
	pkl project resolve examples

## clean: Remove build artifacts
clean:
	rm -rf bin/ dist/

## install: Build and install plugin locally
install: build
	@echo "Installing $(PLUGIN_NAME) v$(PLUGIN_VERSION) (namespace: $(PLUGIN_NAMESPACE))..."
	@rm -rf $(PLUGIN_BASE_DIR)/$(PLUGIN_NAME)
	@mkdir -p $(INSTALL_DIR)/schema/pkl
	@cp bin/$(BINARY) $(INSTALL_DIR)/$(BINARY)
	@cp -r schema/pkl/* $(INSTALL_DIR)/schema/pkl/
	@if [ -f schema/Config.pkl ]; then cp schema/Config.pkl $(INSTALL_DIR)/schema/; fi
	@cp formae-plugin.pkl $(INSTALL_DIR)/
	@echo "Installed to $(INSTALL_DIR)"

## install-dev: Build and install plugin as v0.0.0 (for formae debug builds)
DEV_INSTALL_DIR := $(PLUGIN_BASE_DIR)/$(PLUGIN_NAME)/v0.0.0
install-dev: build
	@echo "Installing $(PLUGIN_NAME) v0.0.0 (dev)..."
	@rm -rf $(PLUGIN_BASE_DIR)/$(PLUGIN_NAME)
	@mkdir -p $(DEV_INSTALL_DIR)/schema/pkl
	@cp bin/$(BINARY) $(DEV_INSTALL_DIR)/$(BINARY)
	@cp -r schema/pkl/* $(DEV_INSTALL_DIR)/schema/pkl/
	@if [ -f schema/Config.pkl ]; then cp schema/Config.pkl $(DEV_INSTALL_DIR)/schema/; fi
	@cp formae-plugin.pkl $(DEV_INSTALL_DIR)/

## help: Show this help message
help:
	@echo "Available targets:"
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/## /  /'

## clean-environment: Clean up test resources on the target repo
clean-environment:
	@./scripts/ci/clean-environment.sh

## conformance-test: Run all conformance tests (CRUD + discovery)
## Usage: make conformance-test [TEST=resource] [TIMEOUT=60]
##
## The conformance SDK installs the latest released formae via orbital
## unless FORMAE_BINARY is set (e.g. by nightly which builds from source).
conformance-test: conformance-test-crud conformance-test-discovery

## conformance-test-crud: Run CRUD lifecycle tests with cleanup (convenience for local dev)
## Usage: make conformance-test-crud [TEST=resource] [TIMEOUT=60]
conformance-test-crud: install
	@echo "Pre-test cleanup..."
	@./scripts/ci/clean-environment.sh || true
	@echo ""
	@$(MAKE) conformance-test-crud-run TEST=$(TEST) TIMEOUT=$(TIMEOUT); \
	TEST_EXIT=$$?; \
	echo ""; \
	echo "Post-test cleanup..."; \
	./scripts/ci/clean-environment.sh || true; \
	exit $$TEST_EXIT

## conformance-test-discovery: Run discovery tests with cleanup (convenience for local dev)
## Usage: make conformance-test-discovery [TEST=resource] [TIMEOUT=60]
conformance-test-discovery: install
	@echo "Pre-test cleanup..."
	@./scripts/ci/clean-environment.sh || true
	@echo ""
	@$(MAKE) conformance-test-discovery-run TEST=$(TEST) TIMEOUT=$(TIMEOUT); \
	TEST_EXIT=$$?; \
	echo ""; \
	echo "Post-test cleanup..."; \
	./scripts/ci/clean-environment.sh || true; \
	exit $$TEST_EXIT

## conformance-test-crud-run: Run only CRUD lifecycle tests (no cleanup)
## Used by CI matrix jobs where cleanup is managed separately.
conformance-test-crud-run:
	@echo "Running CRUD conformance tests..."
	@FORMAE_TEST_FILTER="$(TEST)" FORMAE_TEST_TYPE=crud \
		$(GO) test -tags=conformance -v -timeout $(or $(TIMEOUT),60)m ./...

## conformance-test-discovery-run: Run only discovery tests (no cleanup)
## Used by CI matrix jobs where cleanup is managed separately.
conformance-test-discovery-run:
	@echo "Running discovery conformance tests..."
	@FORMAE_TEST_FILTER="$(TEST)" FORMAE_TEST_TYPE=discovery \
		$(GO) test -tags=conformance -v -timeout $(or $(TIMEOUT),60)m ./...
