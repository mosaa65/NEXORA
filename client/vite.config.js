import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

export default defineConfig({
  plugins: [react()],
  server: {
    host: "0.0.0.0",
    port: 5173,
    proxy: {
      "/api": "http://127.0.0.1:8080",
      // TMDB images are cached by the Go server, not Vite. Proxying this
      // prefix fixes cached posters/backdrops during local development.
      "/assets": "http://127.0.0.1:8080"
    }
  }
});
