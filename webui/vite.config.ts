// Purpose: Build the browser SPA and proxy development API requests to Go.
// Pattern: Build configuration.
// Usage: bun run dev/build from webui/.
// Related: webui/scripts/package.ts, internal/webui/assets.go.

import { fileURLToPath } from 'node:url';
import tailwindcss from '@tailwindcss/vite';
import react from '@vitejs/plugin-react';
import { defineConfig } from 'vite';
import { runtimeNotices } from '#build/notices';
export default defineConfig({
  base: '/ui/',
  plugins: [react(), tailwindcss(), runtimeNotices()],
  resolve: { alias: { '@': fileURLToPath(new URL('./src', import.meta.url)) } },
  server: { proxy: { '/v1': 'http://127.0.0.1:8787' } },
  build: { target: 'es2022', sourcemap: false, assetsInlineLimit: 0 },
});
