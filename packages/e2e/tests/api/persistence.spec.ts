import { expect, test } from "@playwright/test";
import { spawn } from "node:child_process";
import pg from "pg";
import { databaseURL, schema, binary, root, serverEnv } from "../../harness/config.mjs";
import { admin, agent, alice, actor, platform, scope, grant, revoke, rpc, ok } from "./helpers.js";

test("a database failure at commit rolls back the assignment and discards its result", async ({
  request,
}) => {
  const client = new pg.Client({ connectionString: databaseURL });
  await client.connect();
  await client.query('SET search_path TO "' + schema + '"');
  await client.query(`
    CREATE FUNCTION e2e_fail_commit() RETURNS trigger LANGUAGE plpgsql AS $$
    BEGIN
      IF NEW.grantee_subject = 'rollback-probe' THEN RAISE EXCEPTION 'injected commit failure'; END IF;
      RETURN NEW;
    END $$;
    CREATE CONSTRAINT TRIGGER e2e_commit_failure AFTER INSERT ON grants
    DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION e2e_fail_commit();
  `);
  try {
    const result = await rpc(
      request,
      "CreateGrant",
      {
        grantee: { source: "e2e", subject: "rollback-probe" },
        action: "authorization.grants.read",
        scope: platform,
      },
      admin.token,
    );
    expect(result.status()).toBe(503);
    expect((await result.json()).grant).toBeUndefined();
    const persisted = await client.query(
      "SELECT count(*)::int AS total FROM grants WHERE grantee_subject='rollback-probe'",
    );
    expect(persisted.rows[0].total).toBe(0);
  } finally {
    await client.query(
      "DROP TRIGGER e2e_commit_failure ON grants; DROP FUNCTION e2e_fail_commit()",
    );
    await client.end();
  }
});

test("a queued mutation observes a preceding revocation instead of using stale rights", async ({
  request,
}) => {
  const branch = scope("GIT_REF", "branch-a");
  const manage = await grant(request, agent, "authorization.grants.create", branch);
  const update = await grant(request, agent, "git.ref.update", branch);
  const client = new pg.Client({ connectionString: databaseURL });
  await client.connect();
  await client.query("BEGIN; SELECT pg_advisory_xact_lock(728492317)");
  const waiting = async () => {
    // Activity snapshots are cached for this transaction; refresh each observation.
    await client.query("SELECT pg_stat_clear_snapshot()");
    const result = await client.query(
      "SELECT count(*)::int AS total FROM pg_stat_activity WHERE wait_event_type='Lock' AND query LIKE 'SELECT pg_advisory_xact_lock%'",
    );
    return result.rows[0].total as number;
  };
  let pendingRevoke;
  let pendingCreate;
  try {
    pendingRevoke = rpc(request, "RevokeGrant", { id: update.id }, admin.token);
    await expect.poll(waiting).toBe(1);
    pendingCreate = rpc(
      request,
      "CreateGrant",
      { grantee: actor(alice), action: "git.ref.update", scope: branch },
      agent.token,
    );
    await expect.poll(waiting).toBe(2);
    await client.query("COMMIT");
    expect((await pendingRevoke).status()).toBe(200);
    const denied = await pendingCreate;
    expect(denied.status()).toBe(403);
    expect((await denied.json()).code).toBe("permission_denied");
  } finally {
    await client.query("ROLLBACK");
    await Promise.allSettled([pendingRevoke, pendingCreate]);
    await client.end();
    await revoke(request, update.id);
    await revoke(request, manage.id);
  }
});

test("permission storage errors fail closed without persisting a grant", async ({ request }) => {
  const client = new pg.Client({ connectionString: databaseURL });
  await client.connect();
  await client.query('SET search_path TO "' + schema + '"');
  const before = await client.query("SELECT count(*) FROM grants");
  await client.query("ALTER TABLE grants RENAME TO grants_unavailable");
  try {
    for (const method of ["ListGrants", "CreateGrant"]) {
      const result = await rpc(
        request,
        method,
        {
          scope: platform,
          ...(method === "CreateGrant"
            ? { grantee: actor(agent), action: "authorization.grants.read" }
            : {}),
        },
        admin.token,
      );
      expect(result.status()).toBe(503);
      const text = await result.text();
      expect(text).not.toContain("grants_unavailable");
      expect(JSON.parse(text).code).toBe("unavailable");
    }
  } finally {
    await client.query("ALTER TABLE grants_unavailable RENAME TO grants");
    const after = await client.query("SELECT count(*) FROM grants");
    expect(after.rows).toEqual(before.rows);
    await client.end();
  }
});

test("invalid trusted scope relationships are rejected", async ({ request }) => {
  const client = new pg.Client({ connectionString: databaseURL });
  await client.connect();
  await client.query('SET search_path TO "' + schema + '"');
  await client.query(
    "INSERT INTO scopes(kind,id,parent_kind,parent_id) VALUES('project','invalid-parent','platform','platform')",
  );
  try {
    const result = await rpc(
      request,
      "ListGrants",
      { scope: scope("PROJECT", "invalid-parent") },
      admin.token,
    );
    expect(result.status()).toBe(400);
  } finally {
    await client.query("DELETE FROM scopes WHERE id='invalid-parent'");
    await client.end();
  }
});

test("grants and revocations survive a server restart without restoring bootstrap rights", async ({
  request,
}) => {
  const stored = await grant(
    request,
    agent,
    "authorization.grants.read",
    scope("PROJECT", "project-a"),
  );
  const bootstrap = await ok(request, "ListGrants", { scope: platform });
  const force = bootstrap.grants.find(
    (g: { action: string }) => g.action === "git.ref.force_update",
  );
  await revoke(request, force.id);
  const child = spawn(binary, [], {
    cwd: root,
    env: { ...serverEnv, HEPH_LISTEN_ADDR: "127.0.0.1:18081" },
    stdio: ["ignore", "pipe", "pipe"],
  });
  let log = "";
  child.stdout?.on("data", (chunk) => {
    log += chunk;
  });
  child.stderr?.on("data", (chunk) => {
    log += chunk;
  });
  try {
    await expect.poll(() => log, { timeout: 15_000 }).toContain("heph server listening");
    const response = await request.post(
      "http://127.0.0.1:18081/heph.authorization.v1.AuthorizationService/GetGrant",
      {
        data: { id: stored.id },
        headers: { Authorization: "Bearer " + admin.token },
      },
    );
    expect(response.status(), await response.text()).toBe(200);
    expect((await response.json()).grant.id).toBe(stored.id);
    const list = await request.post(
      "http://127.0.0.1:18081/heph.authorization.v1.AuthorizationService/ListGrants",
      {
        data: { scope: platform },
        headers: { Authorization: "Bearer " + admin.token },
      },
    );
    expect((await list.json()).grants.some((g: { id: string }) => g.id === force.id)).toBe(false);
  } finally {
    const exited = new Promise<void>((resolve) => child.once("exit", () => resolve()));
    child.kill("SIGTERM");
    await exited;
    await revoke(request, stored.id);
  }
});
