import path from "node:path";
import { fileURLToPath } from "node:url";

const clientRoot = path.dirname(fileURLToPath(import.meta.url)).replaceAll("\\", "/");

export default {
  plugins: {
    tailwindcss: {
      config: path.resolve(clientRoot, "tailwind.config.js")
    },
    autoprefixer: {}
  }
};
