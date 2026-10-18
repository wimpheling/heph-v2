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
- [ ] Confirm the scope kinds, parent relationships, and grant coverage proposed below.
- [ ] Confirm Git ref scopes and separate actions for ref updates and repository settings.
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
The scope definition and coverage remain under review.

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
`Action` names one registered operation, such as `repository.write`.
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

The proposed scope model below defines grant coverage.
For example, `repository.write` on a project covers repositories in that project.
Resolve resource containment through trusted core data.
Do not accept caller-supplied parent scopes as proof of containment.
Do not match unrelated scopes or actions.
Use exact action names without wildcard patterns for the first implementation.
The grant reader must return only active grants.
After revocation, the grant must not permit subsequent access checks.
An access grant does not by itself authorize issuing grants to other actors.

## Proposed scope model

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

## Proposed permissions for Git refs

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
- [ ] Keep SQL and PostgreSQL types inside `internal/core/postgres/`.
- [ ] Put PostgreSQL storage and migrations in `internal/core/postgres/`.
- [ ] Supply core persistence through the interfaces used by the business services.
- [ ] Use core PostgreSQL persistence with the bootstrap identity driver.
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
- [ ] Require an explicit actor parameter for protected operations, including internal calls.
- [ ] Keep actor identity out of context values and global state.
- [ ] Refuse caller-supplied actor IDs as proof of identity.
- [ ] Add the bootstrap identity driver.
- [ ] Put the bootstrap identity implementation in `internal/drivers/bootstrap/identity/`.
- [ ] Read bootstrap configuration through `internal/config`.

### 5. Check access in core

- [ ] Define the action, resource, scope, and grant types.
- [ ] Implement the approved grant record and validate its fields.
- [ ] Implement the approved scope kinds and validate parent relationships.
- [ ] Add authorization fixtures for exact Git ref scopes beneath repository scopes.
- [ ] Resolve grant scope coverage through trusted resource containment.
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
- [ ] Verify that grants match only the intended actor, action, and scope.
- [ ] Verify that grants cover contained resources and exclude unrelated scopes.
- [ ] Verify that scope grants do not apply to ancestors or sibling scopes.
- [ ] Verify that invalid scope relationships prevent protected actions.
- [ ] Verify that a Git ref grant excludes other refs and repositories.
- [ ] Verify that a ref update grant does not permit repository settings changes.
- [ ] Verify that narrow grants do not cancel broader grants.
- [ ] Verify that a revoked grant cannot permit subsequent access checks.
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
