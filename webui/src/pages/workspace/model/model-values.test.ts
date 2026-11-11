// Purpose: Preserve model form parsing while using the smaller shared Zod implementation.
// Pattern: Boundary contracts for trimming, URL parsing and private credential limits.
// Usage: bun test src/pages/workspace/model/model-values.test.ts.
// Related: webui/src/pages/workspace/model/model-values.ts, webui/src/pages/workspace/ui/settings/model-form.tsx.
import { expect, test } from 'bun:test';
import { modelSchema } from '@/pages/workspace/model/model-values';

const valid = {
  provider: 'ollama',
  endpoint: 'http://127.0.0.1:11434',
  model: ' model-id ',
  credential: '',
  thinking: 'default',
  effort: '',
};
test('model forms keep their parsed ID and accept a bounded private API key', () => {
  expect(modelSchema.parse(valid).model).toBe('model-id');
  expect(modelSchema.safeParse({ ...valid, credential: 'k'.repeat(2048) }).success).toBe(true);
});
test('invalid endpoints, blank model IDs and unsafe credential characters cannot reach a save', () => {
  for (const patch of [
    { endpoint: 'not a URL' },
    { provider: '' },
    { model: '  ' },
    { credential: 'k'.repeat(2049) },
    { credential: 'key\nvalue' },
    { credential: 'key\rvalue' },
    { credential: 'key\0value' },
  ])
    expect(modelSchema.safeParse({ ...valid, ...patch }).success).toBe(false);
});
