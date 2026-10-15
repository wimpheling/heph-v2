import react from "@vitejs/plugin-react";
import { defineConfig, lazyPlugins } from "vite-plus";

export default defineConfig({
  plugins: lazyPlugins(() => [react()]),
  server: {
    host: "127.0.0.1",
    port: 5173,
    strictPort: true,
    proxy: {
      "/rpc": {
        target: process.env.HEPH_RPC_TARGET ?? "http://127.0.0.1:8080",
        rewrite: (path) => path.replace(/^\/rpc/, ""),
      },
    },
  },
});
