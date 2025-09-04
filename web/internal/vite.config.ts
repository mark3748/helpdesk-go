import { defineConfig } from 'vitest/config'
import react from '@vitejs/plugin-react'
import process from 'node:process'

// Dev server with API proxy. Target can be overridden via VITE_API_TARGET.
const apiTarget = process.env.VITE_API_TARGET || 'http://localhost:8080'

export default defineConfig({
  plugins: [react()],
  server: {
    host: true,
    port: 5173,
    proxy: {
      '/api': {
        target: apiTarget,
        changeOrigin: true,
        // Keep the /api prefix so the backend sees /api routes directly
        // (backend mounts all routes under /api).
        // No rewrite needed.
      },
    },
  },
  test: {
    environment: 'jsdom',
    globals: true,
    setupFiles: 'vitest.setup.ts'
  }
})
