import { fileURLToPath } from "node:url";

export const root = fileURLToPath(new URL("../../../", import.meta.url));
export const databaseURL =
  process.env.HEPH_E2E_DATABASE_URL ??
  "postgresql://heph:heph-local-placeholder@127.0.0.1:5432/heph?sslmode=disable";
export const apiURL = process.env.HEPH_E2E_BASE_URL ?? "http://127.0.0.1:18080";
export const webURL = process.env.HEPH_E2E_WEB_URL ?? "http://127.0.0.1:15173";
export const schema = process.env.HEPH_E2E_SCHEMA ?? "heph_e2e";
export const binary = fileURLToPath(new URL("../../../bin/heph-e2e", import.meta.url));
// Synthetic identities for the isolated test installation, never development credentials.
export const identities = [
  { source: "e2e", subject: "admin", token: "e2e-admin-token-placeholder-00000001" },
  { source: "e2e", subject: "alice", token: "e2e-alice-token-placeholder-00000002" },
  { source: "e2e", subject: "agent", token: "e2e-agent-token-placeholder-00000003" },
  { source: "other-source", subject: "agent", token: "e2e-other-token-placeholder-00000004" },
];
export const serverEnv = {
  ...process.env,
  HEPH_DATABASE_URL: databaseURL,
  HEPH_DATABASE_SCHEMA: schema,
  HEPH_BOOTSTRAP_IDENTITIES: JSON.stringify(identities),
  HEPH_LISTEN_ADDR: new URL(apiURL).host,
  ...(process.env.HEPH_E2E_COVERAGE ? { GOCOVERDIR: process.env.HEPH_E2E_COVERAGE } : {}),
};
