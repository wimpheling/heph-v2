import { fileURLToPath } from "node:url";
import { defineConfig, devices } from "@playwright/test";
import { randomUUID } from "node:crypto";

process.env.HEPH_E2E_SCHEMA ??= "e2e_" + randomUUID().replaceAll("-", "");
const apiURL = process.env.HEPH_E2E_BASE_URL ?? "http://127.0.0.1:18080";
const apiOnly = process.env.HEPH_E2E_API_ONLY === "1";

const root = fileURLToPath(new URL("../../", import.meta.url));
const baseURL = "http://127.0.0.1:15173";

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
  webServer: process.env.HEPH_E2E_BASE_URL
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
                command: "pnpm --filter @heph/web dev --port 15173",
                cwd: root,
                env: { HEPH_RPC_TARGET: "http://127.0.0.1:18080" },
                url: baseURL,
                reuseExistingServer: false,
                timeout: 60_000,
                gracefulShutdown: { signal: "SIGTERM" as const, timeout: 5_000 },
              },
            ]
          : []),
      ],
});
