import { test, expect } from "@playwright/test";
import { execFile } from "node:child_process";
import { promisify } from "node:util";
import { randomUUID } from "node:crypto";
import pg from "pg";
import { binary, databaseURL, root, serverEnv } from "../../harness/config.mjs";

const execute = promisify(execFile);

test("server startup rejects every absent or empty required setting without a fallback", async () => {
  for (const name of [
    "HEPH_LISTEN_ADDR",
    "HEPH_DATABASE_URL",
    "HEPH_DATABASE_SCHEMA",
    "HEPH_BOOTSTRAP_IDENTITIES",
  ]) {
    for (const value of [undefined, "", " \t"]) {
      const env = { ...serverEnv };
      if (value === undefined) delete env[name];
      else env[name] = value;
      let failure: unknown;
      try {
        await execute(binary, [], { cwd: root, env, timeout: 5_000 });
      } catch (error) {
        failure = error;
      }
      expect(failure).toMatchObject({ code: 1 });
      const output = (failure as { stderr: string }).stderr;
      expect(output).toContain(name + " must be set");
      expect(output).not.toContain("e2e-admin-token-placeholder");
      expect(output).not.toContain(databaseURL);
    }
  }
});

test("migration and reset commands require only explicit database settings", async () => {
  const schema = "e2e_config_" + randomUUID().replaceAll("-", "");
  const env: NodeJS.ProcessEnv = { ...serverEnv, HEPH_DATABASE_SCHEMA: schema };
  delete env.HEPH_LISTEN_ADDR;
  delete env.HEPH_BOOTSTRAP_IDENTITIES;
  const client = new pg.Client({ connectionString: databaseURL });
  await client.connect();
  try {
    for (const command of ["migrate", "reset"]) {
      await execute(binary, [command], { cwd: root, env, timeout: 15_000 });
      const result = await client.query("SELECT to_regclass($1) AS name", [schema + ".grants"]);
      expect(result.rows[0].name).not.toBeNull();
    }
  } finally {
    await client.query('DROP SCHEMA IF EXISTS "' + schema + '" CASCADE');
    await client.end();
  }
});
