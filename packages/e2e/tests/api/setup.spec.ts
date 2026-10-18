import { expect, test } from "@playwright/test";
import { spawnSync } from "node:child_process";
import pg from "pg";
import { databaseURL, schema, binary, root, serverEnv } from "../../harness/config.mjs";

test("migration and reset commands prepare an empty schema and are repeatable", async () => {
  const isolated = schema + "_cli";
  const env = { ...serverEnv, HEPH_DATABASE_SCHEMA: isolated };
  const command = (name: string) => spawnSync(binary, [name], { cwd: root, env, encoding: "utf8" });
  const db = new pg.Client({ connectionString: databaseURL });
  await db.connect();
  try {
    for (const name of ["migrate", "migrate"]) {
      const result = command(name);
      expect(result.status, result.stdout + result.stderr).toBe(0);
    }
    await db.query('SET search_path TO "' + isolated + '"');
    expect((await db.query("SELECT version FROM schema_migrations")).rows).toEqual([
      { version: 1 },
    ]);
    await db.query("INSERT INTO scopes(kind,id) VALUES('platform','platform')");
    const reset = command("reset");
    expect(reset.status, reset.stdout + reset.stderr).toBe(0);
    expect((await db.query("SELECT count(*)::int AS total FROM scopes")).rows[0].total).toBe(0);
    expect((await db.query("SELECT version FROM schema_migrations")).rows).toEqual([
      { version: 1 },
    ]);
  } finally {
    await db.query('DROP SCHEMA IF EXISTS "' + isolated + '" CASCADE');
    await db.end();
  }
});

test("server startup rejects invalid identity configuration before serving requests", () => {
  for (const identities of [
    "not-json",
    "[]",
    '[{"source":"test","subject":"actor","token":"short"}]',
  ]) {
    const result = spawnSync(binary, [], {
      cwd: root,
      env: { ...serverEnv, HEPH_BOOTSTRAP_IDENTITIES: identities },
      encoding: "utf8",
    });
    expect(result.status).toBe(1);
    expect(result.stdout).not.toContain("heph server listening");
  }
});
