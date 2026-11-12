// Purpose: Reject mixed or misclassified rendering bodies at every browser receive boundary.
// Pattern: Runtime wire-contract regressions with hostile but deterministic payloads.
// Usage: bun test src/shared/api/transcript.test.ts.
// Related: webui/src/shared/api/transcript.ts, webui/src/shared/api/reply.ts.
import { expect, test } from 'bun:test';
import {
  parseEvent,
  parseHistory,
  parseNotification,
  parseSnapshot,
} from '@/shared/api/transcript';

test('fixed notices reject free text and messages cannot masquerade as lifecycle events', () => {
  for (const code of ['RETRYING', 'COMPACTION_STARTED', 'COMPACTION_FINISHED'] as const) {
    const kind = (
      {
        RETRYING: 'retry',
        COMPACTION_STARTED: 'compaction_started',
        COMPACTION_FINISHED: 'compaction_finished',
      } as Record<string, string>
    )[code];
    const safe = { kind, notice: { code } };
    expect(parseEvent(safe)).toBe(safe);
    for (const mixed of [
      { ...safe, text: 'AGENT_TEXT' },
      { ...safe, notice: { code, text: 'AGENT_TEXT' } },
      { ...safe, message: { role: 'assistant', text: 'AGENT_TEXT' } },
    ]) {
      expect(() => parseEvent(mixed)).toThrow();
      expect(() => parseSnapshot({ activity: [mixed] })).toThrow();
      expect(() => parseNotification({ event: mixed })).toThrow();
    }
  }
  expect(() =>
    parseEvent({ kind: 'team_notice', message: { role: 'assistant', text: 'joined' } })
  ).toThrow();
  expect(() =>
    parseEvent({ kind: 'future', message: { role: 'assistant', text: 'unknown' } })
  ).toThrow();
});
test('history rejects hybrid rows, wrong numbers and incompatible terminal bodies', () => {
  const completion = {
    status: 'COMPLETED',
    statistics: { elapsedMillis: '1000', usage: { input: '3', output: '4' } },
  };
  const row = { sequence: '7', completion };
  const page = { through: '7', items: [row] };
  expect(() => parseHistory(page)).not.toThrow();
  for (const bad of [
    { ...row, message: { role: 'assistant', text: 'mixed' } },
    { ...row, event: { kind: 'team_work', work: { summary: 'mixed' } } },
    { ...row, text: 'legacy free text' },
    { ...row, completion: { ...completion, reason: 'free reason' } },
    { ...row, completion: { status: 'AGENT_TEXT' } },
    {
      ...row,
      completion: { status: 'COMPLETED', statistics: { usage: { output: 'AGENT_TEXT' } } },
    },
  ])
    expect(() => parseHistory({ through: '7', items: [bad] })).toThrow();
  expect(() => parseHistory({ through: '6', items: [row] })).toThrow();
});
test('team work stays typed and parsing retains original bodies without a second allocation', () => {
  const work = {
    kind: 'team_work',
    historySequence: '8',
    agent: 'Hana',
    work: {
      summary: 'I own the API.',
      details: { revision: '1', paths: ['api/'], contract: 'Check errors.' },
    },
  };
  expect(parseEvent(work)).toBe(work);
  expect(() => parseSnapshot({ activity: [work] })).not.toThrow();
  expect(() => parseEvent({ ...work, notice: { code: 'RETRYING' } })).toThrow();
  expect(() => parseEvent({ ...work, kind: 'retry' })).toThrow();
});
