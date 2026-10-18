# Platform authorization

Status: Plan for review. The feature is not complete.

Keep the plan and the feature implementation in the same pull request.

## Goal

Require each RPC endpoint to have a documented contract and a domain handler.
Identify each caller.
Check access before each protected action.
Store actor references and permission data in PostgreSQL.
Keep user account management in identity drivers, outside core.

Build the platform before the builder agent.
The builder agent must use the same platform as other Heph agents.

## Current state

- The Go server supports Connect, gRPC, and gRPC-Web.
- Protobuf files define the API contracts.
- The build tools generate Go and TypeScript bindings.
- The system service has one endpoint: `GetSystemInfo`.
- The Go analyzer checks environment reads and selected imports in core packages.
- The server does not authenticate callers or check resource access.
- The analyzer does not require domain handlers or enforce the domain restrictions.

## Terms

| Term               | Meaning                                                                        |
| ------------------ | ------------------------------------------------------------------------------ |
| Actor              | The human or agent that makes a request.                                       |
| Actor reference    | The pair of `Source` and `Subject` that identifies an actor.                   |
| Source             | The stable namespace of a trusted identity source.                             |
| Subject            | The stable actor identifier within that source.                                |
| Authentication     | The check that identifies the actor from a credential.                         |
| Authorization      | The check that permits or refuses an action on a resource.                     |
| Scope              | An ownership boundary, such as a user, organization, or project.               |
| Grant              | A record that permits an actor to do an action in a scope.                     |
| Domain operation   | The guarded entry point that checks access before it calls a business handler. |
| Domain handler     | The private function that runs the business logic after the access check.      |
| Port               | A Go interface through which the domain requests an external operation.        |
| Driver             | The implementation of a port for a selected system.                            |
| Endpoint binding   | The typed connection between an RPC method and its guarded domain operation.   |
| Database migration | A versioned change to the database schema.                                     |

## Proposed request path

```text
RPC endpoint
  -> authentication
  -> RPC adapter
  -> guarded domain operation
  -> resolve resource or scope
  -> authorization check
  -> private business handler
  -> approved services and ports
```

The RPC adapter translates requests, results, and errors.
Each domain operation declares an authorization policy and a private business handler.
The policy identifies the required action and resource or scope.
The platform loads trusted resource data and checks access before it calls the business handler.
Public operations require an explicit public policy.
The platform refuses operations with missing or unknown policies.
If the access check fails, the platform does not call the business handler.
Do not return resource data before the access check permits access.

RPC callers and internal callers must use the same guarded domain operations.

## Decisions to confirm

Confirm these choices before implementation.
Review one decision at a time.
After approval, mark the decision complete and explain the next decision.

- [x] Use `internal/core/domain/` and all its subpackages for the restricted domain handlers.
- [x] Use PostgreSQL for persistent identity and permission data.
- [x] Add a project `compose.yaml` file for local PostgreSQL.
- [x] Approve `internal/core/authorization` and the `authorization.GrantReader` interface as domain call targets.
- [x] Bind each RPC to a guarded domain operation with typed Go references and input and output conversion functions.
- [x] Use a protobuf method option with `AUTHENTICATION_PUBLIC`, `AUTHENTICATION_REQUIRED`, and `AUTHENTICATION_UNSPECIFIED` values.
- [x] Require an authorization guard before each private business handler, including operations called without RPC.
- [x] Identify humans and agents with stable `ActorRef` values containing `Source` and `Subject`.
- [x] Keep user account management in identity drivers, outside core.
- [ ] Treat organizations as ownership scopes.
- [ ] Pass the actor explicitly to domain operations.
- [ ] Keep session IDs, credential IDs, and run IDs separate from actor IDs.
- [ ] Keep the permission engine in core.
- [ ] Supply identity and permission data through injected interfaces.
- [ ] Start with one configured bootstrap user and a bootstrap credential.
- [ ] Define ownership rights and explicit grants for the first access checks.
- [ ] Confirm the Go error exceptions and the domain rules below.

## Confirmed actor reference

```go
type ActorRef struct {
    Source  string
    Subject string
}
```

The authentication driver supplies both fields after it verifies the credential.
The pair identifies the actor.
Different sources can use the same subject value.
Each source must keep its subject values stable and must not assign them to another actor.
Treat both fields as opaque, case-sensitive values.
Core does not require a UUID format.
Do not use a display name or email address as the actor reference.

Core uses actor references for ownership and grants.
Identity drivers manage accounts, profiles, login, and credentials.
An identity driver can use local storage or an external identity provider.
PostgreSQL storage does not require core to manage user accounts.

## Work steps

### 1. Set up PostgreSQL

- [ ] Add a PostgreSQL service to the root `compose.yaml` file.
- [ ] Pin the PostgreSQL version.
- [ ] Bind the local database port to `127.0.0.1`.
- [ ] Add a named volume to keep data across container restarts.
- [ ] Add a database health check.
- [ ] Wait for database readiness before migrations and application startup.
- [ ] Add `.env.example` with local database settings and placeholder credentials.
- [ ] Read the application database connection settings through `internal/config`.
- [ ] Create the database connection pool during platform startup.
- [ ] Close the database connection pool during shutdown.
- [ ] Add versioned migrations for actor references and permission data.
- [ ] Keep any local account storage inside the identity driver.
- [ ] Keep SQL and PostgreSQL types inside the database drivers.
- [ ] Supply the database drivers through the core port interfaces.
- [ ] Use PostgreSQL for the first bootstrap storage drivers.
- [ ] Use test doubles for isolated unit tests.
- [ ] Add commands to start, stop, migrate, and reset the local database.
- [ ] Document how the Go application receives the database connection settings.
- [ ] Use the same PostgreSQL version for local development and CI.
- [ ] Prepare a separate database or schema for each integration test run.
- [ ] Prepare the database for integration tests and browser tests.

### 2. Define and document the endpoint contract

- [ ] Add comments for each RPC method and its request and response fields.
- [ ] Describe the purpose, inputs, outputs, and expected errors.
- [ ] Require an explicit access policy for each RPC method.
- [ ] Declare `GetSystemInfo` as public.
- [ ] Refuse an access policy that is missing or unknown.
- [ ] Generate an endpoint reference from the protobuf contract.
- [ ] Put the generated reference outside `docs/`.

### 3. Connect each endpoint to its domain handler

- [ ] Add a typed endpoint binding.
- [ ] Bind each RPC adapter to a guarded domain operation instead of its raw business handler.
- [ ] Use actual Go function references instead of handler names stored as strings.
- [ ] Add typed conversion functions for the RPC request and the domain result.
- [ ] Require exactly one binding for each RPC method.
- [ ] Use Go type checks to verify compatible binding types.
- [ ] Give the system operation an explicit public policy and a private domain handler.
- [ ] Limit RPC adapters to authentication, data translation, delegation, and error translation.
- [ ] Require endpoint registration through the platform.
- [ ] Use only core types in the domain.

### 4. Identify the caller

- [ ] Define the actor type and the authentication interface.
- [ ] Use the verified `Source` and `Subject` pair as the actor reference.
- [ ] Keep account management outside core.
- [ ] Verify the credential before each protected RPC method.
- [ ] Return an actor only after successful authentication.
- [ ] Pass the verified actor from the RPC adapter to the guarded domain operation.
- [ ] Refuse caller-supplied actor IDs as proof of identity.
- [ ] Add the bootstrap identity driver.
- [ ] Read bootstrap configuration through `internal/config`.

### 5. Check access in core

- [ ] Define the action, resource, scope, and grant types.
- [ ] Require a policy and a private business handler for each domain operation.
- [ ] Require a resource resolver when the policy needs resource data.
- [ ] Resolve resource data through trusted services instead of trusting caller-supplied ownership fields.
- [ ] Run the access check before the business handler.
- [ ] Reject operations with missing or unknown policies during platform registration.
- [ ] Require an explicit policy for public operations.
- [ ] Use the parent scope for an action that creates a resource.
- [ ] Refuse access unless a defined rule permits the action.
- [ ] Refuse unknown actions and unknown scopes.
- [ ] Refuse the action if the permission check fails.
- [ ] Apply the same checks to human and agent actors.
- [ ] Keep permission checks inside the domain operation.
- [ ] Map authentication and permission errors at the RPC boundary.
- [ ] Check delegated rights before a future operation issues an agent credential.

### 6. Enforce the rules with the custom Go analyzer

Use resolved Go symbols and receiver types.
Do not approve a call because its type name contains `Service`.

- [ ] Require endpoint comments and explicit access policies.
- [ ] Find missing, repeated, or invalid endpoint bindings.
- [ ] Require each operation's business handler to belong to `internal/core/domain/` or one of its subpackages.
- [ ] Require an explicit authorization policy for each domain operation.
- [ ] Find RPC adapters that bypass the guarded domain operation.
- [ ] Refuse raw handler references outside the approved operation definitions.
- [ ] Refuse direct calls to raw business handlers in production code.
- [ ] Find endpoint registrations that bypass platform authentication.
- [ ] Permit domain calls only to approved services, ports, and explicit helpers.
- [ ] Permit boolean decisions with `!`, `&&`, and `||`.
- [ ] Refuse raw comparisons, arithmetic, string calculations, and similar computations in domain handlers.
- [ ] Find prohibited computations outside branch conditions as well as inside them.
- [ ] Permit `if` and `else` for business branches.
- [ ] Refuse loops, switches, goroutines, and channel operations in domain handlers.
- [ ] Permit local bindings, field reads, argument passing, and typed result construction.
- [ ] Permit error returns, `err == nil`, `err != nil`, `errors.Is`, and `errors.As`.
- [ ] Refuse arbitrary function variables and anonymous function calls in domain handlers.
- [ ] Refuse direct file, database, network, environment, and process operations in domain handlers.
- [ ] Exclude tests and generated code from the domain restrictions.
- [ ] Apply the domain restrictions to `internal/core/domain/` and all its subpackages.
- [ ] Keep domain services and adapters outside the restricted domain package.
- [ ] Add valid and invalid fixtures for each rule.
- [ ] Add fixtures for import aliases and attempted bypasses.
- [ ] Add the checks to `pnpm check` and CI.

The analyzer checks code structure.
Access tests must also check the permission decisions.

### 7. Verify the complete request path

- [ ] Verify that the Compose service becomes healthy.
- [ ] Verify that migrations prepare an empty database.
- [ ] Verify that actor references and permission data survive application and database restarts.
- [ ] Verify that database errors prevent protected actions.
- [ ] Verify that the public system endpoint works without a credential.
- [ ] Verify that a protected endpoint refuses missing and invalid credentials.
- [ ] Verify that a valid credential produces the correct actor.
- [ ] Verify that an allowed actor can complete a protected action.
- [ ] Verify that a denied actor cannot complete the action.
- [ ] Verify that a permission error prevents the protected action.
- [ ] Verify that a refused request never calls the business handler.
- [ ] Verify that a failed access check never calls the business handler.
- [ ] Verify that platform registration rejects operations without an explicit policy.
- [ ] Verify that the caller cannot choose another actor through request data.
- [ ] Verify that direct domain calls require the same access checks.
- [ ] Verify that agents do not inherit their creator's credentials or rights automatically.
- [ ] Verify that generated bindings and the endpoint reference are current.
- [ ] Run `pnpm check`, `pnpm test`, `pnpm test:e2e`, and `pnpm build`.
- [ ] Update `README.md` and `AGENTS.md` to describe the implemented rules.

## Completion conditions

- The project Compose file starts a healthy PostgreSQL service.
- Database migrations prepare the schema.
- PostgreSQL stores actor references and permission data across restarts.
- Every RPC endpoint has documentation, an access policy, and a domain handler.
- Protected requests have a verified actor.
- Guarded domain operations check access before they call private business handlers.
- RPC callers and internal callers use the same authorization guard.
- The custom analyzer enforces the endpoint and domain rules.
- Tests cover allowed requests, refused requests, and failed permission checks.
- All required checks pass.

After the feature is complete, move this file to `plans/done/authorization.md`.
