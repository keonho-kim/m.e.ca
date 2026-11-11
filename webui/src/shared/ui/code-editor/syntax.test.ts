// Purpose: Keep highlighting work bounded while allowing complete large-document editing.
// Pattern: Source budget boundary tests.
// Usage: bun test src/shared/ui/code-editor/syntax.test.ts.
// Related: webui/src/shared/ui/code-editor/syntax.ts.
import { expect, test } from 'bun:test';
import { canHighlight, syntaxTokens } from '@/shared/ui/code-editor/syntax';

test('ordinary source permits highlighting and pathological line counts or lengths do not', () => {
  expect(canHighlight('const value = "한글 😀";\n')).toBe(true);
  expect(canHighlight('// short\n'.repeat(2001))).toBe(false);
  expect(canHighlight('x'.repeat(8193))).toBe(false);
  expect(canHighlight('// code line\n'.repeat(7000))).toBe(false);
});

test('dense source stops in a streaming batch instead of allocating the whole token set', async () => {
  const signal = new AbortController().signal;
  expect(
    await syntaxTokens(`${'{}[]();'.repeat(10)}\n`.repeat(800), 'javascript', signal)
  ).toBeUndefined();
  const lines = await syntaxTokens('const value = 42;', 'javascript', signal);
  expect(lines?.flat().some((token) => token.type === 'keyword')).toBe(true);
});

test('cancelled token analysis never returns stale tokens', async () => {
  const controller = new AbortController();
  const pending = syntaxTokens('const value = 42;\n'.repeat(1000), 'javascript', controller.signal);
  controller.abort();
  let cancelled = false;
  try {
    await pending;
  } catch (error) {
    cancelled = error instanceof DOMException && error.name === 'AbortError';
  }
  expect(cancelled).toBe(true);
});
