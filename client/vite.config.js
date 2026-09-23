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
  build: {
    // The `player` chunk is video.js, which is ~690 kB minified on its own and
    // cannot be meaningfully reduced. It is already split out of the app bundle
    // and loaded lazily with the watch screen, so the size is expected rather
    // than a regression. The default 500 kB limit produced a warning on every
    // build for a known, accepted cost, which trains people to ignore the report.
    chunkSizeWarningLimit: 750,
    rollupOptions: {
      output: {
        manualChunks: {
          vendor: ["react", "react-dom", "react-router-dom"],
          motion: ["framer-motion"],
          // `video.js` is the engine the NEXORA player (NexoraPlayer.jsx) uses.
          player: ["video.js"]
        }
      }
    }
  },
  server: {
    host: "0.0.0.0",
    port: 5173,
    // The browser talks to the same NEXORA hostname it was opened with.
    // Only Vite's server-to-server development proxy uses this internal URL.
    // Development can be opened locally or from a device on the LAN.
    allowedHosts: ["nexora.local", "localhost", "127.0.0.1", "192.168.1.33"],
    proxy: {
      "/api": apiUpstream,
      // TMDB images are cached by the Go server, not Vite. Proxying this
      // prefix fixes cached posters/backdrops during local development.
      "/assets": apiUpstream
    }
  }
});
