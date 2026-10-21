import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

const device = process.env.DEVICE ?? "https://localhost";

export default defineConfig({
  plugins: [react()],
  build: {
    outDir: "../pkg/runtime/webui/dist",
    emptyOutDir: true,
    chunkSizeWarningLimit: 1024,
    rollupOptions: {
      output: {
        entryFileNames: "assets/[name].js",
        chunkFileNames: "assets/[name].js",
        assetFileNames: "assets/[name][extname]",
      },
    },
  },
  server: {
    proxy: {
      "/api": { target: device, changeOrigin: true, secure: false, ws: true },
    },
  },
});
