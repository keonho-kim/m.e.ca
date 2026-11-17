// Purpose: Verify interleaved generation and activity order, including reconnect snapshots.
// Pattern: Pure presentation regression.
// Usage: Bun runs this with the workspace model tests.
// Related: webui/src/pages/workspace/model/live-timeline.ts.
import { expect, test } from 'bun:test';
import { liveTimeline } from '@/pages/workspace/model/live-timeline';
import { snapshotDrafts } from '@/pages/workspace/model/stream-projection';
import type { Draft } from '@/pages/workspace/model/types';
import type { EventJson } from '@/shared/api';

const draft: Draft = {
  task: 't',
  agent: 'a',
  generation: '2',
  text: '',
  thinking: 'Next thought',
  truncated: false,
  startSequence: '8',
};
const tool: EventJson = {
  taskId: 't',
  agent: 'a',
  runId: 'r',
  sequence: '3',
  kind: 'tool_started',
  tool: { executionId: 'exec', name: 'load_skill', batchId: 'wave' },
};
const notice: EventJson = {
  taskId: 't',
  agent: 'a',
  sequence: '6',
  kind: 'skill_loaded',
  notice: { code: 'SKILL_READ', skill: { name: 'design' } },
};

test('a prior tool and skill remain before the next generation', () => {
  const entries = liveTimeline([draft], [tool, notice], []);
  expect(entries.map((entry) => entry.kind)).toEqual(['activity', 'activity', 'draft']);
});
test('tool completion updates its original slot rather than moving beneath a newer thought', () => {
  const done = { ...tool, sequence: '12', kind: 'tool_succeeded' };
  const entries = liveTimeline([draft], [tool, notice, done], []);
  expect(entries.map((entry) => entry.kind)).toEqual(['activity', 'activity', 'draft']);
  expect(entries[0]?.kind === 'activity' && entries[0].events[0]).toBe(done);
});
test('snapshot sources use publication order rather than alphabetical agent order', () => {
  const drafts = snapshotDrafts(
    {
      drafts: [
        { taskId: 't', agent: 'a', generation: '2', startSequence: '8', thinking: 'Later' },
        { taskId: 't', agent: 'z', generation: '1', startSequence: '2', thinking: 'Earlier' },
      ],
    },
    1
  );
  const entries = liveTimeline(Object.values(drafts), [notice], []);
  expect(entries.map((entry) => entry.kind)).toEqual(['draft', 'activity', 'draft']);
  expect(entries[0]?.kind === 'draft' && entries[0].draft.agent).toBe('z');
});

test('a new request does not jump ahead of a previous request awaiting history handoff', () => {
  const older = { ...draft, task: 'old', startSequence: '100' };
  const newer = { ...draft, task: 'new', startSequence: '2' };
  const entries = liveTimeline([newer, older], [], [], ['old', 'new']);
  expect(entries[0]?.kind === 'draft' && entries[0].draft.task).toBe('old');
});
