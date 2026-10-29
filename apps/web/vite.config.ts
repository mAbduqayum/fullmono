import react from "@vitejs/plugin-react";
import { defineConfig } from "vite";

// The api `just dev` runs on the host: in Docker, Caddy routes these paths instead.
const api = "http://localhost:8080";

export default defineConfig({
    plugins: [react()],
    server: {
        proxy: { "/api": api, "/r": api },
    },
});
