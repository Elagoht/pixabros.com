import babel from "@rolldown/plugin-babel";
import react, { reactCompilerPreset } from "@vitejs/plugin-react";
import path from "path";
import { defineConfig } from "vite";

// Where the Go server listens in development (PIXABROS_ADDR defaults to :8080).
const API_TARGET = process.env.VITE_DEV_API_TARGET ?? "http://localhost:8080";

// https://vite.dev/config/
export default defineConfig({
	// The panel is served under this prefix, and the built asset URLs have to
	// say so: without it Vite emits /assets/..., which collides with the public
	// site's own /assets mount and the panel loads nothing. It also feeds
	// import.meta.env.BASE_URL, which the router's basename reads.
	base: "/I-am-a-pixabro/",
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
