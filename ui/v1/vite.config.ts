import { fileURLToPath } from 'node:url'
import { defineConfig } from 'vite'
import { svelte } from '@sveltejs/vite-plugin-svelte'
import tailwindcss from '@tailwindcss/vite'

// https://vite.dev/config/
// The app lives at the root, in the Vite dev server as well as served by the Go server
export default defineConfig({
  plugins: [svelte(), tailwindcss()],
  resolve: {
    // Generated from openapi/openapi.yaml by `make generate`
    alias: { 'ragbag-frontend-sdk': fileURLToPath(new URL('./src/lib/sdk/index.ts', import.meta.url)) },
  },
  // dist is embedded into the Go binary (ui/ui.go)
})
