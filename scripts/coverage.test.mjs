import { test } from "node:test";
import assert from "node:assert/strict";
import { mergeProfiles } from "./coverage.mjs";
void test("combined coverage preserves disjoint execution without averaging percentages", () => {
  const a = "mode: atomic\napp/a.go:1.1,2.1 2 1\napp/a.go:3.1,4.1 1 0\n";
  const b = "mode: atomic\napp/a.go:1.1,2.1 2 0\napp/a.go:3.1,4.1 1 3\n";
  assert.equal(mergeProfiles([a, b]), "mode: atomic\napp/a.go:1.1,2.1 2 1\napp/a.go:3.1,4.1 1 3\n");
  assert.throws(() => mergeProfiles([a, a.replace("2 1", "9 1")]), /instrumentation differs/);
  assert.throws(() => mergeProfiles(["mode: set\n"]), /atomic/);
});
