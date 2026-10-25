import { createHash } from 'node:crypto'
import { readdirSync, readFileSync } from 'node:fs'
import { join, relative } from 'node:path'
import react from '@vitejs/plugin-react'
import { defineConfig, type Plugin } from 'vite'
import { precacheList } from './src/lib/swStrategy.ts'

/** Files copied from public/ as they are (manifest, icons), relative. */
function publicFiles(dir: string): string[] {
  return readdirSync(dir, { withFileTypes: true, recursive: true })
    .filter((e) => e.isFile())
    .map((e) => relative(dir, join(e.parentPath, e.name)).replaceAll('\\', '/'))
}

/**
 * Builds src/sw/sw.ts to /sw.js (the site root, so it controls every page)
 * and injects the list of files to cache at install plus a version hashed
 * from the whole build (bundle and public/ contents): any change changes
 * the worker, which is what makes browsers install the new one.
 */
function serviceWorker(): Plugin {
  return {
    name: 'typstify-service-worker',
    apply: 'build',
    generateBundle(_, bundle) {
      const sw = bundle['sw.js']
      if (!sw || sw.type !== 'chunk') return
      // Registered as a classic script: it must be one self-contained file.
      if (sw.imports.length || sw.dynamicImports.length) {
        this.error(`sw.js imports other chunks (${[...sw.imports, ...sw.dynamicImports].join(', ')})`)
      }
      const publicDir = join(import.meta.dirname, 'public')
      const pub = publicFiles(publicDir)
      const files = precacheList(Object.keys(bundle), pub)
      const hash = createHash('sha256')
      for (const [name, out] of Object.entries(bundle).sort(([a], [b]) => a.localeCompare(b))) {
        if (name === 'sw.js') continue
        hash.update(name).update(out.type === 'chunk' ? out.code : out.source)
      }
      for (const f of pub) hash.update(f).update(readFileSync(join(publicDir, f)))
      hash.update(sw.code)
      const version = hash.digest('hex').slice(0, 12)
      sw.code = sw.code
        .replace(/self\.__PRECACHE_MANIFEST__/g, JSON.stringify(files))
        .replace(/self\.__SW_VERSION__/g, JSON.stringify(version))
      if (sw.code.includes('__PRECACHE_MANIFEST__') || sw.code.includes('__SW_VERSION__')) {
        this.error('service worker placeholders were not replaced')
      }
    },
  }
}

// During development, the Vite dev server proxies API/WS/preview requests to
// the Go backend (cmd/typstify-server, default :8080) so the frontend can be
// developed with `npm run dev` while pointing at a real backend, without
// dealing with CORS. In production the Go server serves the built dist/
// directly (see -static-dir), so this proxy config only matters for `dev`.
const backend = process.env.TYPSTIFY_BACKEND ?? 'http://127.0.0.1:8080'

export default defineConfig({
  plugins: [react(), serviceWorker()],
  build: {
    // The lazily loaded Editor chunk is CodeMirror plus the Typst grammar
    // (~515 kB, ~160 kB gzipped); the initial bundle is far smaller.
    chunkSizeWarningLimit: 600,
    rollupOptions: {
      input: {
        main: join(import.meta.dirname, 'index.html'),
        sw: join(import.meta.dirname, 'src/sw/sw.ts'),
      },
      output: {
        // The worker must sit at the root under a fixed name.
        entryFileNames: (chunk) => (chunk.name === 'sw' ? 'sw.js' : 'assets/[name]-[hash].js'),
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
          if (/node_modules\/(yjs|y-protocols|y-indexeddb|y-codemirror\.next|lib0)\//.test(id)) {
            return 'collab-vendor'
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
