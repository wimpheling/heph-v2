import { spawn } from "node:child_process";

const windows = process.platform === "win32";
const children = [
  spawn("go", ["run", "./cmd/heph"], { stdio: "inherit", detached: !windows }),
  spawn(windows ? "pnpm.cmd" : "pnpm", ["--filter", "@heph/web", "dev"], {
    stdio: "inherit",
    detached: !windows,
  }),
];
let stopping = false;
function stop(code) {
  if (stopping) return;
  stopping = true;
  process.exitCode = code;
  for (const child of children) {
    if (!child.pid) continue;
    if (windows) {
      spawn("taskkill", ["/pid", String(child.pid), "/T", "/F"], { stdio: "ignore" });
    } else {
      try {
        // Terminate the entire process group, including package-manager children.
        process.kill(-child.pid, "SIGTERM");
      } catch (error) {
        if (error.code !== "ESRCH") throw error;
      }
    }
  }
}
for (const child of children) {
  child.on("error", (error) => {
    console.error(error);
    stop(1);
  });
  child.on("exit", (code) => stop(code ?? 1));
}
process.on("SIGINT", () => stop(0));
process.on("SIGTERM", () => stop(0));
