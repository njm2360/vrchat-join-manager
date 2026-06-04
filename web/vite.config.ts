import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'
import path from 'node:path'

// https://vite.dev/config/
export default defineConfig({
  base: './',
  plugins: [react(), tailwindcss()],
  resolve: {
    alias: {
      '@': path.resolve(__dirname, 'src'),
    },
  },
  server: {
    port: 5173,
    proxy: {
      '/api': 'http://localhost:8080',
      '/openapi.json': 'http://localhost:8080',
      '/docs': 'http://localhost:8080',
    },
  },
  build: {
    outDir: path.resolve(__dirname, '../server/static'),
    emptyOutDir: true,
    sourcemap: false,
    rolldownOptions: {
      output: {
        codeSplitting: {
          groups: [
            { name: 'mui-icons', test: /node_modules\/@mui\/icons-material\// },
            { name: 'mui-pickers', test: /node_modules\/@mui\/x-date-pickers\// },
            { name: 'mui-core', test: /node_modules\/(@mui|@emotion)\// },
            { name: 'router', test: /node_modules\/(react-router|react-router-dom)\// },
            { name: 'query', test: /node_modules\/@tanstack\// },
            { name: 'react', test: /node_modules\/(react|react-dom|scheduler)\// },
          ],
        },
      },
    },
  },
})
