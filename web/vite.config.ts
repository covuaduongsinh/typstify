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
    rollupOptions: {
      output: {
        // CodeMirror and react-markdown rarely change between our own
        // commits, so they're split into their own chunks instead of
        // riding along inside whichever lazy component (Editor, AgentChat,
        // ...) happens to import them. That way a redeploy that only
        // touches app code keeps the browser's cached vendor chunk valid
        // (its content -- and therefore its hashed filename -- doesn't
        // change), instead of invalidating it every time.
        manualChunks(id) {
          if (/node_modules\/(codemirror|@codemirror|@lezer|codemirror-lang-typst)\//.test(id)) {
            return 'codemirror-vendor'
          }
          if (/node_modules\/(react-markdown|remark-|rehype-|mdast-|micromark|unist-|hast-|vfile)/.test(id)) {
            return 'markdown-vendor'
          }
        },
      },
    },
  },
  server: {
    proxy: {
      '/api': { target: backend, changeOrigin: true },
      '/preview': { target: backend, changeOrigin: true, ws: true },
      '/ws': { target: backend, changeOrigin: true, ws: true },
    },
  },
})
