import { fileURLToPath } from "node:url";
import { defineConfig, devices } from "@playwright/test";
import { booleanEnv, requiredEnv } from "../../scripts/environment.mjs";

// The runner allocates one schema and shares it with all workers and servers.
requiredEnv("HEPH_E2E_SCHEMA");
const apiURL = requiredEnv("HEPH_E2E_BASE_URL");
const apiOnly = booleanEnv("HEPH_E2E_API_ONLY");
const managed = booleanEnv("HEPH_E2E_MANAGED");
requiredEnv("HEPH_E2E_DATABASE_URL");

const root = fileURLToPath(new URL("../../", import.meta.url));
const baseURL = requiredEnv("HEPH_E2E_WEB_URL");

export default defineConfig({
  testDir: "./tests",
  fullyParallel: true,
  forbidOnly: Boolean(process.env.CI),
  retries: process.env.CI ? 2 : 0,
  workers: 1,
  reporter: [["list"], ["html", { open: "never" }]],
  use: {
    baseURL,
    trace: "retain-on-failure",
    screenshot: "only-on-failure",
  },
  projects: [
    { name: "api", retries: 0, testMatch: "**/api/**/*.spec.ts", use: { baseURL: apiURL } },
    ...(!apiOnly
      ? [
          {
            name: "chromium",
            testMatch: "**/bootstrap.spec.ts",
            use: { ...devices["Desktop Chrome"] },
          },
          {
            name: "mobile-chromium",
            testMatch: "**/bootstrap.spec.ts",
            use: { ...devices["Pixel 7"] },
          },
        ]
      : []),
  ],
  webServer: !managed
    ? []
    : [
        {
          name: "Go server",
          command: "node harness/start.mjs",
          // The listener is bound before the server emits this startup message.
          wait: { stdout: /heph server listening/ },
          reuseExistingServer: false,
          timeout: 120_000,
          gracefulShutdown: { signal: "SIGTERM", timeout: 15_000 },
        },
        ...(!apiOnly
          ? [
              {
                name: "React frontend",
                command: "pnpm --filter @heph/web dev --port " + new URL(baseURL).port,
                cwd: root,
                env: { HEPH_RPC_TARGET: apiURL },
                url: baseURL,
                reuseExistingServer: false,
                timeout: 60_000,
                gracefulShutdown: { signal: "SIGTERM" as const, timeout: 5_000 },
              },
            ]
          : []),
      ],
});
