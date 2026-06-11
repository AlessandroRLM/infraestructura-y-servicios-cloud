import { RouterProvider } from "@tanstack/react-router";
import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { Providers } from "./core/query/providers";
import { SessionProvider, stubSessionSource, useSession } from "./core/session";
import { router } from "./router";
import "./index.css";

/**
 * Inner component rendered inside SessionProvider.
 *
 * Reads the session from SessionCtx (already resolved by the provider) and
 * injects it into the router context via RouterProvider's context prop.
 *
 * The session query is declared ONCE inside SessionProvider. This component
 * only re-derives the value from context — no second fetch, no duplicated query.
 *
 * auth-and-guards Intent skill pattern: router created once (router.ts) with a
 * placeholder; live state is injected per render without recreating the router.
 */
function AppInner() {
  const session = useSession();
  return <RouterProvider router={router} context={{ session }} />;
}

/**
 * The stub source always returns unauthenticated. _authenticated routes are
 * unreachable until the real SessionSource (GetSession RPC) ships and replaces
 * stubSessionSource here.
 */
function App() {
  return (
    <SessionProvider source={stubSessionSource}>
      <AppInner />
    </SessionProvider>
  );
}

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <Providers>
      <App />
    </Providers>
  </StrictMode>,
);
