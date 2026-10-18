import { spawnSync } from "node:child_process";
import { mkdtemp, readdir, readFile, rm } from "node:fs/promises";
import { tmpdir } from "node:os";
import { join, relative } from "node:path";
import { fileURLToPath } from "node:url";

const root = fileURLToPath(new URL("../", import.meta.url));
const generated = ["gen", "packages/contracts/src/gen", "api"];
const temporary = await mkdtemp(join(tmpdir(), "heph-contracts-"));
async function files(dir) {
  try {
    const entries = await readdir(dir, { withFileTypes: true });
    const groups = await Promise.all(
      entries.map(async (entry) => {
        const path = join(dir, entry.name);
        return entry.isDirectory() ? files(path) : [path];
      }),
    );
    return groups.flat();
  } catch (error) {
    if (error.code === "ENOENT") return [];
    throw error;
  }
}
try {
  const result = spawnSync("buf", ["generate", "--output", temporary], {
    cwd: root,
    stdio: "inherit",
  });
  if (result.error) throw result.error;
  if (result.status !== 0) throw new Error("Contract documentation or generation failed");
  const stale = [];
  for (const dir of generated) {
    const expected = await files(join(temporary, dir));
    const current = await files(join(root, dir));
    const names = new Set([
      ...expected.map((path) => relative(temporary, path)),
      ...current.map((path) => relative(root, path)),
    ]);
    for (const name of names) {
      try {
        const [a, b] = await Promise.all([
          readFile(join(temporary, name)),
          readFile(join(root, name)),
        ]);
        if (!a.equals(b)) stale.push(name);
      } catch (error) {
        if (error.code !== "ENOENT") throw error;
        stale.push(name);
      }
    }
  }
  if (stale.length)
    throw new Error("Run pnpm generate and commit these generated outputs:\n" + stale.join("\n"));
} finally {
  await rm(temporary, { recursive: true, force: true });
}
