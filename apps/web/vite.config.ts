import react from "@vitejs/plugin-react";
import { defineConfig, lazyPlugins } from "vite-plus";
import { requiredEnv } from "../../scripts/environment.mjs";

export default defineConfig(({ command }) => ({
  plugins: lazyPlugins(() => [react()]),
  server:
    command === "serve"
      ? {
          host: "127.0.0.1",
          port: 5173,
          strictPort: true,
          proxy: {
            "/rpc": {
              target: requiredEnv("HEPH_RPC_TARGET"),
              rewrite: (path) => path.replace(/^\/rpc/, ""),
            },
          },
        }
      : undefined,
}));
