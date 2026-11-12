// Purpose: Keep lifecycle copy localized and hide unsupported protocol notices.
// Pattern: Pure presentation contract tests.
// Usage: bun test src/pages/workspace/model/event-notice.test.ts.
// Related: webui/src/pages/workspace/model/event-notice.ts.
import { expect, test } from 'bun:test';
import { eventNotice } from '@/pages/workspace/model/event-notice';

test('stored and live lifecycle notices use presentation keys', () => {
  const t = (key: string) => `translated:${key}`;
  expect(eventNotice({ code: 'RETRYING' }, t)).toBe('translated:retrying');
  expect(eventNotice({ code: 'COMPACTION_STARTED' }, t)).toBe('translated:compacting');
  expect(
    eventNotice(
      {
        code: 'SKILL_READ',
        skill: { name: 'application-design' },
      },
      t
    )
  ).toBe('translated:skillLoaded · application-design');
  expect(eventNotice(undefined, t)).toBeUndefined();
});
