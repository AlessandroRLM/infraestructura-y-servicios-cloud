import { RouterProvider } from "@tanstack/react-router";
import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { Providers } from "./core/query/providers";
import { SessionProvider } from "./features/auth";
import { router, sessionSource } from "./router";
import "./index.css";

function App() {
  return (
    <SessionProvider source={sessionSource}>
      <RouterProvider router={router} />
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
