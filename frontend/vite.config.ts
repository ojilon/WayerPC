import { readFileSync } from "node:fs";
import { join, dirname } from "node:path";
import { fileURLToPath } from "node:url";
import { defineConfig } from "vite";

const root = join(dirname(fileURLToPath(import.meta.url)), "..");
const ver = JSON.parse(readFileSync(join(root, "version.json"), "utf-8"));

// Wails serves frontend/dist in production; `vite dev` (browser or wails dev)
// serves from here with the mock backend when window.go is absent.
export default defineConfig({
  base: "./",
  build: { outDir: "dist", emptyOutDir: true, target: "es2020" },
  server: { port: 5173, strictPort: true },
  define: {
    __APP_VERSION__: JSON.stringify(ver.version),
    __APP_NAME__: JSON.stringify(ver.appName),
  },
});
