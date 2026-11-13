// Purpose: Preserve exact source and surrogate boundaries as code pages grow or are replaced.
// Pattern: Incremental page-index contract.
// Usage: bun test src/shared/ui/code-preview/pages.test.ts.
// Related: webui/src/shared/ui/code-preview/pages.ts.
import { expect, test } from 'bun:test';
import { codePages } from '@/shared/ui/code-preview/pages';

test('append and replacement expose every code unit without splitting surrogate pairs', () => {
  const index = codePages();
  for (const value of [
    'x'.repeat(8191) + '\ud83d',
    'x'.repeat(8191) + '😀',
    'x'.repeat(8191) + '😀' + '\n'.repeat(700),
    'replacement',
    '',
  ]) {
    const positions = index(value);
    const pages = positions.slice(1).map((end, part) => value.slice(positions[part], end));
    expect(pages.join('')).toBe(value);
    for (const page of pages)
      if (page) expect(page.charCodeAt(0) >= 0xdc00 && page.charCodeAt(0) <= 0xdfff).toBe(false);
  }
});
