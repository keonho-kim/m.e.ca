// Purpose: Reconcile durable request identities before retrying an uncertain native submission.
// Pattern: Per-conversation stable submission lifecycle.
// Usage: Direct requests and queued follow-ups share startRequest.
// Related: webui/src/pages/workspace/model/execution.ts, internal/commands/server/request_lookup.go.

import { inputBytes, rememberInputBytes } from '@/pages/workspace/model/input-usage';
import { waitSelections } from '@/pages/workspace/model/reference-queue';
import {
  emptyInput,
  type Input,
  type Target,
  type TaskView,
  targetKey,
  useWorkspaceStore,
} from '@/pages/workspace/model/store';
import { admitSubmission } from '@/pages/workspace/model/submission-limits';
import { trimTasks } from '@/pages/workspace/model/task-state';
import type { TaskJson } from '@/shared/api';
import { type CapabilitiesJson, call, queries, releaseUpload, upload } from '@/shared/api';
import { ApiError, failure } from '@/shared/lib/errors';

type Submission = { key: string; input: Input; target: Target; attachmentIds: string[] };
const pending = new Map<string, Submission>(),
  sending = new Set<string>();
const controllers = new Map<string, AbortController>();
const reading = new Map<string, Set<AbortController>>();
const inflight = new Map<string, Promise<void>>();
let epoch = 0;
async function releaseAttachments(workspace: string, ids: string[]) {
  // Transfer known IDs once; a cancelled upload may still append a late receipt.
  const staged = ids.splice(0);
  await Promise.allSettled(staged.map((id) => releaseUpload(workspace, id)));
}
export function terminal(status: string) {
  return ['answered', 'stopped', 'interrupted'].includes(status);
}
function sameInput(a: Input, b: Input) {
  return (
    a.text === b.text &&
    a.paths.length === b.paths.length &&
    a.paths.every((v, i) => v === b.paths[i]) &&
    a.files.length === b.files.length &&
    a.files.every((v, i) => v === b.files[i])
  );
}
export function recordTask(target: Target, task: TaskJson) {
  if (!task.id) return;
  const state = useWorkspaceStore.getState(),
    old = state.tasks[task.id];
  if (
    state.removedTarget?.workspace === target.workspace &&
    state.removedTarget.session === target.session
  )
    return;
  if (old && terminal(old.status) && !terminal(task.status ?? '')) return;
  const value: TaskView = { id: task.id, ...target, status: task.status ?? 'running' };
  if (
    old?.status === value.status &&
    old.workspace === value.workspace &&
    old.session === value.session
  )
    return;
  const tasks = { ...state.tasks, [task.id]: value };
  trimTasks(tasks);
  state.set({ tasks });
}
async function forget(value: Submission, task: TaskJson, startedEpoch: number) {
  if (startedEpoch !== epoch) return;
  const key = targetKey(value.target);
  if (pending.get(key) !== value) return;
  recordTask(value.target, task);
  pending.delete(key);
  if ('id' in value.input)
    useWorkspaceStore.getState().removeQueue(value.target, String(value.input.id));
  const current = useWorkspaceStore.getState().inputs[key] ?? emptyInput();
  if (
    !('id' in value.input) &&
    current.revision === value.input.revision &&
    sameInput(current, value.input)
  ) {
    useWorkspaceStore.getState().patchInput(value.target, { text: '', files: [] });
  }
  await releaseAttachments(value.target.workspace, value.attachmentIds);
}
async function lookup(value: Submission) {
  const key = targetKey(value.target),
    controller = new AbortController();
  const readers = reading.get(key) ?? new Set<AbortController>();
  readers.add(controller);
  reading.set(key, readers);
  try {
    return (
      await call(
        value.target.workspace,
        {
          lookupRequest: { sessionId: value.target.session, key: value.key },
        },
        controller.signal
      )
    ).task;
  } catch (error) {
    if (error instanceof ApiError && error.code === 'not_found') return undefined;
    throw error;
  } finally {
    readers.delete(controller);
    if (!readers.size && reading.get(key) === readers) reading.delete(key);
  }
}
export async function reconcile(target: Target, taskID?: string) {
  const value = pending.get(targetKey(target));
  if (!value) return;
  const receipt = await lookup(value);
  if (receipt?.id && (!taskID || receipt.id === taskID)) await forget(value, receipt, epoch);
}

export async function startRequest(target: Target, input: Input): Promise<boolean> {
  if (!target.session) throw failure('no_session');
  const capabilities = queries.getQueryData<CapabilitiesJson>(['capabilities']);
  const key = targetKey(target);
  if (sending.has(key)) return false;
  sending.add(key);
  let done = () => {};
  const completion = new Promise<void>((resolve) => {
    done = resolve;
  });
  inflight.set(key, completion);
  const startedEpoch = epoch;
  const controller = new AbortController();
  controllers.set(key, controller);
  let value = pending.get(key);
  try {
    if (capabilities) await admitSubmission(input, capabilities, controller.signal);
    if (value) {
      const receipt = await lookup(value);
      if (controller.signal.aborted) return false;
      if (receipt?.id) {
        const same = sameInput(value.input, input);
        await forget(value, receipt, startedEpoch);
        if (same) return true;
        if (!terminal(receipt.status ?? '')) return false;
        value = undefined;
      } else if (!sameInput(value.input, input)) throw failure('request_pending');
    }
    await waitSelections(target);
    controller.signal.throwIfAborted();
    if (startedEpoch !== epoch) return false;
    if (!value) {
      const values = [...pending.values()];
      const bytes = values.reduce((n, v) => n + inputBytes(v.input), 0) + inputBytes(input);
      const files = new Set([...values.flatMap((v) => v.input.files), ...input.files]);
      if (
        values.length >= 256 ||
        bytes > 32 * 1024 * 1024 ||
        [...files].reduce((n, f) => n + f.size, 0) > 128 * 1024 * 1024
      )
        throw failure('queue_limit');
      value = {
        key: crypto.randomUUID(),
        input: { ...input, paths: [...input.paths], files: [...input.files] },
        target,
        attachmentIds: [],
      };
      rememberInputBytes(value.input, inputBytes(input));
      pending.set(key, value);
      try {
        for (const file of input.files)
          value.attachmentIds.push((await upload(target.workspace, file, controller.signal)).id);
      } catch (error) {
        if (pending.get(key) === value) pending.delete(key);
        await releaseAttachments(target.workspace, value.attachmentIds);
        throw error;
      }
    }
    if (startedEpoch !== epoch || controller.signal.aborted) {
      if (pending.get(key) === value) pending.delete(key);
      await releaseAttachments(value.target.workspace, value.attachmentIds);
      return false;
    }
    let task: TaskJson | undefined;
    try {
      task = (
        await call(
          target.workspace,
          {
            start: {
              sessionId: target.session,
              key: value.key,
              text: value.input.text,
              paths: value.input.paths,
              attachmentIds: value.attachmentIds,
            },
          },
          controller.signal
        )
      ).task;
    } catch (error) {
      if (error instanceof ApiError && error.rejected) {
        const receipt = await lookup(value);
        if (receipt?.id) {
          await forget(value, receipt, startedEpoch);
          return true;
        }
        if (pending.get(key) === value) pending.delete(key);
        await releaseAttachments(target.workspace, value.attachmentIds);
      }
      throw error;
    }
    if (!task?.id) throw failure('invalid_response');
    await forget(value, task, startedEpoch);
    return startedEpoch === epoch;
  } catch (error) {
    if (startedEpoch === epoch && !controller.signal.aborted) throw error;
    return false;
  } finally {
    if (controllers.get(key) === controller) controllers.delete(key);
    if (inflight.get(key) === completion) {
      sending.delete(key);
      inflight.delete(key);
    }
    done();
  }
}
export function pendingSubmission(target: Target) {
  return pending.has(targetKey(target));
}
export function clearPending() {
  epoch++;
  for (const c of controllers.values()) c.abort();
  for (const readers of reading.values()) for (const controller of readers) controller.abort();
  reading.clear();
  controllers.clear();
  pending.clear();
  sending.clear();
}

export async function awaitSubmission(target: Target) {
  await inflight.get(targetKey(target));
}
export function clearPendingTarget(target: Target) {
  const key = targetKey(target);
  controllers.get(key)?.abort();
  for (const controller of reading.get(key) ?? []) controller.abort();
  reading.delete(key);
  const value = pending.get(key);
  pending.delete(key);
  if (value) void releaseAttachments(value.target.workspace, value.attachmentIds);
}
