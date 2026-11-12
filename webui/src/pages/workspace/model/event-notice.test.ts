// Purpose: Keep lifecycle copy localized and hide unsupported protocol notices.
// Pattern: Pure presentation contract tests.
// Usage: bun test src/pages/workspace/model/event-notice.test.ts.
// Related: webui/src/pages/workspace/model/event-notice.ts.
import { expect, test } from 'bun:test';
import { eventNotice } from '@/pages/workspace/model/event-notice';

test('stored and live lifecycle notices use presentation keys', () => {
  const t = (key: string) => `translated:${key}`;
  expect(eventNotice({ kind: 'retry' }, t)).toBe('translated:retrying');
  expect(eventNotice({ kind: 'compaction_started' }, t)).toBe('translated:compacting');
  expect(eventNotice({ kind: 'skill_loaded', skill: { name: 'application-design' } }, t)).toBe(
    'translated:skillLoaded · application-design'
  );
  for (const kind of ['unknown_event', '__proto__', 'toString'])
    expect(eventNotice({ kind }, t)).toBeUndefined();
});
