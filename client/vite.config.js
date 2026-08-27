import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import path from "node:path";
import { fileURLToPath } from "node:url";

const apiUpstream = process.env.NEXORA_API_UPSTREAM || "http://127.0.0.1:8080";
const clientRoot = fileURLToPath(new URL(".", import.meta.url));

export default defineConfig({
  plugins: [react()],
  root: clientRoot,
  css: {
    postcss: path.resolve(clientRoot, "postcss.config.js")
  },
  resolve: {
    dedupe: ["react", "react-dom"],
    alias: {
      react: path.resolve(clientRoot, "node_modules/react"),
      "react-dom": path.resolve(clientRoot, "node_modules/react-dom")
    }
  },
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
