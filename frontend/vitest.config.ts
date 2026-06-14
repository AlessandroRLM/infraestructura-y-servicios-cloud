import { defineConfig } from "vitest/config";
import react from "@vitejs/plugin-react";
import { resolve } from "path";

export default defineConfig({
  plugins: [react()],
  test: {
    environment: "happy-dom",
    globals: true,
    setupFiles: ["./src/core/test/setup.ts"],
    include: ["src/**/__tests__/**/*.test.{ts,tsx}"],
    alias: { "@": resolve(__dirname, "./src") },
  },
});
