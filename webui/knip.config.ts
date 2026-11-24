// Purpose: Find unused frontend files, exports and dependencies from real runtime and tooling entries.
// Pattern: Project-graph analysis configuration.
// Usage: bun run deadcode checks the complete WebUI workspace.
// Related: webui/package.json, webui/vite.config.ts, webui/src/app/styles/theme.ts.
import type { KnipConfig } from 'knip';

const config: KnipConfig = {
  entry: ['src/app/styles/theme.ts', 'src/**/*.test.ts', 'e2e/**/*.spec.ts', 'steiger.config.ts'],
  project: ['src/**/*.{ts,tsx,css}', 'scripts/**/*.ts', 'e2e/**/*.ts', 'tests/**/*.ts', '*.ts'],
  ignore: ['src/shared/api/gen/**'],
  // protoc is a system tool; its --plugin flag loads this npm executable.
  ignoreBinaries: ['protoc'],
  // Astryx Doctor loads the TypeScript 6 API while application checks use TypeScript 7.
  ignoreDependencies: ['@bufbuild/protoc-gen-es', '@typescript/typescript6'],
};
export default config;
