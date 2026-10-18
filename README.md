# heph-v2

A Git forge and agent runtime built in Go, with ConnectRPC contracts and a React frontend. The platform provides PostgreSQL persistence, bootstrap authentication, guarded grant management RPCs, and enforced domain boundaries. Git hosting and agent execution are the next features.

## Setup

Prerequisites: Go 1.26 or later, Node 24, pnpm 12.11.2, and Docker Compose.

```sh
npm install --global pnpm@12.11.2
pnpm install
cp .env.example .env
# Edit .env and replace the bootstrap token with a random secret of at least 24 characters.
set -a
. ./.env
set +a
pnpm db:up
pnpm db:migrate
pnpm generate
pnpm dev
```

Open http://127.0.0.1:5173. The frontend calls the Go server at 127.0.0.1:8080 through Vite's `/rpc` proxy. Use `pnpm dev:web` and `pnpm dev:server` to start the processes separately. Go reads environment settings through `internal/config`; it does not load `.env` itself.

**NO HARDCODED DEFAULT VALUES FOR ENVIRONMENT VARIABLES.** Required settings must fail when absent or empty. Development and test values must be supplied explicitly through `.env.example`, your shell, or CI. Do not add fallback literals in Go, JavaScript, TypeScript, test harnesses, or Compose. Optional settings may be absent only when absence has defined behavior, such as disabling coverage output; they must not substitute another value. Configuration errors name the setting without printing credentials.

PostgreSQL 18.3 is pinned in `compose.yaml` and CI. Compose binds its port to loopback, waits for a health check, and keeps database files in a named volume. `pnpm db:down` stops the service without deleting the volume. `pnpm db:reset` removes the configured schema's Heph authorization tables and reruns migrations. This POC has no compatibility or data retention requirement.

The server checks database readiness, applies versioned migrations, and seeds bootstrap state during startup. SQL, migrations, and PostgreSQL types belong to `internal/core/postgres/`.

| Setting                     | Purpose                                                                         |
| --------------------------- | ------------------------------------------------------------------------------- |
| `HEPH_LISTEN_ADDR`          | Required server IP address and port.                                            |
| `HEPH_DATABASE_URL`         | Required PostgreSQL connection URL.                                             |
| `HEPH_DATABASE_SCHEMA`      | Required PostgreSQL schema name.                                                |
| `HEPH_BOOTSTRAP_IDENTITIES` | JSON array of `source`, `subject`, and `token` objects. Required to serve RPCs. |
| `HEPH_RPC_TARGET`           | Vite's backend proxy target.                                                    |

`migrate` and `reset` require only `HEPH_DATABASE_URL` and `HEPH_DATABASE_SCHEMA`. Server startup additionally requires `HEPH_LISTEN_ADDR` and `HEPH_BOOTSTRAP_IDENTITIES`. Vite development requires `HEPH_RPC_TARGET`; a production frontend build does not contact a backend. Compose requires all four `POSTGRES_*` settings in `.env.example`.

The first configured identity receives explicit platform grants for each registered action, once. Further configured identities receive their own personal scope, without platform grants. The driver verifies tokens using fixed-size hashes and constant-time comparisons. Restarting never restores revoked bootstrap grants. Replacing a token keeps the same actor identity; changing the initial bootstrap actor requires resetting this POC's authorization state.

## Grant RPCs

The service exposes `CreateGrant`, `GetGrant`, `ListGrants`, and `RevokeGrant`. Grant fields are immutable. To change an assignment, revoke it and create another one. Revocation records the verified caller and timestamp; authorized callers can still inspect the original record. Repeated revocation returns the original revocation.

Every protected request uses a bearer token. The server controls grant IDs, issuers, and timestamps. Actor fields in request bodies never authenticate the caller.

```sh
export HEPH_TOKEN='the-token-from-your-local-config'

# List active grants directly attached to the platform scope.
curl http://127.0.0.1:8080/heph.authorization.v1.AuthorizationService/ListGrants \
  -H "Authorization: Bearer $HEPH_TOKEN" -H 'content-type: application/json' \
  -d '{"scope":{"kind":"SCOPE_KIND_PLATFORM","id":"platform"},"pageSize":50}'

# Assign read access to another actor reference.
curl http://127.0.0.1:8080/heph.authorization.v1.AuthorizationService/CreateGrant \
  -H "Authorization: Bearer $HEPH_TOKEN" -H 'content-type: application/json' \
  -d '{"grantee":{"source":"bootstrap","subject":"agent"},"action":"authorization.grants.read","scope":{"kind":"SCOPE_KIND_PLATFORM","id":"platform"}}'

export GRANT_ID='the-id-returned-by-CreateGrant'
curl http://127.0.0.1:8080/heph.authorization.v1.AuthorizationService/GetGrant \
  -H "Authorization: Bearer $HEPH_TOKEN" -H 'content-type: application/json' \
  -d "{\"id\":\"$GRANT_ID\"}"
curl http://127.0.0.1:8080/heph.authorization.v1.AuthorizationService/RevokeGrant \
  -H "Authorization: Bearer $HEPH_TOKEN" -H 'content-type: application/json' \
  -d "{\"id\":\"$GRANT_ID\"}"
```

The grantee reference identifies a human or agent independently of account storage. To authenticate that actor with the bootstrap driver, add its own distinct credential to the configured identity array. External identity providers can implement the same authentication interface later.

| Registered action             | Meaning                                                          |
| ----------------------------- | ---------------------------------------------------------------- |
| `authorization.grants.read`   | Inspect grants within a scope.                                   |
| `authorization.grants.create` | Issue grants within a scope, bounded by the caller's own rights. |
| `authorization.grants.revoke` | Revoke grants within a scope.                                    |
| `git.ref.update`              | Ordinary Git ref updates.                                        |
| `git.ref.create`              | Create a Git ref.                                                |
| `git.ref.delete`              | Delete a Git ref.                                                |
| `git.ref.force_update`        | Force a Git ref update.                                          |
| `repository.settings.update`  | Change repository settings.                                      |

Git actions have authorization fixtures; actual Git mutation endpoints come later. Exact action names match without wildcards. Creation and forced updates are separate permissions from ordinary ref updates.

Scopes form a tree: platform → personal or organization → project → resource → Git ref. Resources can also belong directly to personal or organization scopes. Personal scopes link to complete source-qualified actor references. Organization membership and agent creator relationships do not change scope containment.

Grants cover the named scope and its descendants for the exact action. A grant for one branch cannot authorize another branch, repository settings, or the parent repository. A narrower grant does not remove broader access. Branch protection and branch-pattern selectors are separate future policies.

Creating a grant requires both `authorization.grants.create` and the action being delegated across the entire proposed scope. Revoking requires `authorization.grants.revoke` on the stored grant scope. The original issuer has no automatic exemption. Revocation removes that assignment; independently issued grants remain independent.

The platform resolves trusted scope data and checks permissions before calling private business handlers. RPC and internal callers use the same guarded operations with an explicit actor argument. Grant authorization and effects run in one PostgreSQL transaction. This POC serializes those transactions with an advisory lock to prevent permission changes racing with mutations.

`GetSystemInfo` is explicitly public:

```sh
curl http://127.0.0.1:8080/heph.system.v1.SystemService/GetSystemInfo \
  -H 'content-type: application/json' -d '{}'
```

## Layout

| Path                                   | Responsibility                                                        |
| -------------------------------------- | --------------------------------------------------------------------- |
| `cmd/heph/`                            | Server, migration, and reset commands.                                |
| `internal/core/domain/`                | Restricted handlers and explicit guarded operation definitions.       |
| `internal/core/authorization/`         | Permission rules, grant models, scope resolution, and ports.          |
| `internal/core/identity/`              | Opaque actor references and authentication interface.                 |
| `internal/core/operation/`             | Mandatory operation guards and transaction orchestration.             |
| `internal/core/postgres/`              | Opinionated PostgreSQL persistence, SQL, and migrations.              |
| `internal/drivers/bootstrap/identity/` | Configured bootstrap identity source.                                 |
| `internal/drivers/std/`                | Reserved path for standard implementations of replaceable components. |
| `internal/config/`                     | Typed environment configuration.                                      |
| `internal/rpc/`                        | Typed Connect adapters; `platform/` validates bindings and metadata.  |
| `internal/server/`                     | HTTP service composition and graceful shutdown.                       |
| `proto/`, `gen/`                       | Public contracts and generated Go bindings.                           |
| `api/endpoints.md`                     | Generated endpoint reference.                                         |
| `apps/web/src/app/rpc/`                | Typed frontend adapters.                                              |
| `packages/contracts/`                  | Generated TypeScript bindings.                                        |
| `packages/design-system/`              | Public UI components, Astryx, and styling.                            |
| `packages/e2e/`                        | Independent Playwright workspace with API and browser projects.       |
| `tools/go-lint/`                       | Type-aware Go architecture analyzer and fixtures.                     |
| `tools/protoc-gen-heph/`               | Contract documentation checks and endpoint reference generator.       |
| `tools/oxlint/`                        | Frontend architecture checks.                                         |

The design-system boundary follows [Backbone's approach](https://github.com/wimpheling/backbone-template-v1). [Astryx](https://github.com/facebook/astryx) supplies the underlying React components and neutral theme. Application UI imports `@heph/design-system`; DOM, CSS, and theme changes stay within that package.

## Contracts and enforced boundaries

Edit APIs in `proto/`, run `pnpm generate`, and commit Go bindings, TypeScript bindings, and the generated endpoint reference. Generators are pinned through the workspace and Go module; no global `protoc` installation is required.

The custom protobuf plugin rejects undocumented endpoints, messages, and fields, and missing or unknown authentication options. The Go analyzer requires one typed binding per RPC, operation factories with explicit policies, private handlers, and platform service registration. Runtime startup also verifies complete bindings, descriptor compatibility, and agreement between authentication metadata and operation policies.

Inside `internal/core/domain/` and every subpackage, the analyzer permits only explicitly approved services, ports, operation constructors, and error helpers. Branches use named booleans with `!`, `&&`, and `||`. Raw comparisons, arithmetic, loops, string calculations, switches, function aliases, anonymous functions, and direct infrastructure calls are prohibited. Go error propagation, comparisons with `nil`, `errors.Is`, and `errors.As` remain permitted.

Core cannot import HTTP, ConnectRPC, or generated transport types. Environment reads belong in configuration modules. Frontend application network calls and contract imports belong in typed RPC adapters. The analyzer checks structure; tests prove the policy decisions.

The Go analyzer tracks environment strings through local helpers and rejects fallback assignments and successful fallback returns. Oxlint rejects direct environment fallback operators, assignments, conditional expressions, and destructuring defaults. These checks catch common patterns; configuration loaders must still validate every required input. Add valid and invalid fixtures when extending either rule.

Files under `docs/` remain human-maintained and excluded from agent edits and formatting.

## Tests and coverage

Most behavior tests run through Playwright against the real server and PostgreSQL. Focused unit tests cover guard failures, policy invariants, malformed metadata, and analyzer fixtures.

```sh
pnpm check
pnpm test
pnpm test:go-lints
pnpm --filter @heph/e2e install:browsers
pnpm test:e2e
pnpm test:coverage
pnpm build
```

On Linux, install browser dependencies with `pnpm --filter @heph/e2e exec playwright install --with-deps chromium`.

The dedicated `packages/e2e` project owns its dependencies, Playwright configuration, process harness, and HTTP assertions. Its `api` project runs without a browser. Chromium and mobile Chromium also check the existing frontend connection and failure recovery. API success and denial scenarios use real HTTP requests, without replacing the RPC implementation.

The harness requires the explicit `HEPH_E2E_DATABASE_URL`, `HEPH_E2E_BASE_URL`, `HEPH_E2E_WEB_URL`, `HEPH_E2E_API_ONLY`, and `HEPH_E2E_MANAGED` settings shown in `.env.example`. Export them before running tests. Both flags must be `0` or `1`: API-only skips browsers; managed mode starts and stops the test services. Set managed mode to `0` to test an independently managed installation. The harness allocates a unique schema per run and supplies synthetic test credentials. The example settings start the API on 18080 and Vite on 15173; persistence verification uses a second instance on 18081. Test schemas are dropped during cleanup. Application-specific startup and fixtures live in `harness/`, separate from the reusable API assertions.

```sh
# API-only E2E tests need PostgreSQL but no browser installation.
HEPH_E2E_API_ONLY=1 pnpm test:e2e

# A separately mapped PostgreSQL port can be used for the isolated test installation.
HEPH_E2E_DATABASE_URL='postgresql://heph:heph-local-placeholder@127.0.0.1:15432/heph?sslmode=disable' pnpm test:coverage

pnpm test:coverage:unit
pnpm test:coverage:e2e
pnpm coverage:merge
```

Coverage measures handwritten Go application statements in `cmd/heph` and `internal/`. Generated bindings, dependency code, and development tools are excluded from that denominator. E2E coverage comes from the instrumented real server, including graceful shutdown. Reports fail if the main server does not finish writing its counters.

Each of `coverage/unit/`, `coverage/e2e/`, and `coverage/combined/` contains `coverage.out`, `summary.json`, and `index.html`. Combined coverage merges statement counters over the same source blocks, rather than averaging percentages. `coverage/summary.md` and `coverage/summary.json` compare the three suites. Rerun both suites after source changes before merging their reports.

CI provisions the same PostgreSQL version, checks generated outputs, runs checks, unit tests, Playwright, and builds, and publishes the Playwright report and all coverage reports. Failures retain Playwright traces and screenshots. View them with `pnpm --filter @heph/e2e exec playwright show-report`.

`pnpm build` outputs `bin/heph` and `apps/web/dist`. Production hosting must serve the assets and proxy `/rpc/*` to Go, removing the prefix. Vite's development proxy is not part of the production build.
