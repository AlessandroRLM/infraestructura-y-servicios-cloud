import { createConnectTransport } from "@connectrpc/connect-web";

// Same-origin: the API is served from the page's own origin (the Ingress routes
// Connect paths to the api service). No build-time API URL needed.
export const transport = createConnectTransport({
  baseUrl: window.location.origin,
  useBinaryFormat: import.meta.env.PROD,
  // Include cookies (httpOnly session) with every request
  fetch: (input, init) =>
    globalThis.fetch(input, { ...init, credentials: "include" }),
});
