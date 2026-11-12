// Purpose: Check exact UTF-8 admission for full-size documents and clipboard replacements.
// Pattern: Editor boundary regression tests.
// Usage: bun test src/shared/ui/code-editor/input.test.ts.
// Related: webui/src/shared/ui/code-editor/input.ts.
import { expect, test } from 'bun:test';
import { editorTextAdmission } from '@/shared/ui/code-editor/input';

test('the complete two-MiB boundary counts UTF-8 rather than JavaScript characters', () => {
  const limit = 2 * 1024 * 1024;
  const fitsEditorText = editorTextAdmission(limit);
  expect(fitsEditorText('x'.repeat(limit))).toBe(true);
  expect(fitsEditorText('x'.repeat(limit + 1))).toBe(false);
  const korean = `${'한'.repeat(Math.floor(limit / 3))}xx`;
  expect(fitsEditorText(korean)).toBe(true);
  expect(fitsEditorText(`${korean}x`)).toBe(false);
  expect(fitsEditorText('😀'.repeat(limit / 4))).toBe(true);
  expect(fitsEditorText(`${'😀'.repeat(limit / 4)}x`)).toBe(false);
  expect(editorTextAdmission(0)('')).toBe(false);
});
test('native chunk boundaries preserve surrogate pairs and replace isolated surrogates consistently', () => {
  const values = [`${'x'.repeat(65535)}😀`, `${'한'.repeat(21845)}😀`, '\ud800'.repeat(30000)];
  for (const text of values) {
    const bytes = new TextEncoder().encode(text).length;
    expect(editorTextAdmission(bytes)(text)).toBe(true);
    expect(editorTextAdmission(bytes - 1)(text)).toBe(false);
  }
});
