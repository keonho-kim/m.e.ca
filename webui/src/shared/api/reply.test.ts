// Purpose: Reject mixed operation payloads and misrouted native replies before state projection.
// Pattern: Transport parser regression tests.
// Usage: bun test src/shared/api/reply.test.ts.
// Related: webui/src/shared/api/reply.ts.
import { expect, test } from 'bun:test';
import { commandResult, parseReply, replyFailure } from '@/shared/api/reply';

test('native replies keep one result and the expected workspace and request identity', () => {
  const payload = { id: 'one', workspaceId: 'project', editor: { text: 'large document' } };
  expect(parseReply(payload, 'editor', 'project', 'one')).toBe(payload);
  for (const body of [
    null,
    [],
    {},
    { ...payload, id: 'other' },
    { ...payload, workspaceId: 'other' },
    { ...payload, state: {} },
    { ...payload, editor: 'text' },
  ])
    expect(() => parseReply(body, 'editor', 'project', 'one')).toThrow();
  expect(() => parseReply({ state: {} }, 'editor')).toThrow();
  expect(() => parseReply({ error: { code: 'conflict', message: 'changed' } }, 'editor')).toThrow(
    'changed'
  );
});
test('a command must identify one known native operation', () => {
  expect(commandResult({ editorProbe: { path: 'one' } })).toBe('editorVersion');
  expect(() => commandResult({})).toThrow();
  expect(() => commandResult({ editorRead: {}, editorSave: {} })).toThrow();
});
test('binary and upload error boundaries accept only native code and message strings', () => {
  expect(replyFailure({ error: { code: 'conflict', message: 'Changed' } })).toMatchObject({
    code: 'conflict',
    rejected: true,
  });
  for (const error of [
    { code: 3, message: 'invalid' },
    { code: 'conflict', message: {} },
    { code: '', message: 'invalid' },
  ])
    expect(replyFailure({ error })).toMatchObject({ code: 'invalid_response' });
  expect(replyFailure(null)).toMatchObject({ code: 'network' });
});
