import assert from "node:assert/strict";
import { test } from "node:test";
import { booleanEnv, requiredEnv } from "./environment.mjs";

void test("environment settings must be explicit, non-empty, and never exposed in errors", (t) => {
  const name = "HEPH_ENVIRONMENT_TEST";
  const previous = process.env[name];
  t.after(() => {
    if (previous === undefined) delete process.env[name];
    else process.env[name] = previous;
  });
  for (const value of [undefined, "", " \t\n"]) {
    if (value === undefined) delete process.env[name];
    else process.env[name] = value;
    assert.throws(() => requiredEnv(name), { message: name + " must be set to a non-empty value" });
  }
  process.env[name] = "explicit-sensitive-value";
  assert.equal(requiredEnv(name), "explicit-sensitive-value");
  assert.throws(() => booleanEnv(name), { message: name + " must be set to 0 or 1" });
  process.env[name] = "0";
  assert.equal(booleanEnv(name), false);
  process.env[name] = "1";
  assert.equal(booleanEnv(name), true);
});
