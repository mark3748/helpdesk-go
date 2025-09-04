import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';
import tailwindcss from 'tailwindcss';
import autoprefixer from 'autoprefixer';
// Avoid importing Node types to keep Dockerfile using npm ci.
// Vite runs this file in Node, so process is available at runtime.
// We declare it to satisfy TypeScript without @types/node.
// eslint-disable-next-line @typescript-eslint/no-unused-vars
declare const process: any;

// https://vite.dev/config/
// Dev server with API proxy. Target can be overridden via VITE_API_TARGET.
const apiTarget = (process?.env?.VITE_API_TARGET as string) || 'http://api:8080';

export default defineConfig({
  plugins: [react()],
  css: {
    postcss: {
      plugins: [tailwindcss(), autoprefixer()],
    },
  },
  server: {
    host: true,
    port: 5174,
    proxy: {
      '/api': {
        target: apiTarget,
        changeOrigin: true,
      },
    },
  },
});
