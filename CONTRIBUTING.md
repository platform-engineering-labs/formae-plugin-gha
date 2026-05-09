# Contributing

This document covers local development for plugin authors. For user-facing
plugin docs (configuration, supported resources, examples), see
[README.md](README.md).

## Prerequisites

- Go 1.25+
- [Pkl CLI](https://pkl-lang.org/main/current/pkl-cli/index.html)
- A GitHub PAT with `repo` + `workflow` scopes

## Local Installation

```bash
make install
```

## Building

```bash
make build          # Build plugin binary
make install        # Build + install locally
make lint           # Run linter
make verify-schema  # Validate Pkl schemas
```

## Testing

Integration and conformance tests hit the real GitHub API. Export:

```bash
export GITHUB_TOKEN=$(gh auth token)
export GHA_TEST_OWNER=my-org         # owner of a scratch repo
export GHA_TEST_REPO=my-test-repo    # clean-environment.sh wipes this repo
```

```bash
make test-integration   # direct API coverage per resource
make conformance-test   # full CRUD lifecycle through the formae agent
```

To run conformance against a local formae build (e.g. an unreleased version),
point the harness at the binary:

```bash
export FORMAE_BINARY=/path/to/formae
make conformance-test
```

## Clean Environment

```bash
GHA_TEST_OWNER=my-org GHA_TEST_REPO=my-test-repo ./scripts/ci/clean-environment.sh
```
