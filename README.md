# GitHub Actions plugin for formae

[![CI](https://github.com/platform-engineering-labs/formae-plugin-gha/actions/workflows/ci.yml/badge.svg?branch=main)](https://github.com/platform-engineering-labs/formae-plugin-gha/actions/workflows/ci.yml)
[![Monthly](https://github.com/platform-engineering-labs/formae-plugin-gha/actions/workflows/monthly.yml/badge.svg?branch=main)](https://github.com/platform-engineering-labs/formae-plugin-gha/actions/workflows/monthly.yml)

Manage GitHub Actions CI/CD infrastructure as code. Secrets, variables, environments,
branch policies, and workflow files — declared in Pkl, applied with formae.

[formae](https://github.com/platform-engineering-labs/formae) · [Hub](https://hub.platform.engineering/platform.engineering/gha)

## Install

Requires the formae CLI: see the [quick start](https://docs.formae.ai/documentation/get-started/quickstart).

```bash
formae plugin install gha
```

Restart the formae agent afterwards so it loads the plugin.

**New project:** with the agent running, `formae project init --include gha my-project` creates `my-project` with a `PklProject` that declares the formae and gha schema packages, so `import "@gha/..."` resolves, and a starter `main.pkl`. Don't run it in an existing project: it overwrites both files.

**Existing project:** add the plugin to `dependencies` in your `PklProject`, with the current version from the [hub page](https://hub.platform.engineering/platform.engineering/gha), then run `pkl project resolve`:

```pkl
["gha"] {
  uri = "package://hub.platform.engineering/plugins/gha/schema/pkl/gha/gha@<version>"
}
```

Next: [write your first forma](https://docs.formae.ai/documentation/get-started/write-your-first-forma), then [`formae apply`](https://docs.formae.ai/documentation/reference/cli/apply) (see [apply modes](https://docs.formae.ai/documentation/concepts/apply-modes)).

With an AI coding assistant, use the [formae plugin](https://docs.formae.ai/documentation/guides/ai-coding-assistants) (formerly `formae-mcp`), which can search the hub and fetch plugin examples. The formae documentation is also available as [llms.txt](https://docs.formae.ai/llms.txt).

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

## License

FSL-1.1-ALv2
