/** @type {import('tailwindcss').Config} */
import path from "node:path";
import { fileURLToPath } from "node:url";

const clientRoot = path.dirname(fileURLToPath(import.meta.url));

export default {
  content: [
    path.join(clientRoot, "index.html"),
    path.join(clientRoot, "src/**/*.{js,jsx}")
  ],
  theme: {
    extend: {
      colors: {
        charcoal: "#080A12",
        royal: "#5A32F4",
        electric: "#19B7FF",
        glass: "rgba(255, 255, 255, 0.08)"
      },
      fontFamily: {
        display: ['"Space Grotesk"', '"Cairo"', "sans-serif"],
        body: ['"Cairo"', '"Space Grotesk"', "sans-serif"]
      },
      boxShadow: {
        neon: "0 0 0 1px rgba(25, 183, 255, 0.18), 0 0 48px rgba(90, 50, 244, 0.22)",
        panel: "0 24px 80px rgba(0, 0, 0, 0.5)"
      }
    }
  },
  plugins: []
};
