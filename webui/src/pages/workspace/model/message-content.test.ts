// Purpose: Preserve literal prose while projecting the canonical reference suffix for display.
// Pattern: User message presentation regression tests.
// Usage: bun test src/pages/workspace/model/message-content.test.ts.
// Related: webui/src/pages/workspace/model/message-content.ts.
import { expect, test } from 'bun:test';
import { messageContent } from '@/pages/workspace/model/message-content';

test('canonical file and folder references do not leak engine instructions into the bubble', () => {
  expect(
    messageContent(
      'Inspect #review\nSelected path references (read only as needed):\n/project/a.go\n/project/docs/'
    )
  ).toEqual({ text: 'Inspect #review', paths: ['/project/a.go', '/project/docs/'] });
});
test('relative names and names with spaces retain their exact serialized value', () => {
  expect(
    messageContent(
      'Inspect\nSelected path references (read only as needed):\nREADME.md\nMy folder/\nLICENSE'
    ).paths
  ).toEqual(['README.md', 'My folder/', 'LICENSE']);
});
test('ordinary text mentioning references without a complete reference block is preserved', () => {
  const value = 'Explain Selected path references (read only as needed):\nthis is prose';
  expect(messageContent(value)).toEqual({ text: value, paths: [] });
});
