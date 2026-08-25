import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

const apiUpstream = process.env.NEXORA_API_UPSTREAM || "http://127.0.0.1:8080";

export default defineConfig({
  plugins: [react()],
  server: {
    host: "0.0.0.0",
    port: 5173,
    // The browser talks to the same NEXORA hostname it was opened with.
    // Only Vite's server-to-server development proxy uses this internal URL.
    allowedHosts: ["nexora.local"],
    proxy: {
      "/api": apiUpstream,
      // TMDB images are cached by the Go server, not Vite. Proxying this
      // prefix fixes cached posters/backdrops during local development.
      "/assets": apiUpstream
    }
  }
});
