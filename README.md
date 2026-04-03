# GitHub Actions Plugin for formae

Manage GitHub Actions CI/CD infrastructure as code. Secrets, variables, environments,
branch policies, and workflow files — declared in Pkl, applied with formae.

## Installation

```bash
make install
```

## Supported Resources

| Resource Type | Description |
|---|---|
| `GHA::Repo::Variable` | Repository Actions variable |
| `GHA::Repo::Secret` | Repository Actions secret (sealed-box encrypted) |
| `GHA::Repo::Environment` | Deployment environment with protection rules |
| `GHA::Repo::File` | Repository file via Contents API |
| `GHA::Repo::Workflow` | Structured workflow file (typed via com.github.actions) |
| `GHA::Repo::OIDCClaims` | Repository OIDC subject claim customization |
| `GHA::Environment::Variable` | Environment-scoped variable |
| `GHA::Environment::Secret` | Environment-scoped secret |
| `GHA::Environment::BranchPolicy` | Deployment branch policy |
| `GHA::Org::Variable` | Organization Actions variable |
| `GHA::Org::Secret` | Organization Actions secret |
| `GHA::Org::RunnerGroup` | Organization self-hosted runner group |
| `GHA::Org::OIDCClaims` | Organization OIDC subject claim customization |
| `GHA::Org::Permissions` | Organization Actions permissions |
| `GHA::Org::DefaultWorkflowPermissions` | Organization default workflow token permissions |

## Configuration

### Target

```pkl
import "@gha/gha.pkl"

new formae.Target {
    label = "github"
    namespace = "GHA"
    config = new gha.Config {
        owner = "my-org"
        repo = "my-repo"
    }
}
```

### Authentication

The plugin resolves a GitHub token automatically via this chain:

1. `GITHUB_TOKEN` environment variable
2. `gh auth token` CLI command
3. `~/.config/gh/hosts.yml` config file

No agent restart needed when tokens rotate. Required scopes:

| Scope | Required for |
|---|---|
| `repo` | All resources |
| `workflow` | Files under `.github/workflows/` |
| `admin:org` | Organization-scoped resources |

### Workflow Files

Workflow definitions can be written in typed Pkl using the
[com.github.actions](https://pkl-lang.org/package-docs/pkg.pkl-lang.org/pkl-pantry/com.github.actions/current/index.html)
package from pkl-pantry, rendered to YAML via `YamlRenderer`, and committed
to the repo as a `GHA::Repo::File` resource.

## Example

The [ci-pipeline example](examples/ci-pipeline/) bootstraps the complete CI/CD pipeline
for deploying AWS infrastructure with formae. One forma creates apply and destroy
workflows, staging and production environments with approval gates, AWS OIDC
secrets, deployment variables, and branch policies.

```bash
export GITHUB_TOKEN=$(gh auth token)
export GHA_OWNER=my-org
export GHA_REPO=my-repo

formae apply examples/ci-pipeline/main.pkl
```

See the [example README](examples/ci-pipeline/README.md) for details.

## Development

### Prerequisites

- Go 1.25+
- [Pkl CLI](https://pkl-lang.org/main/current/pkl-cli/index.html)
- A GitHub PAT with `repo` + `workflow` scopes

### Building

```bash
make build          # Build plugin binary
make install        # Build + install locally
make lint           # Run linter
make verify-schema  # Validate Pkl schemas
```

### Testing

```bash
# Integration tests (requires GITHUB_TOKEN, GHA_TEST_OWNER, GHA_TEST_REPO)
go test -v -tags=integration ./pkg/resources/...

# Conformance tests (full CRUD lifecycle through formae agent)
make conformance-test
```

### Clean Environment

```bash
GHA_TEST_OWNER=my-org GHA_TEST_REPO=my-test-repo ./scripts/ci/clean-environment.sh
```

## License

Apache-2.0
