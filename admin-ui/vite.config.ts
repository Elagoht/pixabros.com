import babel from "@rolldown/plugin-babel";
import react, { reactCompilerPreset } from "@vitejs/plugin-react";
import path from "path";
import { defineConfig } from "vite";

// The Go server serves the built SPA from /I-am-a-pixabro/, so assets and the
// react-router basename both have to be prefixed with it.
const BASE = "/I-am-a-pixabro/";

// Where the Go server listens in development (PIXABROS_ADDR defaults to :8080).
const API_TARGET = process.env.VITE_DEV_API_TARGET ?? "http://localhost:8080";

// https://vite.dev/config/
export default defineConfig({
  base: BASE,
  plugins: [react(), babel({ presets: [reactCompilerPreset()] })],
  resolve: {
    alias: {
      "@": path.resolve(import.meta.dirname, "src"),
    },
  },
  server: {
    // Proxying keeps the browser on a single origin in development too, which
    // the SameSite=Strict session cookie requires.
    proxy: {
      "/api": { target: API_TARGET, changeOrigin: true },
      "/media": { target: API_TARGET, changeOrigin: true },
    },
  },
});
