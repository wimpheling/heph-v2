import { expect, test } from "@playwright/test";

const systemInfoPath = "/rpc/heph.system.v1.SystemService/GetSystemInfo";
const connectionMessage = /Connecting to the Rust server|Connected to heph-v2|Server unavailable/;
const connected = /^Connected to heph-v2 · v\d+\.\d+\.\d+/;

test("connects to Rust and checks the connection again", async ({ page }) => {
  await page.goto("/");

  await expect(page).toHaveTitle("heph-v2 · Git forge + agent runtime");
  await expect(page.getByRole("heading", { level: 1 })).toHaveText(
    "A home for code and the agents that build it.",
  );
  await expect(page.getByRole("status").filter({ hasText: connectionMessage })).toHaveText(
    connected,
  );

  const response = page.waitForResponse(
    (response) =>
      new URL(response.url()).pathname === systemInfoPath && response.request().method() === "POST",
  );
  await page.getByRole("button", { name: "Check connection" }).click();
  expect((await response).status()).toBe(200);
  await expect(page.getByRole("status").filter({ hasText: connectionMessage })).toHaveText(
    connected,
  );

  const overflows = await page.evaluate(
    () => document.documentElement.scrollWidth > window.innerWidth,
  );
  expect(overflows).toBe(false);
});

test("shows pending and unavailable states, then recovers on retry", async ({ page }) => {
  const failure = Promise.withResolvers<void>();
  await page.route(`**${systemInfoPath}`, async (route) => {
    await failure.promise;
    await route.abort("connectionfailed");
  });

  await page.goto("/");
  const status = page.getByRole("status").filter({ hasText: connectionMessage });
  const retry = page.getByRole("button", { name: "Check connection" });
  await expect(status).toHaveText("Connecting to the Rust server…");
  await expect(retry).toBeDisabled();

  failure.resolve();
  await expect(status).toHaveText("Server unavailable. Start the Rust server and retry.");
  await expect(retry).toBeEnabled();

  await page.unrouteAll({ behavior: "wait" });
  await retry.click();
  await expect(status).toHaveText(connected);
});
