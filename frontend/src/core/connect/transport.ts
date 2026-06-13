import { createConnectTransport } from "@connectrpc/connect-web";
import { env } from "@/core/config/env";

export const transport = createConnectTransport({
  baseUrl: env.VITE_API_URL,
  useBinaryFormat: import.meta.env.PROD,
  // Include cookies (httpOnly session) with every request
  fetch: (input, init) =>
    globalThis.fetch(input, { ...init, credentials: "include" }),
});
