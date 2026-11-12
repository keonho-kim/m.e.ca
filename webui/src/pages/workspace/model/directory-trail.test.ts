// Purpose: Keep collapsed breadcrumb navigation bound to the complete server path.
// Pattern: Pure path projection contract.
// Usage: bun test src/pages/workspace/model/directory-trail.test.ts.
// Related: webui/src/pages/workspace/model/directory-trail.ts.
import { expect, test } from 'bun:test';
import { directoryTrail } from '@/pages/workspace/model/directory-trail';

test('absolute and relative trails retain navigable ancestors and complete Unicode names', () => {
  expect(directoryTrail('/a/b/한글/file').slice(-3)).toEqual([
    { label: 'b', path: '/a/b' },
    { label: '한글', path: '/a/b/한글' },
    { label: 'file', path: '/a/b/한글/file' },
  ]);
  expect(directoryTrail('./a//b/')).toEqual([
    { label: 'a', path: 'a' },
    { label: 'b', path: 'a/b' },
  ]);
  expect(directoryTrail('.')).toEqual([]);
});
