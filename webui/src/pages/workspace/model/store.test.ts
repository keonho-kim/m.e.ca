// Purpose: Verify no-op input identity, scoped task selection and cache observer disposal.
// Pattern: Deterministic workspace state boundary tests.
// Usage: bun test src exercises workspace subscriptions without rendering UI.
// Related: webui/src/pages/workspace/model/store.ts, webui/src/pages/workspace/api/cache-policy.ts.
import { afterEach, beforeEach, expect, test } from 'bun:test';
import { observeConversationCache } from '@/pages/workspace/api/cache-policy';
import { useWorkspaceStore } from '@/pages/workspace/model/store';
import { activeTask } from '@/pages/workspace/model/task-state';
import { queries } from '@/shared/api';

beforeEach(() => {
  queries.clear();
  useWorkspaceStore.setState(useWorkspaceStore.getInitialState(), true);
});
afterEach(() => queries.clear());
const target = { workspace: 'alpha', session: 'one' };

test('an unchanged input does not invalidate a pending request revision or notify subscribers', () => {
  useWorkspaceStore.getState().patchInput(target, { text: 'original' });
  const before = useWorkspaceStore.getState();
  let notifications = 0;
  const stop = useWorkspaceStore.subscribe(() => notifications++);
  try {
    expect(before.patchInput(target, { text: 'original', paths: [], files: [] })).toBe(true);
    expect(useWorkspaceStore.getState()).toBe(before);
    expect(notifications).toBe(0);
  } finally {
    stop();
  }
});

test('another workspace task does not change the selected active task reference', () => {
  const selected = { id: 'one', ...target, status: 'running' };
  useWorkspaceStore.setState({ tasks: { one: selected } });
  useWorkspaceStore.setState({
    tasks: {
      one: selected,
      two: { id: 'two', workspace: 'beta', session: 'two', status: 'running' },
    },
  });
  expect(activeTask(target, useWorkspaceStore.getState().tasks)).toBe(selected);
});

test('conversation cache retention is bounded only while its page owns the observer', () => {
  const stop = observeConversationCache();
  for (let i = 0; i < 20; i++) queries.setQueryData(['history', 'alpha', String(i)], { body: i });
  expect(queries.getQueryCache().getAll()).toHaveLength(12);
  stop();
  queries.setQueryData(['history', 'alpha', 'after-dispose'], { body: 21 });
  expect(queries.getQueryCache().getAll()).toHaveLength(13);
});

test('irrelevant event batches preserve store identity and do not notify subscribers', () => {
  const before = useWorkspaceStore.getState();
  let notifications = 0;
  const stop = useWorkspaceStore.subscribe(() => notifications++);
  try {
    before.receive([{ kind: 'progress', taskId: 'other', workspaceId: 'beta', sessionId: 'two' }]);
    expect(useWorkspaceStore.getState()).toBe(before);
    expect(notifications).toBe(0);
  } finally {
    stop();
  }
});

test('clearing an input releases empty metadata without reusing its request revision', () => {
  const state = useWorkspaceStore.getState();
  state.patchInput(target, { text: 'first' });
  const revision = useWorkspaceStore.getState().inputs['alpha/one']?.revision ?? 0;
  state.patchInput(target, { text: '' });
  expect(useWorkspaceStore.getState().inputs['alpha/one']).toBeUndefined();
  state.patchInput(target, { text: 'second' });
  expect(useWorkspaceStore.getState().inputs['alpha/one']?.revision).toBeGreaterThan(revision);
});

test('input accounting follows external resets and frees removed draft bytes', () => {
  const state = useWorkspaceStore.getState();
  expect(state.patchInput(target, { text: 'x'.repeat(32 * 1024 * 1024) })).toBe(true);
  expect(state.patchInput({ workspace: 'alpha', session: 'two' }, { text: 'x' })).toBe(false);
  useWorkspaceStore.setState({ inputs: {} });
  expect(state.patchInput(target, { text: 'replacement' })).toBe(true);
});

test('inactive cache byte limits evict one oversized body without waiting for the count limit', () => {
  const stop = observeConversationCache();
  try {
    queries.setQueryData(['history', 'alpha', 'large'], { text: 'x'.repeat(9 * 1024 * 1024) });
    expect(queries.getQueryData(['history', 'alpha', 'large'])).toBeUndefined();
  } finally {
    stop();
  }
});
