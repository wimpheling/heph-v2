# heph-v2

/!\ Do not edit docs folder. This is only manual.

Keep work incremental: this repository is a Git forge and agent runtime, currently at bootstrap stage. We don't need retrocompatibility : this is a POC so you can break APIs, rewrite migrations. There is no data worth anything in the app, at anytime. (We'll remove that if one day the app is used)

- Rust application code uses the root stable toolchain. Custom Dylint code uses the toolchain in `tools/dylint`.
- Define public APIs in `proto/`, run `pnpm generate`, and commit generated TypeScript bindings. Do not edit generated bindings directly.
- Keep application logic in `crates/core` independent of Axum, ConnectRPC, and generated transport types. Translate at `crates/server/src/rpc`.
- Centralize Rust environment reads in `config.rs` or `config/` modules.
- React application code imports UI from `@heph/design-system` and calls typed adapters in `apps/web/src/app/rpc`. Add DOM, CSS, Astryx imports, and theme changes inside the design-system package.
- Use Vite+ for frontend development, formatting, linting, and builds. Custom Oxlint rules live in `tools/oxlint`; add valid and invalid fixtures when changing rules.
- Run `pnpm check`, `pnpm test`, `pnpm test:rust-lints`, and `pnpm build` for changes spanning the bootstrap. Use checks appropriate to smaller edits.
