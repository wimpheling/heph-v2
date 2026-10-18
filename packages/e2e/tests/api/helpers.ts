import { expect, type APIRequestContext } from "@playwright/test";
import { identities, apiURL } from "../../harness/config.mjs";

export const admin = identities[0] as { source: string; subject: string; token: string };
export const alice = identities[1] as typeof admin;
export const agent = identities[2] as typeof admin;
export const other = identities[3] as typeof admin;
export const actor = (identity: typeof admin) => ({
  source: identity.source,
  subject: identity.subject,
});
export const scope = (kind: string, id: string) => ({ kind: "SCOPE_KIND_" + kind, id });
export const platform = scope("PLATFORM", "platform");

export async function rpc(
  request: APIRequestContext,
  method: string,
  data: unknown,
  token?: string,
) {
  return request.post(apiURL + "/heph.authorization.v1.AuthorizationService/" + method, {
    data,
    headers: {
      "Content-Type": "application/json",
      "Connect-Protocol-Version": "1",
      ...(token ? { Authorization: "Bearer " + token } : {}),
    },
  });
}
export async function ok(
  request: APIRequestContext,
  method: string,
  data: unknown,
  token = admin.token,
) {
  const response = await rpc(request, method, data, token);
  expect(response.status(), await response.text()).toBe(200);
  return response.json();
}
export async function grant(
  request: APIRequestContext,
  grantee: typeof admin,
  action: string,
  target: ReturnType<typeof scope>,
) {
  const result = await ok(request, "CreateGrant", {
    grantee: actor(grantee),
    action,
    scope: target,
  });
  return result.grant as {
    id: string;
    action: string;
    grantee: ReturnType<typeof actor>;
    grantedBy: ReturnType<typeof actor>;
    createdAt: string;
  };
}
export async function revoke(request: APIRequestContext, id: string) {
  return ok(request, "RevokeGrant", { id });
}
export async function denied(
  request: APIRequestContext,
  method: string,
  data: unknown,
  token: string,
  code = "permission_denied",
) {
  const response = await rpc(request, method, data, token);
  expect((await response.json()).code).toBe(code);
}
