import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

// The API checks the Origin header against the request Host, so the dev
// proxy must NOT rewrite Host (no changeOrigin). The browser's Origin is
// http://localhost:5173 and Host stays localhost:5173, which matches.
export default defineConfig({
  plugins: [vue()],
  build: { outDir: '../internal/web/dist', emptyOutDir: true },
  server: {
    port: 5173,
    proxy: {
      '/api': 'http://localhost:8080',
      '/bot': 'http://localhost:8080',
      '/ws': { target: 'ws://localhost:8080', ws: true },
    },
  },
})
