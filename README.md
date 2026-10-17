# heph-v2

A Git forge and agent runtime built in Go, with ConnectRPC contracts and a React frontend. The bootstrap establishes the Go module, frontend workspace, design system, architecture checks, and a live system-info RPC. Repository storage, authentication, persistence, and agent execution will follow.

## Setup

Prerequisites: Go 1.26 or later, Node 24, and pnpm 12.11.2.

```sh
npm install --global pnpm@12.11.2
pnpm install
pnpm generate
pnpm dev
```

Open http://127.0.0.1:5173. The page shows the actual connection to the Go server on 127.0.0.1:8080. `pnpm dev` starts both processes and stops them together. Use `pnpm dev:web` and `pnpm dev:server` to start them separately.

The Vite dev proxy forwards `/rpc/*` to Go, removing the `/rpc` prefix. Set `HEPH_LISTEN_ADDR` to change the server bind address and `HEPH_RPC_TARGET` to change the proxy target. Pass these variables in the shell; the server does not load `.env` files. Invalid bind addresses fail at startup. The server handles SIGINT and SIGTERM with a graceful shutdown.

```sh
curl http://127.0.0.1:8080/heph.system.v1.SystemService/GetSystemInfo \
  -H 'content-type: application/json' \
  -H 'connect-protocol-version: 1' \
  -d '{}'
```

`pnpm build` outputs the server binary to `bin/heph` and the frontend to `apps/web/dist`. Production hosting must serve those assets and proxy `/rpc/*` to the Go server with the same prefix rewrite; Vite's development proxy is not included in the build.

## Layout

| Path                      | Responsibility                                         |
| ------------------------- | ------------------------------------------------------ |
| `cmd/heph/`               | Server entry point and process lifecycle               |
| `internal/core/`          | Application types and logic, independent of transport  |
| `internal/config/`        | Typed configuration and environment loading            |
| `internal/rpc/`           | ConnectRPC adapters for application logic              |
| `internal/server/`        | HTTP service composition and shutdown                  |
| `proto/`                  | Versioned protobuf contracts, shared across languages  |
| `gen/`                    | Checked-in generated Go protobuf and Connect bindings  |
| `apps/web/src/app/rpc/`   | Typed Connect browser clients and application adapters |
| `apps/web/`               | React application, bootstrapped with Vite+             |
| `packages/contracts/`     | Checked-in generated TypeScript contract bindings      |
| `packages/design-system/` | Public UI components, Astryx integration, and styling  |
| `packages/e2e/`           | Playwright browser tests against React and Go          |
| `tools/oxlint/`           | Frontend architecture rules run through Vite+'s Oxlint |
| `tools/go-lint/`          | Go architecture analyzer and valid/invalid fixtures    |

The design-system boundary follows [Backbone's approach](https://github.com/wimpheling/backbone-template-v1): application components consume the local package. [Astryx](https://github.com/facebook/astryx) provides the underlying React components and neutral theme. Extend the public design-system surface as features need it.

## Contracts and tooling

After editing protobuf contracts, run `pnpm generate` and commit both Go and TypeScript bindings. Buf generates both languages from `proto/`. The Go generators are pinned as tool dependencies in `go.mod`; Buf invokes them through `go tool`. No global generator installation or standalone `protoc` is required. CI verifies generated bindings are current.

The backend uses one root Go module. `go.mod` and `go.sum` track backend, analyzer, and generator dependencies. Use `go mod tidy` after changing dependencies.

## Checks

```sh
pnpm check          # Formatting, Oxlint, TypeScript, Buf, go vet, and custom Go analyzer
pnpm test           # Oxlint fixtures and all Go tests with the race detector
pnpm test:e2e       # Playwright tests on desktop and mobile Chromium
pnpm test:go-lints  # Go architecture-rule fixtures only
pnpm build          # Frontend and Go server builds
pnpm fmt            # Format frontend files and handwritten Go code
```

Initial enforced boundaries:

- Application JSX uses design-system components. Raw DOM, Astryx imports, and application CSS imports belong in `packages/design-system`.
- Connect/protobuf imports and direct network calls in frontend application code belong in `app/rpc`.
- Go environment reads (`os.Getenv`, `os.LookupEnv`, `os.Environ`, and `os.ExpandEnv`) belong in `config.go` or `config/` packages. Pass typed configuration to other packages. The analyzer resolves function identities, including aliases and dot imports.
- `internal/core` and its subpackages cannot import `net/http`, ConnectRPC, or this module's generated transport types. Translate at `internal/rpc`.

The Go analyzer uses `golang.org/x/tools/go/analysis`; add valid and invalid fixtures when changing its rules. The frontend rules check syntax and import paths, rather than a complete dependency graph. Extend enforcement as architectural decisions become concrete.

Files in `docs/` are maintained manually by humans. Agents may read them for context but may not change them. Automatic formatting excludes that folder.

## Browser tests

Install Chromium once, then run the Playwright workspace package:

```sh
pnpm --filter @heph/e2e install:browsers
pnpm test:e2e
pnpm test:e2e:ui
```

On Linux, use `pnpm --filter @heph/e2e exec playwright install --with-deps chromium` if browser system dependencies are missing.

Playwright starts and stops its own Go server and Vite+ frontend on ports 18080 and 15173. Keep these ports free; the tests do not reuse development servers. The suite checks the real ConnectRPC connection, retry behavior, pending and unavailable states, recovery, and layout width on desktop and mobile Chromium. Only the failure scenario intercepts requests; successful connections reach Go.

Failures retain screenshots and traces in `packages/e2e/test-results/`. Open the HTML report with `pnpm --filter @heph/e2e exec playwright show-report`. CI installs Chromium, runs the suite, and uploads the report.
