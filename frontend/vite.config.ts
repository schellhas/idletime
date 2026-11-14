import react from "@vitejs/plugin-react";
import { defineConfig } from "vite";

const apiTarget = process.env.VITE_API_PROXY_TARGET ?? "http://localhost:8080";

export default defineConfig({
  plugins: [react()],
  server: {
    port: 3000,
    host: true,
    proxy: { "/api": apiTarget },
  },
});
