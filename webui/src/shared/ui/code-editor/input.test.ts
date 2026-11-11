// Purpose: Check exact UTF-8 admission for full-size documents and clipboard replacements.
// Pattern: Editor boundary regression tests.
// Usage: bun test src/shared/ui/code-editor/input.test.ts.
// Related: webui/src/shared/ui/code-editor/input.ts.
import { expect, test } from 'bun:test';
import { fitsEditorText } from '@/shared/ui/code-editor/input';

test('the complete two-MiB boundary counts UTF-8 rather than JavaScript characters', () => {
  const limit = 2 * 1024 * 1024;
  expect(fitsEditorText('x'.repeat(limit), limit)).toBe(true);
  expect(fitsEditorText('x'.repeat(limit + 1), limit)).toBe(false);
  const korean = `${'한'.repeat(Math.floor(limit / 3))}xx`;
  expect(fitsEditorText(korean, limit)).toBe(true);
  expect(fitsEditorText(`${korean}x`, limit)).toBe(false);
  expect(fitsEditorText('😀'.repeat(limit / 4), limit)).toBe(true);
  expect(fitsEditorText(`${'😀'.repeat(limit / 4)}x`, limit)).toBe(false);
  expect(fitsEditorText('', 0)).toBe(false);
});
