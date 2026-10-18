import { createClient } from "@connectrpc/connect";
import { createConnectTransport } from "@connectrpc/connect-web";
import { AuthorizationService } from "@heph/contracts/authorization";

// The caller owns credential acquisition. The adapter does not persist tokens.
export function createAuthorizationAdapter(token: string) {
  const transport = createConnectTransport({
    baseUrl: `${window.location.origin}/rpc`,
    interceptors: [
      (next) => async (request) => {
        request.header.set("Authorization", `Bearer ${token}`);
        return next(request);
      },
    ],
  });
  return createClient(AuthorizationService, transport);
}
