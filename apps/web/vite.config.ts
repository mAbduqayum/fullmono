import react from "@vitejs/plugin-react";
import { defineConfig } from "vitest/config";

// The api `just dev` runs on the host: in Docker, Caddy routes these paths instead.
const api = "http://localhost:8080";

export default defineConfig({
    plugins: [react()],
    server: {
        proxy: { "/api": api, "/r": api },
    },
    // Vitest reads this same file, so the tests build the code as the app does.
    test: {
        // A browser's DOM, simulated in Node: fast, though no real layout or paint.
        environment: "jsdom",
        setupFiles: ["src/test/setup.ts"],
        // Undo every vi.stubGlobal (the tests' fake fetch) after each test.
        unstubGlobals: true,
    },
});
