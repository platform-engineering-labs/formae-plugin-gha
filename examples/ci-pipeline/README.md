# CI Pipeline

Bootstrap the CI/CD pipeline for deploying AWS infrastructure with formae.

## What You Get

- Apply workflow (manual dispatch, per-environment)
- Destroy workflow (manual dispatch, confirmation required)
- Staging environment (deploys from release/*, develop)
- Production environment (15min wait timer, self-review prevention)
- Per-environment AWS OIDC secrets and deployment variables
- Deployment branch policies
- Repo-level variables (project name, region, agent URL)

## Prerequisites

1. formae CLI installed
2. GitHub PAT with `repo` + `workflow` scopes
3. A target repository to configure

## Configuration

Set your target repository:

```bash
export GHA_OWNER=my-org
export GHA_REPO=my-repo
```

`GHA_OWNER` and `GHA_REPO` are read during Pkl evaluation in the CLI.

Authentication is resolved automatically via the auth chain (`GITHUB_TOKEN` env,
`gh auth token`, or `~/.config/gh/hosts.yml`). If you have `gh` authenticated,
no extra setup is needed.

Edit `main.pkl` to customize:

- AWS role ARNs for each environment
- VPC CIDR ranges
- Branch patterns
- Formae agent URL

## Deploy

```bash
formae apply examples/ci-pipeline/main.pkl
```

## Tear Down

```bash
formae destroy examples/ci-pipeline/main.pkl
```

## Architecture

```
Repository (my-org/my-repo)
├── .github/workflows/
│   ├── apply.yml (workflow_dispatch)
│   └── destroy.yml (workflow_dispatch + confirmation)
├── Variables
│   ├── PROJECT_NAME
│   ├── AWS_DEFAULT_REGION
│   └── FORMAE_AGENT_URL
├── Staging Environment
│   ├── Branch Policies: release/*, develop
│   ├── Secret: AWS_ROLE_ARN
│   └── Variables: AWS_REGION, VPC_CIDR
└── Production Environment (15min wait, no self-review)
    ├── Branch Policy: main
    ├── Secret: AWS_ROLE_ARN
    └── Variables: AWS_REGION, VPC_CIDR
```

## How It Works

The workflow files are written as typed Pkl using the
[com.github.actions](https://pkl-lang.org/package-docs/pkg.pkl-lang.org/pkl-pantry/com.github.actions/current/index.html)
package from pkl-pantry, rendered to YAML via `YamlRenderer`, and committed
to the repository as `GHA::Repo::File` resources. The workflows and the
configuration they depend on (secrets, variables, environments) are declared
in the same forma — one apply creates everything.
