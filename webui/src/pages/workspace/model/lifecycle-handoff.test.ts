// Purpose: Prevent terminal regressions and release acknowledged activity without generated text.
// Pattern: Deterministic state-transition regressions.
// Usage: bun test src/pages/workspace/model/lifecycle-handoff.test.ts.
// Related: webui/src/pages/workspace/model/chat-events.ts, webui/src/pages/workspace/model/stream-slice.ts.
import { afterEach, expect, test } from 'bun:test';
import { mergeToolEvents } from '@/pages/workspace/model/chat-events';
import { useWorkspaceStore } from '@/pages/workspace/model/store';
import { toolStatus } from '@/pages/workspace/model/tool-status';
import type { EventJson } from '@/shared/api';

const target = { workspace: 'w', session: 's' };
afterEach(() => useWorkspaceStore.setState(useWorkspaceStore.getInitialState(), true));
test('an older successful observation cannot replace a durable failure', () => {
  const tool = { executionId: 'x', name: 'execute_command' };
  const failed: EventJson = {
    runId: 'r',
    kind: 'tool_failed',
    tool: { ...tool, phase: 'TOOL_PHASE_FAILED', reference: { sessionId: 's', sequence: '3' } },
  };
  const old: EventJson = {
    runId: 'r',
    kind: 'tool_succeeded',
    tool: { ...tool, phase: 'TOOL_PHASE_SUCCEEDED', reference: { sessionId: 's', sequence: '2' } },
  };
  expect(mergeToolEvents([failed], [old])).toEqual([failed]);
});
test('a returned observation remains running while its process is active', () => {
  expect(toolStatus({ kind: 'tool_succeeded', tool: { phase: 'TOOL_PHASE_RUNNING' } })).toBe(
    'running'
  );
  expect(toolStatus({ kind: 'tool_succeeded' })).toBe('pending');
  expect(toolStatus({ kind: 'tool_skipped', tool: { phase: 'TOOL_PHASE_SKIPPED' } })).toBe('error');
});
test('a failed request without a draft releases activity after durable history catches up', () => {
  useWorkspaceStore.setState({ selection: target });
  useWorkspaceStore.getState().receive([
    { workspaceId: 'w', sessionId: 's', taskId: 't', kind: 'request_started', status: 'running' },
    { workspaceId: 'w', sessionId: 's', taskId: 't', kind: 'retry', text: 'retrying' },
    { workspaceId: 'w', sessionId: 's', taskId: 't', kind: 'request_finished', status: 'failed' },
  ]);
  const revision = useWorkspaceStore.getState().historyRevision;
  useWorkspaceStore.getState().acknowledgeHistory(target, revision - 1);
  expect(useWorkspaceStore.getState().events).toHaveLength(2);
  useWorkspaceStore.getState().acknowledgeHistory(target, revision);
  expect(useWorkspaceStore.getState().events).toEqual([]);
});
