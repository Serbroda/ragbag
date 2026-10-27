import { fileURLToPath } from 'node:url'
import { defineConfig, loadEnv } from 'vite'
import { svelte } from '@sveltejs/vite-plugin-svelte'
import tailwindcss from '@tailwindcss/vite'

// https://vite.dev/config/
export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), '')
  const appBase = env.VITE_APP_BASE
  const base = appBase
    ? appBase.endsWith('/')
      ? appBase
      : `${appBase}/`
    : '/'

  return {
    base,
    plugins: [svelte(), tailwindcss()],
    resolve: {
      // Generated from openapi/openapi.yaml by `make generate`
      alias: { 'ragbag-frontend-sdk': fileURLToPath(new URL('./src/lib/sdk/index.ts', import.meta.url)) },
    },
    // dist is embedded into the Go binary (ui/ui.go)
  }
})
