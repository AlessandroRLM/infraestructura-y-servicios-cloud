import tailwindcss from "@tailwindcss/vite";
import { tanstackRouter } from "@tanstack/router-plugin/vite";
import react from "@vitejs/plugin-react";
import { resolve } from "path";
import { defineConfig } from "vite";

// NOTE: server.* options apply to the Vite dev server ONLY — they are not part of
// `vite build`, so none of this reaches the production image.
const apiProxyTarget =
  process.env.VITE_API_PROXY_TARGET ?? "http://localhost:8080";

export default defineConfig({
  plugins: [
    tanstackRouter({
      routesDirectory: "src/routes",
      generatedRouteTree: "src/routeTree.gen.ts",
      routeFileIgnorePattern: "(__tests__|.(test|spec)).(ts|tsx)?$",
    }),
    react({ babel: { plugins: ["babel-plugin-react-compiler"] } }),
    tailwindcss(),
  ],
  resolve: {
    alias: { "@": resolve(__dirname, "./src") },
  },
  server: {
    host: true,
    port: 5173,
    strictPort: true,
    watch:
      process.env.VITE_USE_POLLING === "1"
        ? {
            usePolling: true,
            ignored: [
              "**/vite.config.ts",
              "**/vite.config.ts.timestamp-*.mjs",
              "**/node_modules/**",
              "**/.git/**",
            ],
          }
        : undefined,
    proxy: {
      "^/[a-zA-Z0-9_]+\\.v1\\.": {
        target: apiProxyTarget,
        changeOrigin: true,
      },
    },
  },
});
