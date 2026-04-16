# Infrastructure to Application

Provision a database, deploy an application, verify it works — all from one pipeline.

## What You Get

- Deploy workflow (provisions Azure PostgreSQL → deploys Miniflux → verifies feeds)
- Destroy workflow (tears down container + database)
- Azure OIDC secrets for authentication
- PostgreSQL password secret
- Project configuration variables

## How It Works

1. **Provision** — `formae apply` creates an Azure PostgreSQL Flexible Server
2. **Extract** — `jq` parses the DB endpoint from formae's inventory output
3. **Deploy** — Azure Container Instances runs Miniflux with the database URL
4. **Verify** — health check, 5 RSS feeds added, article count confirmed

Two formae plugins working together: GHA manages the pipeline, Azure manages
the infrastructure.

## Prerequisites

1. formae CLI installed
2. GitHub PAT with `repo` + `workflow` scopes
3. Azure subscription with OIDC configured for GitHub Actions
4. A target repository

## Configuration

Set your target repository:

```bash
export GHA_OWNER=my-org
export GHA_REPO=my-repo
```

Edit `main.pkl` to customize:

- Azure OIDC credentials (client ID, tenant ID, subscription ID)
- PostgreSQL password
- Azure region
- Formae agent URL

Edit `infra/vars.pkl` for database-specific settings.

## Deploy

```bash
formae apply examples/infra-to-app/main.pkl
```

Then trigger the deploy workflow in the GitHub UI.

## Tear Down

Trigger the destroy workflow in the GitHub UI (type "destroy" to confirm).

To remove the GHA resources:

```bash
formae destroy examples/infra-to-app/main.pkl
```

## Architecture

```
GitHub Actions (managed by GHA plugin)
├── .github/workflows/
│   ├── deploy.yml
│   │   ├── Job: provision (formae apply → Azure PostgreSQL)
│   │   ├── Job: deploy-app (Miniflux on Azure Container Instances)
│   │   └── Job: verify (healthcheck + RSS feeds)
│   └── destroy.yml
│       └── Job: destroy (container + database teardown)
├── Secrets: AZURE_CLIENT_ID, AZURE_TENANT_ID, AZURE_SUBSCRIPTION_ID, POSTGRES_PASSWORD
└── Variables: PROJECT_NAME, AZURE_REGION, FORMAE_AGENT_URL

Azure (managed by Azure plugin, run by the pipeline)
├── Resource Group (infra-to-app-rg)
├── PostgreSQL Flexible Server (infra-to-app-pg)
├── Firewall Rule (AllowAzureServices)
└── Container Instance (miniflux-demo)
    └── Miniflux → DATABASE_URL → PostgreSQL
```
