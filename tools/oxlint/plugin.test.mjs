import assert from "node:assert/strict";
import { spawnSync } from "node:child_process";
import { mkdtempSync, mkdirSync, writeFileSync, rmSync } from "node:fs";
import { tmpdir } from "node:os";
import path from "node:path";
import { fileURLToPath } from "node:url";
import test from "node:test";
import { createRequire } from "node:module";

const root = fileURLToPath(new URL("../../", import.meta.url));
const require = createRequire(import.meta.url);
const viteRequire = createRequire(require.resolve("vite-plus/package.json"));
const oxlintBin = path.join(
  path.dirname(path.dirname(viteRequire.resolve("oxlint"))),
  "bin",
  "oxlint",
);

function lint(file, source) {
  const temporary = mkdtempSync(path.join(tmpdir(), "heph-oxlint-"));
  try {
    const target = path.join(temporary, file);
    mkdirSync(path.dirname(target), { recursive: true });
    writeFileSync(target, source);
    const config = path.join(temporary, ".oxlintrc.json");
    writeFileSync(
      config,
      JSON.stringify({
        categories: { correctness: "off" },
        jsPlugins: [{ name: "heph", specifier: path.join(root, "tools/oxlint/plugin.mjs") }],
        rules: { "heph/design-system-boundary": "error", "heph/rpc-boundary": "error" },
      }),
    );
    const result = spawnSync(process.execPath, [oxlintBin, "-c", config, target], {
      cwd: temporary,
      encoding: "utf8",
    });
    assert.ifError(result.error);
    return { code: result.status, output: result.stdout + result.stderr };
  } finally {
    rmSync(temporary, { recursive: true, force: true });
  }
}

void test("application components use the public design system", () => {
  const result = lint(
    "apps/web/src/Page.tsx",
    'import { Text } from "@heph/design-system"; export const Page = () => <Text>Hello</Text>;',
  );
  assert.equal(result.code, 0, result.output);
});

for (const source of [
  "export const Page = () => <button>Go</button>;",
  'export { Button } from "@astryxdesign/core/Button";',
  'import "./page.css";',
  'import "../../../packages/design-system/src/styles.css";',
  'void import("@astryxdesign/core/Button");',
]) {
  void test(`rejects UI boundary violation: ${source}`, () => {
    const result = lint("apps/web/src/Page.tsx", source);
    assert.equal(result.code, 1, result.output);
    assert.match(result.output, /design-system-boundary/);
  });
}

for (const source of [
  'import { createClient } from "@connectrpc/connect"; void createClient;',
  'export { SystemService } from "@heph/contracts/system";',
  'void fetch("/rpc");',
  'void window.fetch("/rpc");',
  'void import("@connectrpc/connect");',
  'const rpc = require("@connectrpc/connect"); void rpc;',
]) {
  void test(`rejects RPC boundary violation: ${source}`, () => {
    const result = lint("apps/web/src/Page.tsx", source);
    assert.equal(result.code, 1, result.output);
    assert.match(result.output, /rpc-boundary/);
  });
}

void test("design system owns Astryx and DOM", () => {
  const result = lint(
    "packages/design-system/src/Button.tsx",
    'import { Button } from "@astryxdesign/core/Button"; export const Demo = () => <div><Button label="Go" /></div>;',
  );
  assert.equal(result.code, 0, result.output);
});

void test("RPC adapter owns Connect imports", () => {
  const result = lint(
    "apps/web/src/app/rpc/system.ts",
    'import { createClient } from "@connectrpc/connect"; export const factory = createClient;',
  );
  assert.equal(result.code, 0, result.output);
});
