import { spawnSync } from "node:child_process";

// Keep generated bindings and human-maintained docs outside the formatter.
const result = spawnSync(
  "gofmt",
  [process.argv.includes("--write") ? "-w" : "-l", "cmd", "internal", "tools/go-lint"],
  { encoding: "utf8" },
);
if (result.error) throw result.error;
if (result.stderr) process.stderr.write(result.stderr);
if (result.status !== 0) process.exit(result.status ?? 1);
if (result.stdout.trim()) {
  console.error(`Go formatting issues:\n${result.stdout}Run pnpm fmt to fix them.`);
  process.exitCode = 1;
}
