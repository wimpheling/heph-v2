import { fileURLToPath } from "node:url";
import { defineConfig, devices } from "@playwright/test";

const root = fileURLToPath(new URL("../../", import.meta.url));
const baseURL = "http://127.0.0.1:15173";

export default defineConfig({
  testDir: "./tests",
  fullyParallel: true,
  forbidOnly: Boolean(process.env.CI),
  retries: process.env.CI ? 2 : 0,
  workers: process.env.CI ? 2 : undefined,
  reporter: [["list"], ["html", { open: "never" }]],
  use: {
    baseURL,
    trace: "retain-on-failure",
    screenshot: "only-on-failure",
  },
  projects: [
    { name: "chromium", use: { ...devices["Desktop Chrome"] } },
    { name: "mobile-chromium", use: { ...devices["Pixel 7"] } },
  ],
  webServer: [
    {
      name: "Go server",
      command: "go run ./cmd/heph",
      cwd: root,
      env: { HEPH_LISTEN_ADDR: "127.0.0.1:18080" },
      // The listener is bound before the server emits this startup message.
      wait: { stdout: /heph server listening/ },
      reuseExistingServer: false,
      timeout: 120_000,
      gracefulShutdown: { signal: "SIGINT", timeout: 5_000 },
    },
    {
      name: "React frontend",
      command: "pnpm --filter @heph/web dev --port 15173",
      cwd: root,
      env: { HEPH_RPC_TARGET: "http://127.0.0.1:18080" },
      url: baseURL,
      reuseExistingServer: false,
      timeout: 60_000,
      gracefulShutdown: { signal: "SIGTERM", timeout: 5_000 },
    },
  ],
});
