// Purpose: Verify no-op input identity, scoped task selection and cache observer disposal.
// Pattern: Deterministic workspace state boundary tests.
// Usage: bun test src exercises workspace subscriptions without rendering UI.
// Related: webui/src/pages/workspace/model/store.ts, webui/src/pages/workspace/api/cache-policy.ts.
import { afterEach, beforeEach, expect, test } from 'bun:test';
import { observeConversationCache } from '@/pages/workspace/api/cache-policy';
import { inputBytes } from '@/pages/workspace/model/input-usage';
import { useWorkspaceStore } from '@/pages/workspace/model/store';
import { activeTask } from '@/pages/workspace/model/task-state';
import { queries } from '@/shared/api';

beforeEach(() => {
  queries.clear();
  useWorkspaceStore.setState(useWorkspaceStore.getInitialState(), true);
});
afterEach(() => queries.clear());
const target = { workspace: 'alpha', session: 'one' };

test('native edit accounting rejects stale revisions and preserves exact Unicode bytes', () => {
  const state = useWorkspaceStore.getState();
  state.patchInput(target, { text: 'x'.repeat(80000) });
  const previous = useWorkspaceStore.getState().inputs['alpha/one']!;
  state.patchInput(
    target,
    { text: previous.text },
    { start: 40000, end: 40001, inserted: 'x', revision: previous.revision }
  );
  expect(useWorkspaceStore.getState().inputs['alpha/one']).toBe(previous);
  state.patchInput(
    target,
    { text: `${previous.text.slice(0, 40000)}한${previous.text.slice(40001)}` },
    { start: 40000, end: 40001, inserted: '한', revision: previous.revision }
  );
  const changed = useWorkspaceStore.getState().inputs['alpha/one']!;
  expect(inputBytes(changed)).toBe(80002);
  state.patchInput(
    target,
    { text: '😀' },
    { start: 0, end: 80000, inserted: '😀', revision: previous.revision }
  );
  expect(inputBytes(useWorkspaceStore.getState().inputs['alpha/one']!)).toBe(4);
});

test('repeated view values notify subscribers only when a value changes', () => {
  let notifications = 0;
  const stop = useWorkspaceStore.subscribe(() => notifications++);
  try {
    for (let i = 0; i < 100; i++) useWorkspaceStore.getState().set({ fileDirty: true });
    expect(notifications).toBe(1);
    useWorkspaceStore.getState().set({ fileDirty: false });
    expect(notifications).toBe(2);
  } finally {
    stop();
  }
});

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

test('accounting does not retain input and queue records after an explicit reset', async () => {
  const seed = () => {
    const state = useWorkspaceStore.getState();
    state.patchInput(target, {
      text: 'discarded draft',
      files: [new File(['pixels'], 'draft.png')],
    });
    state.enqueue(target, useWorkspaceStore.getState().inputs['alpha/one']!);
    state.patchInput(target, { text: 'another draft' });
    return {
      inputs: new WeakRef(useWorkspaceStore.getState().inputs),
      queues: new WeakRef(useWorkspaceStore.getState().queues),
    };
  };
  const discarded = seed();
  useWorkspaceStore.setState({ inputs: {}, queues: {} });
  await new Promise<void>((resolve) => setImmediate(resolve));
  Bun.gc(true);
  expect(discarded.inputs.deref()).toBeUndefined();
  expect(discarded.queues.deref()).toBeUndefined();
});

test('an image-bearing queued request remains valid when its text is cleared', () => {
  const state = useWorkspaceStore.getState();
  const image = new File(['pixels'], 'draft.png', { type: 'image/png' });
  state.patchInput(target, { text: 'Optional caption', files: [image] });
  state.enqueue(target, useWorkspaceStore.getState().inputs['alpha/one']!);
  const item = useWorkspaceStore.getState().queues['alpha/one']![0]!;
  expect(state.updateQueue(target, item.id, '')).toBe(true);
  expect(useWorkspaceStore.getState().queues['alpha/one']![0]!.files[0]).toBe(image);
});
