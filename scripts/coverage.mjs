import { spawnSync } from "node:child_process";
import { mkdirSync, readFileSync, writeFileSync, rmSync, readdirSync } from "node:fs";
import { resolve } from "node:path";
import { fileURLToPath } from "node:url";

const root = fileURLToPath(new URL("../", import.meta.url));
const coverage = resolve(root, "coverage");
function run(command, args, env = {}) {
  const result = spawnSync(command, args, {
    cwd: root,
    env: { ...process.env, ...env },
    stdio: "inherit",
  });
  if (result.error) throw result.error;
  if (result.status !== 0)
    throw new Error(command + " failed with " + (result.status ?? result.signal));
}
export function mergeProfiles(profiles) {
  const blocks = new Map();
  for (const profile of profiles) {
    const lines = profile.trim().split("\n");
    if (lines.shift() !== "mode: atomic") throw new Error("Expected atomic coverage mode");
    for (const line of lines) {
      const match = /^(.*) (\d+) (\d+)$/.exec(line);
      if (!match) throw new Error("Invalid coverage block: " + line);
      const key = match[1];
      const statements = Number(match[2]);
      const count = Number(match[3]);
      const previous = blocks.get(key);
      if (previous && previous.statements !== statements)
        throw new Error("Coverage instrumentation differs: " + key);
      blocks.set(key, { statements, count: (previous?.count ?? 0) + count });
    }
  }
  return (
    "mode: atomic\n" +
    [...blocks]
      .sort(([a], [b]) => a.localeCompare(b))
      .map(([key, b]) => key + " " + b.statements + " " + b.count)
      .join("\n") +
    "\n"
  );
}
function report(kind) {
  const dir = resolve(coverage, kind);
  const profile = readFileSync(resolve(dir, "coverage.out"), "utf8");
  let covered = 0,
    total = 0;
  for (const line of profile.trim().split("\n").slice(1)) {
    const match = /^(.*) (\d+) (\d+)$/.exec(line);
    if (!match) throw new Error("Invalid coverage profile");
    total += Number(match[2]);
    if (Number(match[3]) > 0) covered += Number(match[2]);
  }
  if (total === 0) throw new Error("Coverage report contains no application statements");
  const summary = {
    suite: kind,
    coveredStatements: covered,
    totalStatements: total,
    percent: Number(((100 * covered) / total).toFixed(2)),
  };
  writeFileSync(resolve(dir, "summary.json"), JSON.stringify(summary, null, 2) + "\n");
  run("go", [
    "tool",
    "cover",
    "-html=" + resolve(dir, "coverage.out"),
    "-o",
    resolve(dir, "index.html"),
  ]);
  run("go", ["tool", "cover", "-func=" + resolve(dir, "coverage.out")]);
  return summary;
}
function prepare(kind) {
  const dir = resolve(coverage, kind);
  rmSync(dir, { recursive: true, force: true });
  mkdirSync(dir, { recursive: true });
  return dir;
}
function unit() {
  const dir = prepare("unit");
  run("node", ["--test", "tools/oxlint/plugin.test.mjs", "scripts/coverage.test.mjs"]);
  run("go", [
    "test",
    "-race",
    "-count=1",
    "-covermode=atomic",
    "-coverpkg=./cmd/heph,./internal/...",
    "-coverprofile=" + resolve(dir, "coverage.out"),
    "./...",
  ]);
  report("unit");
}
function e2e() {
  const dir = prepare("e2e");
  const raw = resolve(dir, "raw");
  mkdirSync(raw);
  run("pnpm", ["test:e2e"], { HEPH_E2E_COVERAGE: raw });
  const manifest = JSON.parse(readFileSync(resolve(raw, "server.json"), "utf8"));
  if (!manifest.flushed) throw new Error("Main E2E server coverage is missing");
  if (readdirSync(raw).some((name) => name.startsWith("tmp.covcounters.")))
    throw new Error("A server coverage write was interrupted");
  if (!readdirSync(raw).some((name) => name.startsWith("covcounters.")))
    throw new Error("E2E server did not flush its coverage counters");
  run("go", ["tool", "covdata", "textfmt", "-i=" + raw, "-o=" + resolve(dir, "coverage.out")]);
  report("e2e");
}
function merge() {
  const dir = prepare("combined");
  const profiles = ["unit", "e2e"].map((kind) =>
    readFileSync(resolve(coverage, kind, "coverage.out"), "utf8"),
  );
  // Require identical source blocks so both reports use the same denominator.
  const keys = (profile) =>
    new Set(
      profile
        .trim()
        .split("\n")
        .slice(1)
        .map((line) => line.replace(/ \d+ \d+$/, "")),
    );
  const [unitKeys, e2eKeys] = profiles.map(keys);
  if (unitKeys.size !== e2eKeys.size || [...unitKeys].some((key) => !e2eKeys.has(key))) {
    throw new Error(
      "Unit and E2E profiles instrument different source blocks; rerun coverage from the same checkout",
    );
  }
  writeFileSync(resolve(dir, "coverage.out"), mergeProfiles(profiles));
  const combined = report("combined");
  const summaries = ["unit", "e2e"]
    .map((kind) => JSON.parse(readFileSync(resolve(coverage, kind, "summary.json"), "utf8")))
    .concat(combined);
  writeFileSync(resolve(coverage, "summary.json"), JSON.stringify(summaries, null, 2) + "\n");
  writeFileSync(
    resolve(coverage, "summary.md"),
    "# Go application coverage\n\n| Suite | Covered statements | Total statements | Coverage |\n| --- | ---: | ---: | ---: |\n" +
      summaries
        .map(
          (s) =>
            "| " +
            s.suite +
            " | " +
            s.coveredStatements +
            " | " +
            s.totalStatements +
            " | " +
            s.percent +
            "% |",
        )
        .join("\n") +
      "\n",
  );
}
if (process.argv[1] && resolve(process.argv[1]) === fileURLToPath(import.meta.url)) {
  mkdirSync(coverage, { recursive: true });
  const mode = process.argv[2] ?? "all";
  if (!["unit", "e2e", "merge", "all"].includes(mode))
    throw new Error("usage: coverage.mjs [unit|e2e|merge|all]");
  if (mode === "unit" || mode === "all") unit();
  if (mode === "e2e" || mode === "all") e2e();
  if (mode === "merge" || mode === "all") merge();
}
