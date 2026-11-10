// Purpose: Verify durable submission reconciliation and follow-up ordering under delayed acknowledgements.
// Pattern: Deterministic API boundary tests with controlled promises.
// Usage: bun test src/execution.test.ts.
// Related: webui/src/pages/workspace/model/submission.ts, webui/src/pages/workspace/model/followups.ts, webui/src/pages/workspace/model/store.ts.
import { afterAll, beforeEach, expect, spyOn, test } from 'bun:test';
import type { ReplyJson } from '@/shared/api';
import * as api from '@/shared/api';
import { queries as cache } from '@/shared/api';

const invoke = spyOn(api, 'call');
afterAll(() => invoke.mockRestore());
const { useWorkspaceStore, emptyInput, targetKey } = await import('@/pages/workspace/model/store');
const { startRequest, clearPending, awaitSubmission } = await import(
  '@/pages/workspace/model/submission'
);
const { completed, clearFollowups, queueEditing } = await import(
  '@/pages/workspace/model/followups'
);
const { ApiError } = await import('@/shared/lib/errors');
const target = { workspace: 'project', session: 'conversation' };
const input = (text: string) => ({ ...emptyInput(), text });
function deferred<T>() {
  let resolve!: (value: T) => void;
  const promise = new Promise<T>((yes) => {
    resolve = yes;
  });
  return { promise, resolve };
}
beforeEach(() => {
  cache.clear();
  clearPending();
  clearFollowups();
  invoke.mockReset();
  useWorkspaceStore.setState({
    selection: target,
    inputs: {},
    queues: {},
    pausedQueues: {},
    tasks: {},
    drafts: {},
    events: [],
    historyRevision: 0,
    removedTarget: null,
  });
});

test('a terminal event arriving before its start acknowledgement advances a captured follow-up once', async () => {
  const ack = deferred<ReplyJson>(),
    started = deferred<void>(),
    queuedStart = deferred<void>();
  const texts: string[] = [];
  invoke.mockImplementation(async (_workspace, command) => {
    if (command.start) {
      texts.push(command.start.text ?? '');
      if (texts.length === 1) {
        started.resolve();
        return ack.promise;
      }
      queuedStart.resolve();
      return { task: { id: 'second', sessionId: target.session, status: 'running' } };
    }
    return {};
  });
  const sending = startRequest(target, input('first'));
  await started.promise;
  useWorkspaceStore.getState().enqueue(target, input('queued'));
  useWorkspaceStore.getState().receive([
    {
      kind: 'request_finished',
      workspaceId: target.workspace,
      sessionId: target.session,
      taskId: 'first',
      status: 'answered',
      outcome: { status: 'completed' },
    },
  ]);
  completed(target, 'first', 'completed', '');
  expect(texts).toEqual(['first']);
  ack.resolve({ task: { id: 'first', sessionId: target.session, status: 'running' } });
  await sending;
  await queuedStart.promise;
  await awaitSubmission(target);
  expect(texts).toEqual(['first', 'queued']);
  expect(useWorkspaceStore.getState().tasks.first?.status).toBe('answered');
  expect(useWorkspaceStore.getState().queues[targetKey(target)] ?? []).toHaveLength(0);
  completed(target, 'first', 'completed', '');
  await Promise.resolve();
  expect(texts).toHaveLength(2);
});

test('answered persistence does not imply successful work or run a queued side effect', async () => {
  useWorkspaceStore.getState().enqueue(target, input('must wait'));
  completed(target, 'failed', 'failed', '');
  await Promise.resolve();
  await Promise.resolve();
  await Promise.resolve();
  expect(invoke).not.toHaveBeenCalled();
  expect(useWorkspaceStore.getState().queues[targetKey(target)]).toHaveLength(1);
  expect(useWorkspaceStore.getState().pausedQueues[targetKey(target)]).toBe(true);
});

test('queue editing holds completion until the edited content is committed', async () => {
  const started = deferred<void>();
  invoke.mockImplementation(async (_workspace, command) => {
    if (command.start) {
      started.resolve();
      return { task: { id: 'edited', status: 'running' } };
    }
    return {};
  });
  useWorkspaceStore.getState().enqueue(target, input('before'));
  const entry = useWorkspaceStore.getState().queues[targetKey(target)]?.[0];
  if (!entry) throw new Error('Expected the queued fixture request');
  queueEditing(target, true);
  completed(target, 'predecessor', 'completed', '');
  await Promise.resolve();
  expect(invoke).not.toHaveBeenCalled();
  useWorkspaceStore.getState().updateQueue(target, entry.id, 'after');
  queueEditing(target, false);
  await started.promise;
  await awaitSubmission(target);
  expect(invoke.mock.calls.find(([, c]) => c.start)?.[1].start?.text).toBe('after');
});

test('an uncertain submission is looked up and never started twice', async () => {
  let starts = 0;
  invoke.mockImplementation(async (_workspace, command) => {
    if (command.start) {
      starts++;
      throw new ApiError('network', 'disconnected');
    }
    if (command.lookupRequest)
      return { task: { id: 'confirmed', sessionId: target.session, status: 'running' } };
    return {};
  });
  // oxlint-disable-next-line typescript/await-thenable -- Bun awaits rejects matchers at runtime, but its declaration returns void.
  await expect(startRequest(target, input('original'))).rejects.toThrow('disconnected');
  expect(await startRequest(target, input('original'))).toBe(true);
  expect(starts).toBe(1);
});

test('image retention is bounded across unsent conversations', () => {
  const large = new File([new Uint8Array(80 * 1024 * 1024)], 'large.png', { type: 'image/png' });
  expect(useWorkspaceStore.getState().patchInput(target, { files: [large] })).toBe(true);
  const other = { ...target, session: 'other' };
  const next = new File([new Uint8Array(50 * 1024 * 1024)], 'next.png', { type: 'image/png' });
  expect(useWorkspaceStore.getState().patchInput(other, { files: [next] })).toBe(false);
  expect(useWorkspaceStore.getState().inputs[targetKey(other)]).toBeUndefined();
});

test('conversation deletion releases its cached bodies and drafts without clearing another conversation', async () => {
  const { forgetConversation } = await import('@/pages/workspace/model/deletion');
  cache.setQueryData(['history', target.workspace, target.session, '0'], { body: 'private body' });
  cache.setQueryData(['history', target.workspace, 'other', '0'], { body: 'keep body' });
  useWorkspaceStore.getState().patchInput(target, { text: 'unsent' });
  useWorkspaceStore.getState().enqueue(target, input('queued'));
  forgetConversation(target);
  expect(cache.getQueryData(['history', target.workspace, target.session, '0'])).toBeUndefined();
  expect(cache.getQueryData(['history', target.workspace, 'other', '0'])).toEqual({
    body: 'keep body',
  });
  expect(useWorkspaceStore.getState().inputs[targetKey(target)]).toBeUndefined();
  expect(useWorkspaceStore.getState().queues[targetKey(target)]).toBeUndefined();
});
