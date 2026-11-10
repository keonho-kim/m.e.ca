// Purpose: Enforce frontend layer direction and public API boundaries.
// Pattern: Steiger FSD configuration.
// Usage: bun run architecture validates the src tree.
// Related: webui/src.

import fsd from '@feature-sliced/steiger-plugin';
import { defineConfig } from 'steiger';
export default defineConfig([
  ...fsd.configs.recommended,
  { ignores: ['**/shared/api/gen/**', '**/app/styles/generated/**'] },
]);
