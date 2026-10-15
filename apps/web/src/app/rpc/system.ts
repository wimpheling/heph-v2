import { createClient } from "@connectrpc/connect";
import { createConnectTransport } from "@connectrpc/connect-web";
import { SystemService } from "@heph/contracts/system";

const transport = createConnectTransport({
  baseUrl: `${window.location.origin}/rpc`,
});
const system = createClient(SystemService, transport);

export async function getSystemInfo(signal?: AbortSignal) {
  return system.getSystemInfo({}, { signal, timeoutMs: 5000 });
}
