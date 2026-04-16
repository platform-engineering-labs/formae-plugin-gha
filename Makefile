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

.PHONY: all build test test-unit test-integration lint verify-schema gen-pkl clean install install-dev help clean-environment conformance-test conformance-test-crud conformance-test-discovery

all: build

## build: Build the plugin binary
build:
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

## verify-schema: Validate Pkl schema files
verify-schema:
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
	@cp formae-plugin.pkl $(DEV_INSTALL_DIR)/

## help: Show this help message
help:
	@echo "Available targets:"
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/## /  /'

## clean-environment: Clean up test resources on the target repo
clean-environment:
	@./scripts/ci/clean-environment.sh

## conformance-test: Run all conformance tests (CRUD + discovery)
## Usage: make conformance-test [VERSION=0.84.0] [TEST=resource] [TIMEOUT=5]
conformance-test: conformance-test-crud conformance-test-discovery

## conformance-test-crud: Run only CRUD lifecycle tests
## Usage: make conformance-test-crud [VERSION=0.84.0] [TEST=resource] [TIMEOUT=5]
conformance-test-crud: install
	@echo "Pre-test cleanup..."
	@./scripts/ci/clean-environment.sh || true
	@echo ""
	@echo "Running CRUD conformance tests..."
	@FORMAE_TEST_FILTER="$(TEST)" FORMAE_TEST_TYPE=crud FORMAE_TEST_TIMEOUT="$(TIMEOUT)" ./scripts/run-conformance-tests.sh $(VERSION); \
	TEST_EXIT=$$?; \
	echo ""; \
	echo "Post-test cleanup..."; \
	./scripts/ci/clean-environment.sh || true; \
	exit $$TEST_EXIT

## conformance-test-discovery: Run only discovery tests
## Usage: make conformance-test-discovery [VERSION=0.84.0] [TEST=resource] [TIMEOUT=5]
conformance-test-discovery: install
	@echo "Pre-test cleanup..."
	@./scripts/ci/clean-environment.sh || true
	@echo ""
	@echo "Running discovery conformance tests..."
	@FORMAE_TEST_FILTER="$(TEST)" FORMAE_TEST_TYPE=discovery FORMAE_TEST_TIMEOUT="$(TIMEOUT)" ./scripts/run-conformance-tests.sh $(VERSION); \
	TEST_EXIT=$$?; \
	echo ""; \
	echo "Post-test cleanup..."; \
	./scripts/ci/clean-environment.sh || true; \
	exit $$TEST_EXIT
