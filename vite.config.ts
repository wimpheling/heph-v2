import { defineConfig } from "vite-plus";

export default defineConfig({
  fmt: { ignorePatterns: ["packages/contracts/src/gen/**", "docs/**"] },
  lint: {
    plugins: ["react", "typescript", "oxc"],
    ignorePatterns: ["packages/contracts/src/gen/**", "tools/oxlint/fixtures/**"],
    options: { typeAware: true, typeCheck: true },
    jsPlugins: [{ name: "heph", specifier: "./tools/oxlint/plugin.mjs" }],
    rules: {
      "react/rules-of-hooks": "error",
      "heph/design-system-boundary": "error",
      "heph/rpc-boundary": "error",
    },
  },
});
