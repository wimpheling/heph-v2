# heph-v2

- PR mode: Start each change on a new branch. Run appropriate checks, commit and push the branch, and open a pull request. Never commit or push directly to `main`. Never merge pull requests; the human handles merging.
- Human-only docs: Agents may read `docs/` for context, but must never create, modify, delete, rename, or automatically format files in that folder. Only humans may change `docs/`, manually.

Keep work incremental: this repository is a Git forge and agent runtime, currently at bootstrap stage. We don't need retrocompatibility : this is a POC so you can break APIs, rewrite migrations. There is no data worth anything in the app, at anytime. (We'll remove that if one day the app is used)

- Backend code uses the root Go module and the Go version declared in `go.mod`. Keep the executable entry point in `cmd/heph` and application packages in `internal/`.
- Define public APIs in `proto/`, run `pnpm generate`, and commit generated Go (`gen/`) and TypeScript (`packages/contracts/src/gen/`) bindings. Do not edit generated bindings directly.
- Keep application logic in `internal/core` independent of HTTP, ConnectRPC, and generated transport types. Translate at `internal/rpc`; compose HTTP services in `internal/server`.
- Centralize Go environment reads in `internal/config` (or `config.go` / `config/` packages) and pass typed configuration elsewhere.
- React application code imports UI from `@heph/design-system` and calls typed adapters in `apps/web/src/app/rpc`. Add DOM, CSS, Astryx imports, and theme changes inside the design-system package.
- Use Vite+ for frontend development, formatting, linting, and builds. Custom Oxlint rules live in `tools/oxlint`; add valid and invalid fixtures when changing rules.
- Go architecture rules live in `tools/go-lint` and use `golang.org/x/tools/go/analysis`. Add valid and invalid fixtures when changing rules. Format handwritten Go code with gofmt and keep dependencies tidy with `go mod tidy`.
- Run `pnpm check`, `pnpm test`, `pnpm test:e2e`, and `pnpm build` for changes spanning the bootstrap. `pnpm test` includes Go architecture-rule fixtures and the race detector; use `pnpm test:go-lints` for focused analyzer changes. Use checks appropriate to smaller edits.

## Use-case orchestration guideline

Keep core use-case orchestration restricted and readable as business policy. Domain models and services implement the predicates and computations that orchestration composes.

- Use-case code may call only explicitly approved domain services and declared port interfaces. An imported package or a type named `Service` is not automatically approved; future lint checks must resolve the actual called symbols and receiver types.
- Business branches consume named boolean results or boolean fields, optionally combined with `!`, `&&`, and `||`. Move raw value comparisons, arithmetic, string manipulation, and other computations into named domain operations. Prefer `CanStartAgent(...)` over comparing a status string in the use case.
- Use `if` / `else` for business branching. Put value-based switches, iteration, and algorithms behind approved domain operations or ports rather than implementing them in orchestration.
- Allow ordinary Go error handling as an explicit exception: `err == nil`, `err != nil`, error propagation, and `errors.Is` / `errors.As`. Do not replace idiomatic errors with success booleans to satisfy this guideline. Other helper calls require explicit approval in the call allowlist.
- Allow local bindings, field access, argument passing, and construction and return of typed core inputs and results. HTTP and ConnectRPC response construction stays in the RPC adapters.
- Side effects go through declared ports. Use cases must not directly access files, databases, the network, environment variables, or process execution.
- Keep meaningful predicates and computations in independently testable domain operations; avoid creating arbitrary helper services solely to bypass the restrictions.

The Go analyzer enforces these restrictions in `internal/core/domain/` and every subpackage. Operation factories only return `operation.Define` with an explicit policy and a private handler. Only approved authorization services, the `authorization.GrantReader` port, system metadata, and explicit operation/error helpers are call targets. Add valid and invalid fixtures when extending this allowlist. Domain services, transport adapters, generated code, and tests are outside the restricted handler rules.

## Platform authorization

- PostgreSQL is an opinionated core dependency. Keep SQL, migrations, and PostgreSQL types in `internal/core/postgres/`. Keep permission decisions in `internal/core/authorization`, independent of SQL and transport types.
- Replaceable drivers belong under `internal/drivers/bootstrap/` or `internal/drivers/std/`. The bootstrap identity driver is `internal/drivers/bootstrap/identity/`. User accounts and credentials stay outside core.
- Identify a caller with the verified `identity.ActorRef{Source, Subject}` pair. Pass it explicitly to protected operations; do not retrieve actor identity from context values, global state, or request claims.
- Define each RPC's comments and explicit authentication option in `proto/`. The custom generator rejects missing documentation and unknown policies and writes `api/endpoints.md`. Register services through `rpc/platform.Register`; bind each method exactly once to a domain operation with typed converters.
- Raw handlers are private and may only be referenced by `operation.Define`. RPC and internal callers execute guarded operations. Use the transaction service supplied to the handler so authorization and effects share the same database transaction.
- Grant assignments are immutable. Grant creation requires management rights and possession of the delegated action over the complete target scope. Revocation keeps the original record and its audit metadata. Bootstrap grants are seeded once; restarting must not restore revoked grants.
- Use Playwright in the dedicated `packages/e2e/` workspace for most behavior tests, including API-only tests against real PostgreSQL. Keep focused unit tests for policy invariants, guard failures, and analyzer fixtures. Keep the test harness separate from HTTP assertions so it can be replaced with another application runtime.
- Run `pnpm test:coverage` for unit, E2E, and combined Go application coverage. The reports must use the same source blocks and merge counters, not percentages. CI runs both suites and publishes coverage and Playwright artifacts.
