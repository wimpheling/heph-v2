import { expect, test } from "@playwright/test";
import {
  actor,
  admin,
  agent,
  alice,
  other,
  scope,
  platform,
  rpc,
  ok,
  grant,
  revoke,
  denied,
} from "./helpers.js";

test.describe.configure({ mode: "serial" });
const project = scope("PROJECT", "project-a");
const repo = scope("RESOURCE", "repo-a");
const branch = scope("GIT_REF", "branch-a");
const read = "authorization.grants.read";
const create = "authorization.grants.create";
const remove = "authorization.grants.revoke";

test("bootstrap has explicit grants and public metadata needs no credential", async ({
  request,
}) => {
  const result = await ok(request, "ListGrants", { scope: platform });
  expect(result.grants.map((g: { action: string }) => g.action)).toEqual(
    expect.arrayContaining([read, create, remove]),
  );
  const response = await request.post("/heph.system.v1.SystemService/GetSystemInfo", { data: {} });
  expect(response.ok()).toBe(true);
  expect((await response.json()).name).toBe("heph-v2");
});

for (const method of ["CreateGrant", "GetGrant", "ListGrants", "RevokeGrant"]) {
  test(method + " requires a valid credential before input processing", async ({ request }) => {
    for (const token of [undefined, "invalid-test-token"]) {
      const response = await rpc(request, method, {}, token);
      expect(response.status()).toBe(401);
      expect((await response.json()).code).toBe("unauthenticated");
    }
  });
}

test("creates, gets, lists, revokes and retains original audit data", async ({ request }) => {
  const assignment = await grant(request, agent, read, project);
  expect(assignment.grantee).toEqual(actor(agent));
  expect(assignment.grantedBy).toEqual(actor(admin));
  expect(Date.parse(assignment.createdAt)).not.toBeNaN();
  expect((await ok(request, "GetGrant", { id: assignment.id })).grant).toEqual(assignment);
  const list = await ok(request, "ListGrants", { scope: project }, agent.token);
  expect(list.grants).toContainEqual(assignment);
  const first = await revoke(request, assignment.id);
  expect(first.revocation.revokedBy).toEqual(actor(admin));
  expect(await revoke(request, assignment.id)).toEqual(first);
  const record = await ok(request, "GetGrant", { id: assignment.id });
  expect(record.grant).toEqual(assignment);
  expect(record.revocation).toEqual(first.revocation);
  await denied(request, "ListGrants", { scope: project }, agent.token);
  expect((await ok(request, "ListGrants", { scope: project })).grants ?? []).not.toContainEqual(
    assignment,
  );
});

test("source and subject jointly identify the caller", async ({ request }) => {
  const assignment = await grant(request, agent, read, project);
  await ok(request, "ListGrants", { scope: project }, agent.token);
  await denied(request, "ListGrants", { scope: project }, other.token);
  await revoke(request, assignment.id);
});

test("a project grant reaches descendants, but not siblings or ancestors", async ({ request }) => {
  const assignment = await grant(request, agent, read, project);
  for (const target of [project, repo, branch, scope("RESOURCE", "secret-a")]) {
    await ok(request, "ListGrants", { scope: target }, agent.token);
  }
  for (const target of [
    platform,
    scope("ORGANIZATION", "acme"),
    scope("PROJECT", "project-b"),
    scope("GIT_REF", "branch-b"),
  ]) {
    await denied(request, "ListGrants", { scope: target }, agent.token);
  }
  await revoke(request, assignment.id);
});

test("branch-only grants stay inside that repository and ref", async ({ request }) => {
  const assignment = await grant(request, agent, read, branch);
  await ok(request, "ListGrants", { scope: branch }, agent.token);
  for (const target of [repo, scope("GIT_REF", "branch-main"), scope("GIT_REF", "branch-b")]) {
    await denied(request, "ListGrants", { scope: target }, agent.token);
  }
  await revoke(request, assignment.id);
});

test("narrow grants do not remove inherited permissions", async ({ request }) => {
  const broad = await grant(request, agent, read, project);
  const narrow = await grant(request, agent, read, branch);
  await ok(request, "ListGrants", { scope: scope("GIT_REF", "branch-main") }, agent.token);
  await revoke(request, broad.id);
  await denied(request, "ListGrants", { scope: scope("GIT_REF", "branch-main") }, agent.token);
  await ok(request, "ListGrants", { scope: branch }, agent.token);
  await revoke(request, narrow.id);
});

test("creation requires management rights and possession of delegated rights", async ({
  request,
}) => {
  const manage = await grant(request, agent, create, branch);
  const body = { grantee: actor(alice), action: "git.ref.update", scope: branch };
  await denied(request, "CreateGrant", body, agent.token);
  const update = await grant(request, agent, "git.ref.update", branch);
  const created = await ok(request, "CreateGrant", body, agent.token);
  expect(created.grant.grantedBy).toEqual(actor(agent));
  await denied(
    request,
    "CreateGrant",
    { ...body, action: "repository.settings.update" },
    agent.token,
  );
  await denied(request, "CreateGrant", { ...body, scope: repo }, agent.token);
  await denied(
    request,
    "CreateGrant",
    { ...body, scope: scope("GIT_REF", "branch-main") },
    agent.token,
  );
  await denied(request, "CreateGrant", { ...body, action: remove }, agent.token);
  const broaderManagement = await grant(request, agent, create, repo);
  // Repository management cannot widen branch-only write rights.
  await denied(request, "CreateGrant", { ...body, scope: repo }, agent.token);
  await revoke(request, broaderManagement.id);
  await revoke(request, created.grant.id);
  await revoke(request, update.id);
  await revoke(request, manage.id);
});

test("the original issuer still needs revocation rights", async ({ request }) => {
  const manage = await grant(request, agent, create, branch);
  const update = await grant(request, agent, "git.ref.update", branch);
  const result = await ok(
    request,
    "CreateGrant",
    { grantee: actor(alice), action: "git.ref.update", scope: branch },
    agent.token,
  );
  await denied(request, "RevokeGrant", { id: result.grant.id }, agent.token);
  const revoker = await grant(request, agent, remove, branch);
  const revoked = await ok(request, "RevokeGrant", { id: result.grant.id }, agent.token);
  expect(revoked.revocation.revokedBy).toEqual(actor(agent));
  for (const entry of [manage, update, revoker]) await revoke(request, entry.id);
});

test("IDs resolve stored scopes and cannot bypass authorization", async ({ request }) => {
  const existing = await grant(request, alice, read, project);
  for (const method of ["GetGrant", "RevokeGrant"]) {
    await denied(request, method, { id: existing.id }, agent.token);
    const missing = await rpc(request, method, { id: "nonexistent-grant" }, admin.token);
    expect(missing.status()).toBe(404);
  }
  await revoke(request, existing.id);
});

test("duplicate active grants conflict; revoked assignments can be issued again", async ({
  request,
}) => {
  const first = await grant(request, agent, read, branch);
  const duplicate = await rpc(
    request,
    "CreateGrant",
    { grantee: actor(agent), action: read, scope: branch },
    admin.token,
  );
  expect(duplicate.status()).toBe(409);
  await revoke(request, first.id);
  const second = await grant(request, agent, read, branch);
  expect(second.id).not.toBe(first.id);
  await revoke(request, second.id);
});

test("pagination is bounded, ordered, and scoped to one exact boundary", async ({ request }) => {
  const result = await ok(request, "ListGrants", { scope: platform, pageSize: 2 });
  expect(result.grants).toHaveLength(2);
  expect(result.nextPageToken).toBeTruthy();
  const next = await ok(request, "ListGrants", {
    scope: platform,
    pageSize: 2,
    pageToken: result.nextPageToken,
  });
  expect(next.grants[0].id > result.grants[1].id).toBe(true);
  await denied(
    request,
    "ListGrants",
    { scope: project, pageToken: result.nextPageToken },
    admin.token,
    "invalid_argument",
  );
  const direct = await grant(request, agent, read, branch);
  const parent = await ok(request, "ListGrants", { scope: repo });
  expect((parent.grants ?? []).some((g: { id: string }) => g.id === direct.id)).toBe(false);
  await revoke(request, direct.id);
});

test("invalid scopes, actions, identities, limits and cursors fail closed", async ({ request }) => {
  for (const input of [
    {},
    { scope: { kind: "SCOPE_KIND_UNSPECIFIED", id: "x" } },
    { scope: { kind: "SCOPE_KIND_PLATFORM", id: "\u0000" } },
    { scope: platform, pageSize: -1 },
    { scope: platform, pageSize: 101 },
    { scope: platform, pageToken: "invalid-token" },
  ])
    await denied(request, "ListGrants", input, admin.token, "invalid_argument");
  for (const input of [
    { grantee: {}, action: read, scope: platform },
    { grantee: actor(agent), action: "unknown.action", scope: platform },
    { grantee: actor(agent), action: read, scope: { kind: "SCOPE_KIND_PROJECT", id: "" } },
  ])
    await denied(request, "CreateGrant", input, admin.token, "invalid_argument");
  const unknown = await rpc(
    request,
    "ListGrants",
    { scope: scope("PROJECT", "missing-project") },
    admin.token,
  );
  expect(unknown.status()).toBe(404);
});

test("caller-supplied audit fields and actor claims cannot become trusted identity", async ({
  request,
}) => {
  const forged = await rpc(
    request,
    "CreateGrant",
    {
      grantee: actor(alice),
      action: read,
      scope: platform,
      grantedBy: actor(admin),
    },
    agent.token,
  );
  expect([400, 403]).toContain(forged.status());
  await denied(request, "ListGrants", { scope: platform, actor: actor(admin) }, agent.token);
  const created = await ok(request, "CreateGrant", {
    grantee: actor(alice),
    action: read,
    scope: branch,
    grantedBy: actor(agent),
    createdAt: "2000-01-01T00:00:00Z",
  });
  expect(created.grant.grantedBy).toEqual(actor(admin));
  expect(Date.parse(created.grant.createdAt)).toBeGreaterThan(Date.parse("2000-01-01T00:00:00Z"));
  await revoke(request, created.grant.id);
});

test("concurrent duplicate creation persists only one active grant", async ({ request }) => {
  const body = { grantee: actor(agent), action: read, scope: scope("ORGANIZATION", "other") };
  const responses = await Promise.all([
    rpc(request, "CreateGrant", body, admin.token),
    rpc(request, "CreateGrant", body, admin.token),
  ]);
  expect(responses.map((r) => r.status()).sort((a, b) => a - b)).toEqual([200, 409]);
  const created = await responses.find((r) => r.status() === 200)!.json();
  await revoke(request, created.grant.id);
});
