// Purpose: Verify incremental UTF-8 accounting through edits and surrogate boundaries.
// Pattern: Deterministic boundary and randomized edit contracts.
// Usage: bun test src/shared/lib/utf8/delta.test.ts.
// Related: webui/src/shared/lib/utf8/index.ts.
import { expect, test } from 'bun:test';
import { editedBytes, utf8Bytes } from '@/shared/lib/utf8';

test('edits match full encoding across every UTF-16 boundary', () => {
  const source = 'x한😀\ud800abc\udc00z';
  for (let start = 0; start <= source.length; start++)
    for (let end = start; end <= source.length; end++)
      for (const insert of ['', 'word', '한', '😀', '\ud800', '\udc00']) {
        const next = source.slice(0, start) + insert + source.slice(end);
        expect(editedBytes(source, next, utf8Bytes(source))).toBe(
          new TextEncoder().encode(next).length
        );
        expect(editedBytes(source, next, utf8Bytes(source), { start, end, inserted: insert })).toBe(
          new TextEncoder().encode(next).length
        );
      }
});
test('native block comparisons preserve edits beyond a large common prefix and suffix', () => {
  const before = '한'.repeat(20000) + '😀' + 'x'.repeat(20000);
  for (const next of [
    before + 'x',
    before.slice(0, -1),
    before.slice(0, 20001) + '𐐀new' + before.slice(20002),
    'new' + before,
  ])
    expect(editedBytes(before, next, utf8Bytes(before))).toBe(utf8Bytes(next));
});
