import { expect, test } from "@playwright/test";
import { spawn } from "node:child_process";
import pg from "pg";
import { databaseURL, schema, binary, root, serverEnv } from "../../harness/config.mjs";
import { admin, agent, actor, platform, scope, grant, revoke, rpc, ok } from "./helpers.js";

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
