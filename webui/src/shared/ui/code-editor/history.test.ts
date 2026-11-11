// Purpose: Verify editor history budgets retain the current selection and valid undo/redo neighbors.
// Pattern: Deterministic editor session contract tests.
// Usage: bun test src/shared/ui/code-editor/history.test.ts.
// Related: webui/src/shared/ui/code-editor/history.ts.
import { expect, test } from 'bun:test';
import { trimEditorHistory } from '@/shared/ui/code-editor/history';

function history(offset: number) {
  return {
    stack: ['first', 'second', 'current', 'redo'].map((value, index) => ({
      value: value.repeat(100),
      selectionStart: index,
      selectionEnd: index,
      timestamp: index,
    })),
    offset,
  };
}
test('oversized undo records release old text without changing the current selection', () => {
  const value = history(2),
    current = value.stack[2];
  trimEditorHistory(value, 2500);
  expect(value.stack).toHaveLength(2);
  expect(value.stack[value.offset]).toBe(current);
  expect(value.stack[value.offset + 1]?.value).toBe('redo'.repeat(100));
  expect(current?.selectionStart).toBe(2);
});
test('redo trimming keeps the current record when it is the first item', () => {
  const value = history(0),
    current = value.stack[0];
  trimEditorHistory(value, 1500);
  expect(value.stack).toHaveLength(1);
  expect(value.stack[value.offset]).toBe(current);
});
test('ordinary history stays intact and empty sessions are valid', () => {
  const value = history(2),
    original = [...value.stack];
  trimEditorHistory(value);
  expect(value.stack).toEqual(original);
  expect(value.offset).toBe(2);
  expect(() => trimEditorHistory({ stack: [], offset: -1 })).not.toThrow();
});

test('the full two-MiB boundary keeps one undo step within the text budget', () => {
  const value = {
    stack: ['x'.repeat(2 * 1024 * 1024 - 1), 'x'.repeat(2 * 1024 * 1024)].map((text) => ({
      value: text,
      selectionStart: text.length,
      selectionEnd: text.length,
      timestamp: 0,
    })),
    offset: 1,
  };
  trimEditorHistory(value);
  expect(value.stack).toHaveLength(2);
  expect(value.offset).toBe(1);
});
