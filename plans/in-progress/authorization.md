# Platform authorization

Status: Plan for review. The feature is not complete.

Keep the plan and the feature implementation in the same pull request.

## Goal

Require each RPC endpoint to have a documented contract and a domain handler.
Identify each caller.
Check access before each protected action.

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

| Term             | Meaning                                                                 |
| ---------------- | ----------------------------------------------------------------------- |
| Actor            | The human or agent that makes a request.                                |
| Authentication   | The check that identifies the actor from a credential.                  |
| Authorization    | The check that permits or refuses an action on a resource.              |
| Scope            | An ownership boundary, such as a user, organization, or project.        |
| Grant            | A record that permits an actor to do an action in a scope.              |
| Domain handler   | The function that controls a business operation.                        |
| Port             | A Go interface through which the domain requests an external operation. |
| Driver           | The implementation of a port for a selected system.                     |
| Endpoint binding | The typed connection between an RPC method and its domain handler.      |

## Proposed request path

```text
RPC endpoint
  -> authentication
  -> RPC adapter
  -> domain handler
  -> authorization check
  -> approved services and ports
```

The RPC adapter translates requests, results, and errors.
The domain handler controls the business operation.
The domain handler checks access before each protected action.
A service can load resource data that the access check requires.
Such data must not reach the caller before the check permits access.

Internal callers must use the same domain operations and access checks.

## Decisions to confirm

Confirm these choices before implementation.
Review one decision at a time.
After approval, mark the decision complete and explain the next decision.

- [x] Use `internal/core/domain/` for the restricted domain handlers.
- [ ] Select the approved service packages and port interfaces.
- [ ] Select the typed form of the endpoint bindings.
- [ ] Select the protobuf metadata for public access and required authentication.
- [ ] Use stable actor IDs for humans and agents.
- [ ] Treat organizations as ownership scopes.
- [ ] Pass the actor explicitly to domain operations.
- [ ] Keep session IDs, credential IDs, and run IDs separate from actor IDs.
- [ ] Keep the permission engine in core.
- [ ] Supply identity and permission data through injected interfaces.
- [ ] Start with one configured bootstrap user and a bootstrap credential.
- [ ] Use memory storage for the first bootstrap drivers.
- [ ] Define ownership rights and explicit grants for the first access checks.
- [ ] Confirm the Go error exceptions and the domain rules below.

## Work steps

### 1. Define and document the endpoint contract

- [ ] Add comments for each RPC method and its request and response fields.
- [ ] Describe the purpose, inputs, outputs, and expected errors.
- [ ] Require an explicit access policy for each RPC method.
- [ ] Declare `GetSystemInfo` as public.
- [ ] Refuse an access policy that is missing or unknown.
- [ ] Generate an endpoint reference from the protobuf contract.
- [ ] Put the generated reference outside `docs/`.

### 2. Connect each endpoint to its domain handler

- [ ] Add a typed endpoint binding.
- [ ] Require exactly one binding for each RPC method.
- [ ] Use Go type checks to verify compatible binding types.
- [ ] Move the system operation behind a domain handler.
- [ ] Limit RPC adapters to authentication, data translation, delegation, and error translation.
- [ ] Require endpoint registration through the platform.
- [ ] Use only core types in the domain.

### 3. Identify the caller

- [ ] Define the actor type and the authentication interface.
- [ ] Verify the credential before each protected RPC method.
- [ ] Return an actor only after successful authentication.
- [ ] Pass the verified actor from the RPC adapter to the domain handler.
- [ ] Refuse caller-supplied actor IDs as proof of identity.
- [ ] Add the bootstrap identity driver.
- [ ] Read bootstrap configuration through `internal/config`.

### 4. Check access in core

- [ ] Define the action, resource, scope, and grant types.
- [ ] Use the parent scope for an action that creates a resource.
- [ ] Refuse access unless a defined rule permits the action.
- [ ] Refuse unknown actions and unknown scopes.
- [ ] Refuse the action if the permission check fails.
- [ ] Apply the same checks to human and agent actors.
- [ ] Keep permission checks inside the domain operation.
- [ ] Map authentication and permission errors at the RPC boundary.
- [ ] Check delegated rights before a future operation issues an agent credential.

### 5. Enforce the rules with the custom Go analyzer

Use resolved Go symbols and receiver types.
Do not approve a call because its type name contains `Service`.

- [ ] Require endpoint comments and explicit access policies.
- [ ] Find missing, repeated, or invalid endpoint bindings.
- [ ] Require each bound handler to belong to the domain package.
- [ ] Find RPC adapters that bypass the domain handler.
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
- [ ] Keep domain services and adapters outside the restricted domain package.
- [ ] Add valid and invalid fixtures for each rule.
- [ ] Add fixtures for import aliases and attempted bypasses.
- [ ] Add the checks to `pnpm check` and CI.

The analyzer checks code structure.
Access tests must also check the permission decisions.

### 6. Verify the complete request path

- [ ] Verify that the public system endpoint works without a credential.
- [ ] Verify that a protected endpoint refuses missing and invalid credentials.
- [ ] Verify that a valid credential produces the correct actor.
- [ ] Verify that an allowed actor can complete a protected action.
- [ ] Verify that a denied actor cannot complete the action.
- [ ] Verify that a permission error prevents the protected action.
- [ ] Verify that the caller cannot choose another actor through request data.
- [ ] Verify that direct domain calls require the same access checks.
- [ ] Verify that agents do not inherit their creator's credentials or rights automatically.
- [ ] Verify that generated bindings and the endpoint reference are current.
- [ ] Run `pnpm check`, `pnpm test`, `pnpm test:e2e`, and `pnpm build`.
- [ ] Update `README.md` and `AGENTS.md` to describe the implemented rules.

## Completion conditions

- Every RPC endpoint has documentation, an access policy, and a domain handler.
- Protected requests have a verified actor.
- Protected actions have an access check inside the domain operation.
- The custom analyzer enforces the endpoint and domain rules.
- Tests cover allowed requests, refused requests, and failed permission checks.
- All required checks pass.

After the feature is complete, move this file to `plans/done/authorization.md`.
