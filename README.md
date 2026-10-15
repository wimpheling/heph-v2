# heph-v2

A Git forge and agent runtime built in Rust, with ConnectRPC contracts and a React frontend. This first step establishes the workspaces, design system, tooling, and a live system-info RPC. Repository storage, authentication, persistence, and agent execution will follow.

## Setup

Prerequisites: Node 24, pnpm 12.11.2, Rust (selected by `rust-toolchain.toml`), and `protoc` on PATH. On Debian/Ubuntu, install `protobuf-compiler` for `protoc`.

```sh
npm install --global pnpm@12.11.2
pnpm install
pnpm generate
pnpm dev
```

Open http://127.0.0.1:5173. The page shows the actual connection to the Rust server on 127.0.0.1:8080. `pnpm dev` starts both processes and stops them together. Use `pnpm dev:web` and `pnpm dev:server` to start them separately.

The Vite dev proxy forwards `/rpc/*` to Rust, removing the `/rpc` prefix. Set `HEPH_LISTEN_ADDR` to change the Rust bind address and `HEPH_RPC_TARGET` to change the proxy target. These variables can be passed in the shell; the Rust server does not load `.env` files.

```sh
curl http://127.0.0.1:8080/heph.system.v1.SystemService/GetSystemInfo \
  -H 'content-type: application/json' \
  -H 'connect-protocol-version: 1' \
  -d '{}'
```

Production builds output the frontend to `apps/web/dist`. Production hosting must serve those assets and proxy `/rpc/*` to Rust with the same prefix rewrite; Vite's development proxy is not included in the build.

## Layout

| Path                      | Responsibility                                                     |
| ------------------------- | ------------------------------------------------------------------ |
| `proto/`                  | Versioned protobuf contracts, shared across languages              |
| `crates/core/`            | Application types and logic, independent of transport              |
| `crates/proto/`           | Rust protobuf and Connect service bindings generated at build time |
| `crates/server/`          | Configuration, RPC adapters, and server composition                |
| `apps/web/src/app/rpc/`   | Typed Connect browser clients and application adapters             |
| `apps/web/`               | React application, bootstrapped with Vite+                         |
| `packages/contracts/`     | Checked-in generated TypeScript contract bindings                  |
| `packages/design-system/` | Public UI components, Astryx integration, and styling              |
| `packages/e2e/`           | Playwright browser tests against React and Rust                    |
| `tools/oxlint/`           | Project architecture rules run through Vite+'s Oxlint              |
| `tools/dylint/`           | Project architecture rules compiled against a pinned nightly       |

The design-system boundary follows [Backbone's approach](https://github.com/wimpheling/backbone-template-v1): application components consume the local package. [Astryx](https://github.com/facebook/astryx) provides the underlying React components and neutral theme. The initial package exposes a small set of layout primitives and Astryx's Button; extend this public surface as features need it.

## Checks

Install the custom Rust lint tooling once:

```sh
rustup toolchain install nightly-2026-04-16 --profile minimal --component rustc-dev --component llvm-tools-preview
cargo install cargo-dylint dylint-link --version 6.0.0 --locked
```

```sh
pnpm check             # Formatting, Oxlint, TypeScript, Buf, Clippy, and custom Dylint
pnpm test              # Architecture-rule fixtures and HTTP-level Rust RPC tests
pnpm test:e2e          # Playwright tests on desktop and mobile Chromium
pnpm test:rust-lints   # Dylint diagnostic fixtures
pnpm build             # Frontend and Rust builds
pnpm fmt               # Format frontend and Rust workspace
```

The Rust application uses stable Rust. Dylint separately uses `nightly-2026-04-16`, because lint libraries depend on compiler internals. Keep the lint toolchain, Dylint version, diagnostic snapshots, and CI configuration synchronized when updating them.

Initial enforced boundaries:

- Application JSX uses design-system components. Raw DOM, Astryx imports, and application CSS imports belong in `packages/design-system`.
- Connect/protobuf imports and direct network calls in frontend application code belong in `app/rpc`.
- Rust reads process environment variables in `config.rs` or `config/` modules, then passes typed configuration elsewhere. The custom lint resolves definitions, including aliased imports.

The frontend rules check syntax and import paths; they are an initial boundary, not a complete dependency graph validator. Add rules and violation fixtures as architectural decisions become concrete.

After editing protobuf contracts, run `pnpm generate`. Rust bindings regenerate during Cargo builds. CI lints the schemas and verifies checked-in TypeScript bindings are current.

## Browser tests

Install Chromium once, then run the Playwright workspace package:

```sh
pnpm --filter @heph/e2e install:browsers
pnpm test:e2e
pnpm test:e2e:ui
```

On Linux, use `pnpm --filter @heph/e2e exec playwright install --with-deps chromium` if browser system dependencies are missing.

Playwright starts and stops its own Rust server and Vite+ frontend on ports 18080 and 15173. Keep these ports free; the tests do not reuse development servers. The suite checks the real ConnectRPC connection, retry behavior, pending and unavailable states, recovery, and layout width on desktop and mobile Chromium. Only the failure scenario intercepts requests; successful connections reach Rust.

Failures retain screenshots and traces in `packages/e2e/test-results/`. Open the HTML report with `pnpm --filter @heph/e2e exec playwright show-report`. CI installs Chromium, runs the suite, and uploads the report.
