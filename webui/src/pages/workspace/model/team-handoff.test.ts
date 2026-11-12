// Purpose: Keep team prose out of notices and retire each live event only at a confirmed durable position.
// Pattern: Deterministic model state-transition regressions.
// Usage: bun test src/pages/workspace/model/team-handoff.test.ts.
// Related: webui/src/pages/workspace/model/stream-slice.ts, webui/src/pages/workspace/model/chat-events.ts.
import { afterEach, expect, test } from 'bun:test';
import { liveActivity } from '@/pages/workspace/model/chat-events';
import { eventNotice } from '@/pages/workspace/model/event-notice';
import { historyRows } from '@/pages/workspace/model/history-rows';
import { useWorkspaceStore } from '@/pages/workspace/model/store';
import type { EventJson } from '@/shared/api';

const target = { workspace: 'w', session: 's' };
const base = { workspaceId: 'w', sessionId: 's', taskId: 't' };
const work: EventJson = {
  ...base,
  kind: 'team_work',
  historySequence: '9',
  agent: 'Hana',
  work: { summary: 'I own the API.' },
};
afterEach(() => useWorkspaceStore.setState(useWorkspaceStore.getInitialState(), true));
test('authored team work never projects to system notice and saved work replaces live work', () => {
  const item = { sequence: '9', event: work };
  expect(eventNotice(work.notice, (key: string) => key)).toBeUndefined();
  expect(historyRows([item])).toHaveLength(1);
  expect(liveActivity([work], [item])).toEqual([]);
  useWorkspaceStore.setState({ selection: target });
  useWorkspaceStore
    .getState()
    .receive([{ ...base, kind: 'request_started', status: 'running' }, work]);
  useWorkspaceStore.getState().acknowledgeHistory(target, 0, { through: '8' });
  expect(useWorkspaceStore.getState().events).toContainEqual(work);
  useWorkspaceStore.getState().acknowledgeHistory(target, 0, { through: '9', items: [item] });
  expect(useWorkspaceStore.getState().events).toEqual([]);
});
test('task completion cannot discard a draft or activity against stale history', () => {
  useWorkspaceStore.setState({ selection: target });
  useWorkspaceStore.getState().receive([
    { ...base, kind: 'request_started', status: 'running' },
    work,
    {
      ...base,
      kind: 'assistant_delta',
      agent: 'Ren',
      message: { role: 'assistant', text: 'last response' },
    },
    { ...base, kind: 'request_finished', status: 'answered', historyThrough: '12' },
  ]);
  const revision = useWorkspaceStore.getState().historyRevision;
  useWorkspaceStore.getState().acknowledgeHistory(target, revision, { through: '9' });
  expect(Object.values(useWorkspaceStore.getState().drafts)).toHaveLength(1);
  useWorkspaceStore.getState().acknowledgeHistory(target, revision, { through: '12' });
  expect(useWorkspaceStore.getState().drafts).toEqual({});
  expect(useWorkspaceStore.getState().events).toEqual([]);
  useWorkspaceStore
    .getState()
    .receive([
      { ...base, kind: 'assistant_delta', message: { role: 'assistant', text: 'late response' } },
    ]);
  expect(useWorkspaceStore.getState().drafts).toEqual({});
});
test('duplicates and replay from an older epoch are discarded before delta compaction', () => {
  useWorkspaceStore.setState({ selection: target, replayCursor: 'epoch:1' });
  const a = {
    ...base,
    kind: 'assistant_delta',
    cursor: 'epoch:2',
    message: { role: 'assistant', text: 'A' },
  };
  const b = { ...a, cursor: 'epoch:3', message: { role: 'assistant', text: 'B' } };
  useWorkspaceStore.getState().receive([a]);
  useWorkspaceStore
    .getState()
    .receive([a, b, { ...b, cursor: 'old:99', message: { role: 'assistant', text: 'LEAK' } }]);
  expect(useWorkspaceStore.getState().drafts['t/']?.text).toBe('AB');
});
