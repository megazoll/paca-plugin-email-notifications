import { defineConfig } from "vite";

export default defineConfig({
  build: {
    lib: {
      entry: "./src/index.ts",
      formats: ["es"],
      fileName: () => "mcp.js",
    },
    target: "es2022",
    minify: false,
    rollupOptions: {
      external: [],
    },
  },
});
