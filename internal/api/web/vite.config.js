import { defineConfig } from 'vite'
import { svelte } from '@sveltejs/vite-plugin-svelte'

// https://vite.dev/config/
export default defineConfig({
  plugins: [svelte()],
  server: {
    proxy: {
      '/news': 'http://localhost:7077',
      '/feeds': 'http://localhost:7077',
      '/filters': 'http://localhost:7077',
      '/login': 'http://localhost:7077',
      '/api-keys': 'http://localhost:7077',
      '/stats': 'http://localhost:7077',
      '/settings': 'http://localhost:7077',
      '/export': 'http://localhost:7077',
      '/import': 'http://localhost:7077',
      '/backup': 'http://localhost:7077',
      '/restore': 'http://localhost:7077',
    }
  }
})
