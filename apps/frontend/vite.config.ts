import react from "@vitejs/plugin-react";
import path from "node:path";
import { defineConfig } from "vite";

export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: {
      "@": path.resolve(__dirname, "./src"),
    },
  },
  server: {
    port: 3000,
    proxy: {
      "/paas-deploy": {
        target: "http://localhost:8081",
        changeOrigin: true,
        ws: true,
      },
      "/events": {
        target: "http://localhost:8081",
        changeOrigin: true,
        ws: true,
      },
      "/auth": {
        target: "http://localhost:8081",
        changeOrigin: true,
      },
      "/api": {
        target: "http://localhost:8081",
        changeOrigin: true,
      },
    },
  },
});
