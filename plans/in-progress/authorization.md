# Platform authorization

Status: Implementation complete. Final CI validation is pending. The PR remains open for human review.

Keep the plan and the feature implementation in the same pull request.

## Goal

Require each RPC endpoint to have a documented contract and a domain handler.
Identify each caller.
Check access before each protected action.
Store actor references and permission data in PostgreSQL.
Provide RPC endpoints to create, inspect, list, and revoke grants.
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

| Term               | Meaning                                                                                            |
| ------------------ | -------------------------------------------------------------------------------------------------- |
| Actor              | The human or agent that makes a request.                                                           |
| Actor reference    | The pair of `Source` and `Subject` that identifies an actor.                                       |
| Source             | The stable namespace of a trusted identity source.                                                 |
| Subject            | The stable actor identifier within that source.                                                    |
| Authentication     | The check that identifies the actor from a credential.                                             |
| Authorization      | The check that permits or refuses an action on a resource.                                         |
| Scope              | A named authorization boundary with a stable internal ID and one parent, except the platform root. |
| Grant              | A record that permits an actor to do an action in a scope.                                         |
| Domain operation   | The guarded entry point that checks access before it calls a business handler.                     |
| Domain handler     | The private function that runs the business logic after the access check.                          |
| Port               | A Go interface through which the domain requests an external operation.                            |
| Driver             | The implementation of a port for a selected system.                                                |
| Endpoint binding   | The typed connection between an RPC method and its guarded domain operation.                       |
| Database migration | A versioned change to the database schema.                                                         |

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
- [x] Make PostgreSQL an opinionated core dependency, with storage code in `internal/core/postgres/`.
- [x] Add a project `compose.yaml` file for local PostgreSQL.
- [x] Approve `internal/core/authorization` and the `authorization.GrantReader` interface as domain call targets.
- [x] Bind each RPC to a guarded domain operation with typed Go references and input and output conversion functions.
- [x] Use a protobuf method option with `AUTHENTICATION_PUBLIC`, `AUTHENTICATION_REQUIRED`, and `AUTHENTICATION_UNSPECIFIED` values.
- [x] Require an authorization guard before each private business handler, including operations called without RPC.
- [x] Identify humans and agents with stable `ActorRef` values containing `Source` and `Subject`.
- [x] Keep user account management in identity drivers, outside core.
- [x] Treat organizations as ownership scopes.
- [x] Pass the actor explicitly to each protected domain operation.
- [x] Keep session IDs, credential IDs, and run IDs separate from actor references.
- [x] Keep the permission engine in `internal/core/authorization`.
- [x] Supply identity and permission data through injected interfaces.
- [x] Start with one configured bootstrap user and a bootstrap credential.
- [x] Group drivers under `internal/drivers/bootstrap/` and `internal/drivers/std/`.
- [x] Use ownership rights and explicit grants for the first access checks.
- [x] Use the grant record fields defined below.
- [x] Use the scope kinds, parent relationships, and grant coverage defined below.
- [x] Use Git ref scopes and separate actions for ref updates and repository settings.
- [x] Add further scope boundaries when features require them, with explicit definitions and access tests.
- [x] Use the Go error exceptions and domain rules below.
- [x] Use `AuthorizationService.ListGrants` as the first protected endpoint.
- [x] Include the complete grant management lifecycle in this PR.
- [x] Use the grant RPC set and immutable grant fields below.
- [x] Use the grant creation and revocation policies below, including prevention of privilege escalation.
- [x] Use Playwright E2E tests for most behavioral verification, including API-only requests.
- [x] Produce separate unit, E2E, and combined coverage reports locally and in CI.

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

## Confirmed organization model

An organization is an ownership scope.
It can own projects and resources.
Humans and agents receive permissions within that scope.
Each request retains the verified human or agent as its actor.
For example, Alice creates a repository owned by Acme.
The authorization check uses Alice's permissions within Acme.

## Confirmed actor parameter

Each protected domain operation receives the verified actor as an explicit parameter.
For example: `CreateRepository(ctx, actor, input)`.
Authentication supplies the actor.
The RPC adapter supplies the translated input.
The operation's guard checks the actor's permissions before it calls the private handler.
Internal callers must also supply the actor explicitly.
Do not retrieve the actor from context values or global state.
Do not accept request fields as proof of the caller's identity.

## Confirmed ID separation

The actor reference identifies the human or agent that performs an action.
A session ID identifies a login session.
A credential ID identifies the credential used for authentication.
A run ID identifies one execution of an agent.
Keep these identifiers separate.
An actor retains its reference across sessions, credential rotations, and runs.
Permissions attach to the actor reference.
Keep session, credential, and run identifiers available for separate tracing.
This decision does not add session management or agent execution to this feature.

## Confirmed permission engine boundary

Keep permission rules in `internal/core/authorization`.
The engine evaluates ownership, grants, actions, and scopes to permit or refuse access.
Identity drivers and core persistence supply the required data through interfaces.
The guarded domain operation calls the engine before its private business handler.
RPC callers and internal callers use the same rules.
Keep HTTP, ConnectRPC, generated transport types, SQL, and PostgreSQL types outside the permission engine.
Test permission decisions without a database.

## Confirmed interface wiring

Use an authentication interface to verify credentials and return the actor.
Use `authorization.GrantReader` to load grants for permission checks.
At startup, supply the configured identity driver and core PostgreSQL grant implementation through ordinary Go constructors.
The permission engine depends on interfaces and core types.
Core must not import replaceable implementations under `internal/drivers/`.
Unit tests supply test doubles through the same interfaces.
Do not add a dependency injection framework.

## Confirmed bootstrap identity

Start with one configured bootstrap user and a bearer token.
The identity driver verifies the token and returns a stable bootstrap `ActorRef`.
Read the configuration through `internal/config`.
Keep the token outside Git.
The bootstrap actor uses the same authorization guard as every other actor.
Assign its permissions explicitly.

## Confirmed persistence and driver paths

```text
internal/
  core/
    postgres/       # Core PostgreSQL storage, SQL, and migrations.
  drivers/
    bootstrap/
      identity/     # Configured bootstrap identity and token verification.
    std/            # Standard implementations of replaceable components.
```

PostgreSQL is an opinionated core choice.
Keep PostgreSQL storage, SQL, and migrations in `internal/core/postgres/`.
Keep the permission engine independent of SQL and PostgreSQL types.
Use interfaces to test business rules without a database.
These interfaces do not require support for other database engines.
Keep replaceable component implementations outside `internal/core/`.
Put simplified bootstrap drivers under `internal/drivers/bootstrap/`.
Put standard drivers under `internal/drivers/std/`.
Use `internal/drivers/bootstrap/identity/` for the first identity driver.
Use `internal/server/` to connect core persistence and replaceable drivers to their interfaces during startup.
Keep these packages in the root Go module.

## Confirmed access rules

A human or agent owner can manage its owned scope and resources.
Other actors need a grant for the requested action and scope.
For organization-owned resources, check the caller's rights within that organization.
Refuse access when no rule permits the action.
Creating an agent does not give it access to its creator's resources.

## Confirmed grant record

Define these types in `internal/core/authorization`.
Use the confirmed scope definition and coverage below.

```go
type GrantID string
type Action string
type ScopeID string
type ScopeKind string

type ScopeRef struct {
    Kind ScopeKind
    ID   ScopeID
}

type Grant struct {
    ID        GrantID
    Grantee   ActorRef
    Action    Action
    Scope     ScopeRef
    GrantedBy ActorRef
    CreatedAt time.Time
}
```

`ID` identifies the grant for lookup and revocation.
`Grantee` identifies the human or agent that receives permission.
`Action` names one registered operation, such as `git.ref.update`.
`Scope` identifies the boundary in which the permission applies.
`GrantedBy` identifies the actor that issued the grant.
`CreatedAt` records when the grant was created.
The issuer and timestamp record provenance; they do not give the grantee additional rights.

Use one grant for each actor, action, and scope combination.
Accept only registered actions and scope kinds.
Refuse empty identifiers and invalid actor references.
Scope IDs are stable internal identifiers.
Resolve personal scope ownership through the complete actor reference, including its source.
Do not use an external subject alone as a personal scope ID.

The confirmed scope model below defines grant coverage.
For example, `git.ref.update` on a project covers refs in that project's repositories.
Resolve resource containment through trusted core data.
Do not accept caller-supplied parent scopes as proof of containment.
Do not match unrelated scopes or actions.
Use exact action names without wildcard patterns for the first implementation.
The grant reader must return only active grants.
After revocation, the grant must not permit subsequent access checks.
An access grant does not by itself authorize issuing grants to other actors.

## Confirmed scope model

A scope is the authorization boundary to which a grant applies.
It has a stable internal ID and a registered kind.
Each scope has one parent, except the platform root.
The pair of kind and ID identifies the scope.

```go
const (
    ScopePlatform     ScopeKind = "platform"
    ScopePersonal     ScopeKind = "personal"
    ScopeOrganization ScopeKind = "organization"
    ScopeProject      ScopeKind = "project"
    ScopeResource     ScopeKind = "resource"
    ScopeGitRef       ScopeKind = "git_ref"
)

type Scope struct {
    Ref    ScopeRef
    Parent *ScopeRef
}
```

| Kind         | Meaning                                                    | Permitted parent                    |
| ------------ | ---------------------------------------------------------- | ----------------------------------- |
| Platform     | The entire Heph installation. There is one platform scope. | None.                               |
| Personal     | Resources owned by one human or agent.                     | Platform.                           |
| Organization | Resources owned by one organization.                       | Platform.                           |
| Project      | A project and its resources.                               | Personal or organization.           |
| Resource     | One resource, such as a repository or secret.              | Personal, organization, or project. |
| Git ref      | One named Git ref within a repository.                     | The repository's resource scope.    |

```text
platform
  personal:alice
    project:personal-tools
      resource:repository-a
    resource:personal-secret
  organization:acme
    project:heph
      resource:repository-b
        git_ref:refs/heads/main
        git_ref:refs/heads/agents/builder
```

The labels in this example describe the scopes; stored references use stable internal IDs.
Bind a personal scope to its owner's complete `ActorRef` in trusted core storage.
Bind organization, project, and resource scopes to their corresponding core records.
A resource scope ID identifies its permission boundary, independently of its resource name or slug.
This scope definition does not implement the future resource features.

The core resolver loads scope kinds, IDs, and parent relationships from trusted storage.
Refuse unknown scopes, invalid parent kinds, cycles, and missing parent records.
Do not use caller-supplied parent relationships for authorization.
Organization membership does not change the parent of a personal scope.
An agent's creator relationship does not create scope containment.

A grant applies to its named scope and all descendants, for its exact action.
It does not apply to ancestors or siblings.
A grant at platform scope covers the installation and must be assigned explicitly.
For a resource action, resolve the existing resource's scope and its ancestors.
For resource creation, check the intended parent scope and any target-specific policy before creating the resource.
Moving a resource changes its inherited access and needs a separate authorized operation.

Add further scope boundaries when a feature needs finer access control.
Define each new kind, its valid parents, its target resolver, and its action coverage explicitly.
Add access tests for each new boundary.
Do not accept unregistered scope kinds.
Do not add unused scope kinds in advance.

## Confirmed permissions for Git refs

Treat each branch as a Git ref scope beneath its repository scope.
Identify the target with the repository's stable ID and the full Git ref name.
For example, use `refs/heads/main` and `refs/heads/agents/builder`.
The core resolver maps this target to its scope and trusted ancestors.
Do not require a live branch before an exact ref scope can be registered.
The scope identifies the ref name within that repository, including after deletion and recreation.

Use an exact action such as `git.ref.update` for an ordinary branch update.
Use separate actions for ref creation, ref deletion, forced updates, and repository settings.
Define each action's exact meaning before implementing its Git operation.
A ref update grant does not permit changing repository settings.

| Grant scope      | Coverage for `git.ref.update`         |
| ---------------- | ------------------------------------- |
| Repository       | All refs in that repository.          |
| One Git ref      | Only that ref within that repository. |
| Several Git refs | One grant for each selected ref.      |

A branch-limited agent must not receive broader grants for the same action.
Narrow grants add access; they do not remove inherited or ownership rights.
Protected branch rules would be a separate restrictive policy.
Confirm that policy before implementing protected branches.

The Git mutation path must check every affected ref before it changes any ref.
Ref creation must check the intended ref target, including any required parent-scope permission.
This plan defines the authorization model and its unit fixtures.
It does not implement Git pushes or branch management in the first platform feature.
Branch name patterns remain a separate decision; start with exact ref names.

## Confirmed domain lint rules

Apply the restrictions to `internal/core/domain/` and all its subpackages.
Permit calls only to explicitly approved services, ports, and helpers.
Business branches use named booleans, optionally combined with `!`, `&&`, and `||`.
Permit `if`, `else`, local variables, field access, and typed results.
Permit error propagation, error comparisons with `nil`, `errors.Is`, and `errors.As`.
Resolve Go types when applying error exceptions; a variable name alone is insufficient.
Refuse raw business comparisons, arithmetic, string calculations, loops, switches, and direct infrastructure access.
Keep computations in approved services outside the restricted domain packages.
The analyzer implementation and its fixtures remain work to complete.

## Confirmed first protected endpoint

The existing `GetSystemInfo` endpoint is public.
Add a protected endpoint to verify the complete authentication and authorization path.

| Item            | Definition                                               |
| --------------- | -------------------------------------------------------- |
| RPC             | `AuthorizationService.ListGrants`                        |
| Authentication  | `AUTHENTICATION_REQUIRED`                                |
| Input           | The target `ScopeRef` and pagination fields.             |
| Required action | `authorization.grants.read` on the target scope.         |
| Result          | A page of active grants attached directly to that scope. |

Resolve the target scope from trusted core storage before the access check.
Use the verified caller as the actor.
Do not permit the caller to select another actor for authentication.
Check access through the guarded domain operation before reading the grant results.
Do not include ancestor grants or descendant grants in this endpoint's results.
Use PostgreSQL records for the integration test.
Verify that denied requests never invoke the private handler.

## Grant management RPCs

Complete grant management is required for this PR.
Keep `ListGrants` as the first implementation step, then complete the remaining endpoints in the same PR.
Implement the following API shape.

| RPC           | Input                       | Result                                         | Required action                                      |
| ------------- | --------------------------- | ---------------------------------------------- | ---------------------------------------------------- |
| `CreateGrant` | Grantee, action, and scope. | The created grant.                             | `authorization.grants.create` on the proposed scope. |
| `GetGrant`    | Grant ID.                   | The grant and its revocation state.            | `authorization.grants.read` on its stored scope.     |
| `ListGrants`  | Scope and pagination.       | Active grants attached directly to that scope. | `authorization.grants.read` on the requested scope.  |
| `RevokeGrant` | Grant ID.                   | The recorded revocation.                       | `authorization.grants.revoke` on its stored scope.   |

Require authentication for all four endpoints.
The server assigns grant IDs, creation timestamps, and `GrantedBy` from the verified actor.
Clients must not choose the issuer or creation timestamp.
Keep the grantee, action, and scope immutable after creation.
Change an assignment by revoking the old grant and creating a new grant.
The API has no `UpdateGrant` endpoint.
Retain the original grant record after revocation.
Record the revoking actor and revocation timestamp.
Exclude revoked grants from access decisions and the active grant list.
Derive authorization scopes for ID-based requests from trusted stored records.

## Grant mutation policy

Apply these rules to grant mutations.
Creating a grant requires `authorization.grants.create` on the target scope.
The caller must also hold the action being granted across the entire target scope.
A branch-limited grant cannot justify granting the same action on the whole repository.
Do not allow a caller to grant an action or scope coverage that the caller does not have.
Creating grant-management permissions follows the same rule.
Revoking a grant requires `authorization.grants.revoke` on that grant's stored scope.
The original issuer does not receive an automatic exception to these checks.
Run the final permission checks and mutation in one consistent database transaction.
Refuse the mutation if permissions cannot be verified.

Bootstrap setup seeds the initial explicit grants before the first protected API call.
Include the grant-management actions needed by the bootstrap actor.
This initialization is the starting point for the first grant, rather than another grant RPC call.
After initialization, normal grant operations must use the guarded domain operations.

## Work steps

### 1. Set up PostgreSQL

- [x] Add a PostgreSQL service to the root `compose.yaml` file.
- [x] Pin the PostgreSQL version.
- [x] Bind the local database port to `127.0.0.1`.
- [x] Add a named volume to keep data across container restarts.
- [x] Add a database health check.
- [x] Wait for database readiness before migrations and application startup.
- [x] Add `.env.example` with local database settings and placeholder credentials.
- [x] Read the application database connection settings through `internal/config`.
- [x] Create the database connection pool during platform startup.
- [x] Close the database connection pool during shutdown.
- [x] Add versioned migrations for actor references and permission data.
- [x] Keep any local account storage inside the identity driver.
- [x] Keep SQL and PostgreSQL types inside `internal/core/postgres/`.
- [x] Put PostgreSQL storage and migrations in `internal/core/postgres/`.
- [x] Supply core persistence through the interfaces used by the business services.
- [x] Use core PostgreSQL persistence with the bootstrap identity driver.
- [x] Use test doubles for isolated unit tests.
- [x] Add commands to start, stop, migrate, and reset the local database.
- [x] Document how the Go application receives the database connection settings.
- [x] Use the same PostgreSQL version for local development and CI.
- [x] Prepare a separate database or schema for each integration test run.
- [x] Prepare the database for integration tests and browser tests.

### 2. Define and document the endpoint contract

- [x] Add comments for each RPC method and its request and response fields.
- [x] Describe the purpose, inputs, outputs, and expected errors.
- [x] Require an explicit access policy for each RPC method.
- [x] Declare `GetSystemInfo` as public.
- [x] Define all four approved grant RPCs and their documented authorization actions.
- [x] Refuse an access policy that is missing or unknown.
- [x] Generate an endpoint reference from the protobuf contract.
- [x] Put the generated reference outside `docs/`.

### 3. Connect each endpoint to its domain handler

- [x] Add a typed endpoint binding.
- [x] Bind each RPC adapter to a guarded domain operation instead of its raw business handler.
- [x] Use actual Go function references instead of handler names stored as strings.
- [x] Add typed conversion functions for the RPC request and the domain result.
- [x] Require exactly one binding for each RPC method.
- [x] Use Go type checks to verify compatible binding types.
- [x] Give the system operation an explicit public policy and a private domain handler.
- [x] Limit RPC adapters to authentication, data translation, delegation, and error translation.
- [x] Require endpoint registration through the platform.
- [x] Use only core types in the domain.

### 4. Identify the caller

- [x] Define the actor type and the authentication interface.
- [x] Use the verified `Source` and `Subject` pair as the actor reference.
- [x] Keep account management outside core.
- [x] Verify the credential before each protected RPC method.
- [x] Return an actor only after successful authentication.
- [x] Pass the verified actor from the RPC adapter to the guarded domain operation.
- [x] Require an explicit actor parameter for protected operations, including internal calls.
- [x] Keep actor identity out of context values and global state.
- [x] Refuse caller-supplied actor IDs as proof of identity.
- [x] Add the bootstrap identity driver.
- [x] Put the bootstrap identity implementation in `internal/drivers/bootstrap/identity/`.
- [x] Read bootstrap configuration through `internal/config`.

### 5. Check access in core

- [x] Define action and grant types, plus scope types with trusted resource bindings.
- [x] Implement the approved grant record and validate its fields.
- [x] Implement the approved grant creation and revocation policies.
- [x] Persist grant creation and revocation through core PostgreSQL transactions.
- [x] Keep issuer and revocation metadata controlled by the server.
- [x] Implement the approved scope kinds and validate parent relationships.
- [x] Add authorization fixtures for exact Git ref scopes beneath repository scopes.
- [x] Resolve grant scope coverage through trusted resource containment.
- [x] Require a policy and a private business handler for each domain operation.
- [x] Require a resource resolver when the policy needs resource data.
- [x] Resolve resource data through trusted services instead of trusting caller-supplied ownership fields.
- [x] Run the access check before the business handler.
- [x] Reject operations with missing or unknown policies during platform registration.
- [x] Require an explicit policy for public operations.
- [x] Use the parent scope for an action that creates a resource.
- [x] Refuse access unless a defined rule permits the action.
- [x] Refuse unknown actions and unknown scopes.
- [x] Refuse the action if the permission check fails.
- [x] Apply the same checks to human and agent actors.
- [x] Keep permission checks inside the domain operation.
- [x] Map authentication and permission errors at the RPC boundary.
- [x] Require a delegation policy before adding an operation that issues agent credentials.

### 6. Enforce the rules with the custom Go analyzer

Use resolved Go symbols and receiver types.
Do not approve a call because its type name contains `Service`.

- [x] Require endpoint comments and explicit access policies.
- [x] Find missing, repeated, or invalid endpoint bindings.
- [x] Require each operation's business handler to belong to `internal/core/domain/` or one of its subpackages.
- [x] Require an explicit authorization policy for each domain operation.
- [x] Find RPC adapters that bypass the guarded domain operation.
- [x] Refuse raw handler references outside the approved operation definitions.
- [x] Refuse direct calls to raw business handlers in production code.
- [x] Find endpoint registrations that bypass platform authentication.
- [x] Permit domain calls only to approved services, ports, and explicit helpers.
- [x] Permit boolean decisions with `!`, `&&`, and `||`.
- [x] Refuse raw comparisons, arithmetic, string calculations, and similar computations in domain handlers.
- [x] Find prohibited computations outside branch conditions as well as inside them.
- [x] Permit `if` and `else` for business branches.
- [x] Refuse loops, switches, goroutines, and channel operations in domain handlers.
- [x] Permit local bindings, field reads, argument passing, and typed result construction.
- [x] Permit error returns, `err == nil`, `err != nil`, `errors.Is`, and `errors.As`.
- [x] Refuse arbitrary function variables and anonymous function calls in domain handlers.
- [x] Refuse direct file, database, network, environment, and process operations in domain handlers.
- [x] Exclude tests and generated code from the domain restrictions.
- [x] Apply the domain restrictions to `internal/core/domain/` and all its subpackages.
- [x] Keep domain services and adapters outside the restricted domain package.
- [x] Add valid and invalid fixtures for each rule.
- [x] Add fixtures for import aliases and attempted bypasses.
- [x] Add the checks to `pnpm check` and CI.

The analyzer checks code structure.
Access tests must also check the permission decisions.

### 7. Verify the complete request path

- [x] Keep the dedicated Playwright workspace at `packages/e2e/` self-contained and portable.
- [x] Add an API-only Playwright project with real RPC requests against the Go server and PostgreSQL.
- [x] Put most behavioral scenarios in Playwright; keep focused unit tests for policies, guards, and analyzer fixtures.
- [x] Isolate test database state and credentials from development data.
- [x] Collect Go coverage from the real server process during Playwright tests.
- [x] Produce unit, E2E, and combined coverage profiles, HTML reports, and machine-readable summaries.
- [x] Merge covered statements rather than averaging unit and E2E percentages.
- [x] Use the same handwritten application package set for all three coverage reports.
- [ ] Run the test suites and coverage collection in CI and publish report artifacts.

- [x] Verify that the Compose service becomes healthy.
- [x] Verify that migrations prepare an empty database.
- [x] Verify that actor references and permission data survive application and database restarts.
- [x] Verify that database errors prevent protected actions.
- [x] Verify that the public system endpoint works without a credential.
- [x] Verify that a protected endpoint refuses missing and invalid credentials.
- [x] Verify that a valid credential produces the correct actor.
- [x] Verify that an allowed actor can complete a protected action.
- [x] Verify that the approved protected endpoint returns persisted data only after authorization.
- [x] Verify the complete create, get, list, and revoke lifecycle through real RPC requests.
- [x] Verify that the server assigns issuer and creation metadata from trusted values.
- [x] Verify that revocation records the verified actor and timestamp.
- [x] Verify that revoked records remain inspectable by authorized callers.
- [x] Verify that a caller cannot grant actions or scope coverage it does not hold.
- [x] Verify that a branch-limited caller cannot grant repository-wide access.
- [x] Verify that grant-management rights cannot be escalated through grant creation.
- [x] Verify that denied mutations and database failures leave grant state unchanged.
- [x] Verify that a denied actor cannot complete the action.
- [x] Verify that grants match only the intended actor, action, and scope.
- [x] Verify that grants cover contained resources and exclude unrelated scopes.
- [x] Verify that scope grants do not apply to ancestors or sibling scopes.
- [x] Verify that invalid scope relationships prevent protected actions.
- [x] Verify that a Git ref grant excludes other refs and repositories.
- [x] Verify that a ref update grant does not permit repository settings changes.
- [x] Verify that narrow grants do not cancel broader grants.
- [x] Verify that a revoked grant cannot permit subsequent access checks.
- [x] Verify that a permission error prevents the protected action.
- [x] Verify that a refused request never calls the business handler.
- [x] Verify that a failed access check never calls the business handler.
- [x] Verify that platform registration rejects operations without an explicit policy.
- [x] Verify that the caller cannot choose another actor through request data.
- [x] Verify that direct domain calls require the same access checks.
- [x] Verify that agents do not inherit their creator's credentials or rights automatically.
- [x] Verify that generated bindings and the endpoint reference are current.
- [x] Run `pnpm check`, `pnpm test`, `pnpm test:e2e`, and `pnpm build`.
- [x] Update `README.md` and `AGENTS.md` to describe the implemented rules.

## Definition of done

The pull request is complete only when all these conditions are true.
Plan approval alone does not complete the feature.

- A fresh checkout can start PostgreSQL with the project `compose.yaml` file.
- Versioned migrations prepare an empty database without manual SQL changes.
- Documented commands start, stop, migrate, and reset the local database.
- Core PostgreSQL persistence stores actor references, scopes, and grants across restarts.
- Bootstrap setup provides the initial scope and explicit grants needed to manage grants through the API.
- The bootstrap identity driver verifies its configured token and returns the correct stable actor reference.
- Credentials stay outside Git; `.env.example` contains placeholders.
- `GetSystemInfo` remains explicitly public and works without a credential.
- `AuthorizationService.ListGrants` requires authentication and `authorization.grants.read` on the requested scope.
- `ListGrants` returns paginated active grants attached directly to the authorized scope.
- `CreateGrant`, `GetGrant`, and `RevokeGrant` implement the approved lifecycle and access policies.
- Grant creation cannot assign rights or scope coverage beyond the caller's own rights.
- Grant records remain immutable; revocation records who revoked access and when.
- An authorized caller can create, inspect, list, revoke, and inspect the revoked grant through RPC.
- Revoked grants no longer permit subsequent access checks.
- Missing credentials, invalid credentials, and insufficient permissions prevent the private handler from running.
- Permission lookup failures also prevent the private handler from running.
- Core implements the approved grant model, scope hierarchy, ownership rules, and exact action matching.
- Access tests prove inherited access, scope isolation, Git ref boundaries, and removal of revoked access.
- Every RPC has documented protobuf contracts, explicit access metadata, and one typed binding to a guarded domain operation.
- Internal callers use the same guarded operations and pass the actor explicitly.
- Generated Go bindings, TypeScript bindings, and the endpoint reference are current and committed.
- The generated endpoint reference is outside `docs/`.
- The custom Go analyzer enforces the approved domain, endpoint binding, and authorization guard rules.
- Valid and invalid analyzer fixtures cover each rule, including aliases and attempted bypasses.
- RPC integration tests use PostgreSQL and prove both allowed and refused requests.
- Playwright API tests verify the complete grant lifecycle and most observable authorization behavior.
- CI runs unit tests and Playwright tests against real PostgreSQL.
- Unit, E2E, and combined coverage reports are available as CI artifacts.
- `pnpm check`, `pnpm test`, `pnpm test:e2e`, and `pnpm build` pass, including the required CI checks.
- `README.md` documents setup and the complete grant RPC lifecycle.
- `AGENTS.md` describes the implemented architecture and lint rules.
- All completed work is committed and pushed to this feature PR.
- The PR is ready for human review and remains unmerged.

## Feature boundary

This PR builds the platform authorization foundation and the complete grant management RPC lifecycle.
Git ref scopes are implemented and tested as authorization boundaries.
Git push handling and live branch management are later features.
Grant creation and revocation APIs implement the agreed mutation policies.
External identity providers, login screens, and session management are later features.
Agent execution, chat, and the builder agent are later features.

After the feature is complete, move this file to `plans/done/authorization.md`.
