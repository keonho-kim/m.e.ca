// Purpose: Verify live-output memory bounds and the history handoff without final-response flicker.
// Pattern: Pure presentation contract tests.
// Usage: bun test src/pages/workspace/model/stream-projection.test.ts.
// Related: webui/src/pages/workspace/model/stream-projection.ts, webui/src/shared/lib/utf8/index.ts.
import { expect, test } from 'bun:test';
import type { WorkspaceState } from '@/pages/workspace/model/store';
import { useWorkspaceStore } from '@/pages/workspace/model/store';
import { applyEvents } from '@/pages/workspace/model/stream-projection';
import { trimTasks } from '@/pages/workspace/model/task-state';

test('a completed draft remains visible until durable history takes over', () => {
  let state = {
    selection: { workspace: 'w', session: 's' },
    drafts: {},
    tasks: {},
    events: [],
    historyRevision: 0,
  } as unknown as WorkspaceState;
  state = {
    ...state,
    ...applyEvents(state, [
      {
        workspaceId: 'w',
        sessionId: 's',
        taskId: 't',
        kind: 'assistant_delta',
        text: 'final answer',
      },
    ]),
  };
  state = {
    ...state,
    ...applyEvents(state, [
      {
        workspaceId: 'w',
        sessionId: 's',
        taskId: 't',
        kind: 'request_finished',
        status: 'answered',
      },
    ]),
  };
  expect(state.drafts['t/']?.text).toBe('final answer');
  expect(state.drafts['t/']?.doneRevision).toBe(1);
  state = {
    ...state,
    ...applyEvents(state, [
      {
        workspaceId: 'elsewhere',
        sessionId: 's',
        taskId: 'other',
        kind: 'assistant_delta',
        text: 'never retain',
      },
    ]),
  };
  expect(Object.keys(state.drafts)).toEqual(['t/']);
});
test('a streamed draft releases text beyond the display window', () => {
  const state = {
    selection: { workspace: 'w', session: 's' },
    drafts: {},
    tasks: {},
    events: [],
    historyRevision: 0,
  } as unknown as WorkspaceState;
  const result = applyEvents(state, [
    {
      workspaceId: 'w',
      sessionId: 's',
      taskId: 't',
      kind: 'assistant_delta',
      text: 'a'.repeat(256 * 1024),
    },
  ]);
  expect(result.drafts?.['t/']?.text.length).toBe(128 * 1024);
  expect(result.drafts?.['t/']?.truncated).toBe(true);
});

test('final usage remains available after durable history releases live activity', () => {
  const state = {
    selection: { workspace: 'w', session: 's' },
    drafts: {},
    tasks: {},
    events: [],
    historyRevision: 0,
    statistics: null,
  } as unknown as WorkspaceState;
  const projected = applyEvents(state, [
    {
      workspaceId: 'w',
      sessionId: 's',
      taskId: 't',
      kind: 'request_finished',
      status: 'answered',
      statistics: { usage: { input: '7', output: '5' } },
    },
  ]);
  expect({ ...projected, events: [] }.statistics?.usage?.output).toBe('5');
});

test('delta compaction preserves generation reset ordering', () => {
  const state = {
    ...useWorkspaceStore.getInitialState(),
    selection: { workspace: 'w', session: 's' },
  };
  const base = { workspaceId: 'w', sessionId: 's', taskId: 't' };
  const result = applyEvents(state, [
    { ...base, kind: 'assistant_delta', generation: '1', text: 'a' },
    { ...base, kind: 'assistant_delta', generation: '1', text: 'b' },
    { ...base, kind: 'draft_reset', generation: '2' },
    { ...base, kind: 'assistant_delta', generation: '2', text: 'c' },
    { ...base, kind: 'assistant_delta', generation: '2', text: 'd' },
  ]);
  expect(result.drafts?.['t//1']?.text).toBe('ab');
  expect(result.drafts?.['t/']?.text).toBe('cd');
});
test('terminal cache eviction protects accepted and canceling work', () => {
  const tasks = Object.fromEntries(
    Array.from({ length: 1025 }, (_, i) => [
      String(i),
      {
        id: String(i),
        workspace: 'w',
        session: String(i),
        status: i === 0 ? 'accepted' : i === 1 ? 'canceling' : 'answered',
      },
    ])
  );
  trimTasks(tasks);
  expect(tasks['0']?.status).toBe('accepted');
  expect(tasks['1']?.status).toBe('canceling');
  expect(Object.keys(tasks)).toHaveLength(1024);
});
