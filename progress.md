# formae-plugin-gha — Development Progress

## Plugin

15 resource types managing GitHub Actions CI/CD infrastructure:

### Wave 1 — Core

| Resource | Description |
|---|---|
| `GHA::Repo::Variable` | Repository variable |
| `GHA::Repo::Secret` | Repository secret (sealed-box encrypted, SHA-256 drift detection) |
| `GHA::Repo::Environment` | Deployment environment with protection rules + Resolvable references |
| `GHA::Repo::File` | Repository file via Contents API |
| `GHA::Repo::Workflow` | Structured workflow (typed via com.github.actions, YAML↔JSON round-trip) |
| `GHA::Environment::Variable` | Environment-scoped variable |
| `GHA::Environment::Secret` | Environment-scoped secret |
| `GHA::Environment::BranchPolicy` | Deployment branch policy |
| `GHA::Org::Variable` | Organization variable with visibility |
| `GHA::Org::Secret` | Organization secret with visibility |

### Wave 2 — Advanced

| Resource | Description |
|---|---|
| `GHA::Org::RunnerGroup` | Organization self-hosted runner group (full CRUD, server-assigned ID) |
| `GHA::Repo::OIDCClaims` | Repository OIDC subject claim customization (singleton) |
| `GHA::Org::OIDCClaims` | Organization OIDC subject claim customization (singleton) |
| `GHA::Org::Permissions` | Organization Actions permissions (singleton) |
| `GHA::Org::DefaultWorkflowPermissions` | Organization default workflow token permissions (singleton) |

## Authentication

Token resolved at runtime via chain (no agent restart needed):
1. `GITHUB_TOKEN` environment variable
2. `gh auth token` CLI command
3. `~/.config/gh/hosts.yml` config file

## Example

`examples/ci-pipeline/` — bootstraps a complete CI/CD pipeline for deploying
AWS infrastructure. Uses `GHA::Repo::Workflow` for structured workflow files
(typed via `com.github.actions` from pkl-pantry), staging and production
environments with approval gates, AWS OIDC secrets, deployment variables,
and branch policies. 16 resources from one forma.

## Tests

- 12 integration tests against real GitHub API (all repo/env pass, org skips on 403)
- 1 unit test for `stripEmpty` (8 cases: nil, empty map, nested, scalar, array, mixed)
- 24-step conformance test — all steps pass, all properties match at every stage
  (create, extract, sync idempotency, update, replace, destroy, OOB delete detection).
  Exit code is FAIL due to an Ergo node teardown error in the conformance harness
  during post-test cleanup (`meta.MessagePortTerminate`). All plugin assertions pass.
  This is a formae core issue, not a plugin bug.
- End-to-end ci-pipeline example: apply + destroy verified clean

**Org tests blocked**: Runner groups, org OIDC, org permissions, and org workflow permissions
all return 403. The `admin:org` OAuth scope is granted, but the test account (`browdues`) is
not an org admin on `platform-engineering-labs`. Org admin role required to unblock.

## Test Repo

`platform-engineering-labs/formae-plugin-gha-test` (internal)

## Bugs Found & Fixed

1. **Variable name case normalization**: GitHub uppercases variable/secret names.
   Fix: normalize to uppercase in Create.

2. **Env secret API uses integer repo IDs**: Fix: `getRepoID()` helper.

3. **NativeID included owner/repo**: Fix: repo-scoped = name, env-scoped = env/name,
   singleton = fixed string.

4. **Stale environment on destroy**: Fix: Resolvable references (`env.res.name`)
   for correct dependency ordering.

5. **`workflow` OAuth scope required**: For files under `.github/workflows/`.

6. **Agent env var coupling**: Plugin read `GITHUB_TOKEN` from `os.Getenv` inside the
   agent process. If the agent started without it, all operations failed. Fix: auth
   chain resolves token at runtime from env → gh CLI → gh config file.

7. **Empty fields in workflow YAML**: Pkl produces `env: {}` for unset optional fields.
   If GitHub strips empty maps, sync would detect false drift. Fix: `stripEmpty()`
   removes nil values and empty maps before marshaling to YAML.

## infra-to-app Example

`examples/infra-to-app/` — end-to-end demo deploying Miniflux RSS reader on Azure.
Two plugins: GHA manages the CI/CD pipeline, Azure manages infrastructure.

### Fixes Applied (2026-04-07)

1. **Container image architecture**: ACR image was `arm64`, ACI requires `amd64`.
   Pulled official `miniflux/miniflux:2.2.19` for `linux/amd64` and pushed to
   `formaedemoacr.azurecr.io/miniflux:2.2.19`.

2. **`LISTEN_ADDR` env var**: Miniflux defaults to `127.0.0.1:8080` — unreachable
   from outside the container. Added `LISTEN_ADDR=0.0.0.0:8080` to `main.pkl`.

3. **DNS name label**: Container had no FQDN. Added `--dns-name-label` to
   `az container create` in `main.pkl`. URL: `http://infra-to-app.westus.azurecontainer.io:8080`.

4. **Firewall rule**: `AllowAzureServices` (0.0.0.0/0.0.0.0) does not cover ACI
   outbound IPs. Replaced with `AllowAll` (0.0.0.0/255.255.255.255) in `infra/database.pkl`.

5. **App URL output**: Changed from `ipAddress.ip` to `ipAddress.fqdn` in the
   "Get app URL" step so downstream jobs use the stable DNS name.

### Verified Working

- Miniflux login page at FQDN
- `/healthcheck` endpoint returns OK
- API auth with `admin:demo1234`
- All 5 RSS feeds added via API
- 100 entries loaded after refresh
- Full verify pipeline logic tested manually against live instance

### Remaining

- No workflow step to ensure ACR image is `amd64` — relies on image being pre-pushed
- `AllowAll` firewall rule is broad — acceptable for demo, not production

## Known Nits

- **YAML key ordering**: Go's `yaml.Marshal` on `map[string]interface{}` sorts keys
  alphabetically. Workflow YAML shows `jobs` before `name` before `on` instead of the
  conventional `name`, `on`, `jobs` order. Cosmetic only — GitHub doesn't care about
  key order, and formae compares structured JSON, not YAML text.

## GHA::Repo::File vs GHA::Repo::Workflow

`File` treats content as an opaque string — sync is blob-level, extract produces a
string literal. `Workflow` stores structured properties — sync is field-level, extract
produces typed Pkl. Use `Workflow` for `.github/workflows/*.yml`. Use `File` for other
repo-managed files at scale:

| File | Use case |
|---|---|
| `.github/CODEOWNERS` | Code ownership rules across an org |
| `.github/dependabot.yml` | Dependency update configuration |
| `.github/pull_request_template.md` | Uniform PR templates |
| `LICENSE` | License compliance |
| `SECURITY.md` | Security policy enforcement |
| `renovate.json` | Renovate bot config |

## Lessons for GitLab Plugin

1. **NativeID = provider identity, not target info.** Owner/repo comes from target config.
2. **Resolvable references for parent-child dependencies.** Without these, destroy ordering breaks.
3. **Name normalization.** Test early for provider-specific normalization.
4. **Singleton resources** — Create = Set, Delete = reset to defaults, NativeID = fixed string.
5. **Auth chain** — resolve credentials at runtime, not at agent startup. Avoid storing in target config/DB.
6. **Compose with pkl-pantry** — if a typed package exists for the provider's domain model, use it.
7. **Structured resource for CI config** — blob resources break bidirectional sync. Parse on Read.
8. **Go side is thin** — Pkl schema defines the type system. Go just converts wire formats.
9. **`identifier` convention** — use property name (`"name"`, `"id"`), not JSONPath (`"$.name"`). Match other plugins.

## Token Requirements

| Scope | Required for |
|---|---|
| `repo` | All repo-scoped resources |
| `workflow` | Files under `.github/workflows/` |
| `admin:org` | Organization-scoped resources |
