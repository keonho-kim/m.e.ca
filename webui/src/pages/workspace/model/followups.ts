// Purpose: Run each captured follow-up as a separate task after a confirmed successful predecessor.
// Pattern: Deduplicated per-conversation completion queue.
// Usage: Stream terminal events and queue editing/resume controls call this owner.
// Related: webui/src/pages/workspace/model/submission.ts, webui/src/pages/workspace/ui/composer/queue.tsx.

import { reportError } from '@/entities/server-session';
import { type Target, targetKey, useWorkspaceStore } from '@/pages/workspace/model/store';
import {
  awaitSubmission,
  pendingSubmission,
  reconcile,
  startRequest,
} from '@/pages/workspace/model/submission';

type Completion = { target: Target; id: string; success: boolean };
const handled = new Set<string>(),
  queued = new Set<string>(),
  workers = new Map<string, number>(),
  editing = new Set<string>();
let epoch = 0;
const completions = new Map<string, Completion[]>();

export function completed(target: Target, id: string, outcome: string, errorCode: string) {
  const state = useWorkspaceStore.getState(),
    key = targetKey(target);
  if (state.pausedQueues[key] || (!state.queues[key]?.length && !pendingSubmission(target))) return;
  if (handled.has(id) || queued.has(id)) return;
  queued.add(id);
  const list = completions.get(key) ?? [];
  list.push({ target, id, success: outcome === 'completed' && !errorCode });
  completions.set(key, list);
  void drain(target);
}
async function drain(target: Target) {
  const key = targetKey(target);
  if (
    workers.get(key) === epoch ||
    editing.has(key) ||
    useWorkspaceStore.getState().pausedQueues[key]
  )
    return;
  workers.set(key, epoch);
  const startedEpoch = epoch;
  try {
    const list = completions.get(key) ?? [];
    while (list.length) {
      const event = list[0];
      if (!event) break;
      await awaitSubmission(target);
      if (startedEpoch !== epoch) return;
      await reconcile(target, event.id);
      if (
        editing.has(key) ||
        pendingSubmission(target) ||
        useWorkspaceStore.getState().pausedQueues[key]
      )
        return;
      list.shift();
      queued.delete(event.id);
      handled.add(event.id);
      if (handled.size > 2048) handled.delete(handled.values().next().value!);
      const state = useWorkspaceStore.getState();
      if (!event.success && state.queues[key]?.length) {
        state.set({ pausedQueues: { ...state.pausedQueues, [key]: true } });
        return;
      }
      const next = state.queues[key]?.[0];
      if (next) await startRequest(target, next);
    }
    completions.delete(key);
  } catch (error) {
    if (startedEpoch !== epoch) return;
    const state = useWorkspaceStore.getState();
    state.set({ pausedQueues: { ...state.pausedQueues, [key]: true } });
    reportError(error);
  } finally {
    if (workers.get(key) === startedEpoch) workers.delete(key);
  }
}
export function queueEditing(target: Target, value: boolean) {
  const key = targetKey(target);
  if (value) editing.add(key);
  else {
    editing.delete(key);
    void drain(target);
  }
}
export async function resumeQueue(target: Target) {
  if (
    Object.values(useWorkspaceStore.getState().tasks).some(
      (t) =>
        t.workspace === target.workspace &&
        t.session === target.session &&
        ['running', 'accepted', 'canceling'].includes(t.status)
    )
  )
    return;
  const key = targetKey(target),
    state = useWorkspaceStore.getState();
  const pausedQueues = { ...state.pausedQueues };
  delete pausedQueues[key];
  state.set({ pausedQueues });
  if (completions.get(key)?.length) {
    await drain(target);
    return;
  }
  const next = state.queues[key]?.[0];
  if (next) {
    try {
      if (!(await startRequest(target, next)))
        useWorkspaceStore
          .getState()
          .set({ pausedQueues: { ...useWorkspaceStore.getState().pausedQueues, [key]: true } });
    } catch (error) {
      useWorkspaceStore
        .getState()
        .set({ pausedQueues: { ...useWorkspaceStore.getState().pausedQueues, [key]: true } });
      reportError(error);
    }
  }
}
export function clearFollowups() {
  epoch++;
  handled.clear();
  queued.clear();
  completions.clear();
  editing.clear();
  workers.clear();
}
export function clearFollowupsTarget(target: Target) {
  const key = targetKey(target);
  for (const event of completions.get(key) ?? []) queued.delete(event.id);
  completions.delete(key);
  editing.delete(key);
}
