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

This is an architecture guideline, not a rule currently enforced by the Go analyzer. When implementing enforcement, explicitly identify the use-case packages or functions, define the approved call targets and exceptions, and add valid and invalid fixtures. Do not apply these restrictions indiscriminately to domain services, adapters, generated code, or tests.
