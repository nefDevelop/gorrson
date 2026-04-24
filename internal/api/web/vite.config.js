import { defineConfig } from "vite";
import { svelte } from "@sveltejs/vite-plugin-svelte";

const backendUrl = "http://localhost:7077";
const apiRoutes = [
  "/news",
  "/feeds",
  "/filters",
  "/login",
  "/api-keys",
  "/stats",
  "/settings",
  "/export",
  "/import",
  "/backup",
  "/restore",
  "/proxy",
  "/recipes",
  "/categories",
];
const proxy = {};
apiRoutes.forEach((route) => (proxy[route] = backendUrl));

// https://vite.dev/config/
export default defineConfig({
  plugins: [svelte()],
  server: {
    proxy,
  },
});
