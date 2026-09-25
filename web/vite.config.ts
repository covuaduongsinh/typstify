import react from '@vitejs/plugin-react'
import { defineConfig } from 'vite'

// During development, the Vite dev server proxies API/WS/preview requests to
// the Go backend (cmd/typstify-server, default :8080) so the frontend can be
// developed with `npm run dev` while pointing at a real backend, without
// dealing with CORS. In production the Go server serves the built dist/
// directly (see -static-dir), so this proxy config only matters for `dev`.
const backend = process.env.TYPSTIFY_BACKEND ?? 'http://127.0.0.1:8080'

export default defineConfig({
  plugins: [react()],
  build: {
    // The lazily loaded Editor chunk is CodeMirror plus the Typst grammar
    // (~515 kB, ~160 kB gzipped); the initial bundle is far smaller.
    chunkSizeWarningLimit: 600,
  },
  server: {
    proxy: {
      '/api': { target: backend, changeOrigin: true },
      '/preview': { target: backend, changeOrigin: true, ws: true },
      '/ws': { target: backend, changeOrigin: true, ws: true },
    },
  },
})
