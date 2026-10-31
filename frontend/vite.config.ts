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
  // host 0.0.0.0 + allowedHosts: the dev server is also used from the Arena
  // sandbox preview (a proxied domain), not just localhost. Dev only —
  // `vite build` output is embedded in the desktop app.
  server: { port: 5173, strictPort: true, host: "0.0.0.0", allowedHosts: true },
  define: {
    __APP_VERSION__: JSON.stringify(ver.version),
    __APP_NAME__: JSON.stringify(ver.appName),
  },
});
