import { spawn } from "node:child_process";
import { mkdir, readdir, writeFile } from "node:fs/promises";
import pg from "pg";
import { apiURL, binary, databaseURL, root, schema, serverEnv } from "./config.mjs";

function run(command, args) {
  return new Promise((resolve, reject) => {
    const child = spawn(command, args, { cwd: root, env: serverEnv, stdio: "inherit" });
    child.on("error", reject);
    child.on("exit", (code, signal) =>
      code === 0 ? resolve() : reject(new Error(command + " failed: " + (code ?? signal))),
    );
  });
}
const db = new pg.Client({ connectionString: databaseURL });
await db.connect();
let server;
let stopping = false;
async function cleanup(signal = "SIGTERM") {
  if (stopping) return;
  stopping = true;
  if (server && server.exitCode === null && server.signalCode === null) {
    const exited = new Promise((resolve) => server.once("exit", resolve));
    server.kill(signal);
    await exited;
  }
  if (server && process.env.HEPH_E2E_COVERAGE) {
    const files = await readdir(process.env.HEPH_E2E_COVERAGE);
    const flushed = files.some(
      (name) => name.startsWith("covcounters.") && name.includes("." + server.pid + "."),
    );
    await writeFile(
      process.env.HEPH_E2E_COVERAGE + "/server.json",
      JSON.stringify({ pid: server.pid, flushed }),
    );
    if (!flushed) {
      console.error("The E2E server did not finish writing coverage counters");
      process.exitCode = 1;
    }
  }
  // Only the randomly named schema owned by this test run is removed.
  if (/^e2e_[a-z0-9_]+$/.test(schema)) {
    await db.query('DROP SCHEMA IF EXISTS "' + schema + '" CASCADE');
  }
  await db.end();
}
process.on("SIGINT", () => void cleanup());
process.on("SIGTERM", () => void cleanup());
try {
  await mkdir(new URL("../../../bin/", import.meta.url), { recursive: true });
  const flags = process.env.HEPH_E2E_COVERAGE
    ? ["-cover", "-covermode=atomic", "-coverpkg=./cmd/heph,./internal/..."]
    : [];
  await run("go", ["build", ...flags, "-o", binary, "./cmd/heph"]);
  await run(binary, ["migrate"]);
  // Fixtures define resource ownership only. Every grant mutation is exercised through RPC.
  await db.query('SET search_path TO "' + schema + '"');
  await db.query(`
    INSERT INTO scopes(kind,id,parent_kind,parent_id,resource_kind) VALUES
      ('platform','platform',NULL,NULL,''),
      ('organization','acme','platform','platform',''),
      ('organization','other','platform','platform',''),
      ('project','project-a','organization','acme',''),
      ('project','project-b','organization','acme',''),
      ('resource','repo-a','project','project-a','repository'),
      ('resource','repo-b','project','project-b','repository'),
      ('resource','secret-a','project','project-a','secret')
    ON CONFLICT DO NOTHING;
    INSERT INTO scopes(kind,id,parent_kind,parent_id,ref_name) VALUES
      ('git_ref','branch-a','resource','repo-a','refs/heads/agents/builder'),
      ('git_ref','branch-main','resource','repo-a','refs/heads/main'),
      ('git_ref','branch-b','resource','repo-b','refs/heads/agents/builder')
    ON CONFLICT DO NOTHING;
  `);
  // Playwright signals the harness process group. The harness alone stops Go,
  // avoiding a second SIGTERM during the runtime's final coverage flush.
  server = spawn(binary, [], { cwd: root, env: serverEnv, stdio: "inherit", detached: true });
  server.on("error", (error) => {
    console.error(error.message);
    process.exitCode = 1;
    void cleanup();
  });
  server.on("exit", (code, signal) => {
    if (!stopping) {
      console.error("API server exited unexpectedly", { code, signal, apiURL });
      process.exitCode = 1;
      void cleanup();
    }
  });
} catch (error) {
  console.error(error.message);
  process.exitCode = 1;
  await cleanup();
}
