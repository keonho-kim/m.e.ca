// Purpose: Compile the browser SPA, bound initial chunks and emit optional bundle analysis.
// Pattern: Build configuration.
// Usage: bun run dev/build from webui/.
// Related: webui/scripts/package.ts, internal/webui/assets.go.

import { fileURLToPath } from 'node:url';
import babel from '@rolldown/plugin-babel';
import tailwindcss from '@tailwindcss/vite';
import react, { reactCompilerPreset } from '@vitejs/plugin-react';
import { visualizer } from 'rollup-plugin-visualizer';
import { defineConfig } from 'vite';
import { runtimeNotices } from '#build/notices';
export default defineConfig(({ mode }) => ({
  base: '/ui/',
  plugins: [
    react(),
    babel({ presets: [reactCompilerPreset({ target: '19' })] }),
    tailwindcss(),
    runtimeNotices(),
    mode === 'analyze' &&
      visualizer({
        filename: '.cache/bundle-analysis.html',
        gzipSize: true,
        open: false,
      }),
  ],
  resolve: { alias: { '@': fileURLToPath(new URL('./src', import.meta.url)) } },
  server: { proxy: { '/v1': 'http://127.0.0.1:8787' } },
  build: {
    target: 'es2022',
    sourcemap: false,
    assetsInlineLimit: 0,
    rolldownOptions: {
      output: {
        codeSplitting: {
          groups: [
            {
              name: 'react',
              test: /node_modules\/(?:react|react-dom|scheduler)\//,
              tags: ['$initial'],
              priority: 1,
            },
            {
              name: 'initial-deps',
              test: /node_modules/,
              // Merge only the dependency closure already required by the application entry.
              tags: ['$initial'],
            },
          ],
        },
      },
    },
  },
}));
