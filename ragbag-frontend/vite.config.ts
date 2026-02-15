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
  }
})
