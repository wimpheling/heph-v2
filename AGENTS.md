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
