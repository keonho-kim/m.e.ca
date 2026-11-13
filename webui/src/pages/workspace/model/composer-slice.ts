// Purpose: Admit atomic drafts and follow-ups without retaining discarded input records.
// Pattern: Zustand mutations with weak identity-bound byte accounting.
// Usage: Workspace owns the live records; cached totals release them after reset or deletion.
// Related: webui/src/pages/workspace/model/input-usage.ts, webui/src/pages/workspace/model/types.ts.
import type { StateCreator } from 'zustand';
import { reportError } from '@/entities/server-session';
import {
  attachmentBytes,
  inputBytes,
  rememberInputBytes,
} from '@/pages/workspace/model/input-usage';
import { emptyInput, targetKey, type WorkspaceState } from '@/pages/workspace/model/types';
import { failure } from '@/shared/lib/errors';
import { editedBytes } from '@/shared/lib/utf8';

type ComposerSlice = Pick<
  WorkspaceState,
  'inputs' | 'queues' | 'pausedQueues' | 'patchInput' | 'enqueue' | 'removeQueue' | 'updateQueue'
>;
const textLimit = 32 * 1024 * 1024;
const imageLimit = 128 * 1024 * 1024;
function sameItems<T>(left: T[], right: T[]) {
  return (
    left === right ||
    (left.length === right.length && left.every((value, index) => value === right[index]))
  );
}
export const createComposerSlice: StateCreator<WorkspaceState, [], [], ComposerSlice> = (
  set,
  get
) => {
  const inputTotals = new WeakMap<WorkspaceState['inputs'], number>();
  const queueTotals = new WeakMap<WorkspaceState['queues'], { bytes: number; count: number }>();
  let revision = 0;
  const inputTotal = (inputs: WorkspaceState['inputs']) => {
    let bytes = inputTotals.get(inputs);
    if (bytes === undefined) {
      bytes = Object.values(inputs).reduce((total, input) => total + inputBytes(input), 0);
      inputTotals.set(inputs, bytes);
    }
    return bytes;
  };
  const queueTotal = (queues: WorkspaceState['queues']) => {
    let usage = queueTotals.get(queues);
    if (!usage) {
      usage = { bytes: 0, count: 0 };
      for (const queue of Object.values(queues)) {
        usage.count += queue.length;
        for (const input of queue) usage.bytes += inputBytes(input);
      }
      queueTotals.set(queues, usage);
    }
    return usage;
  };
  const reject = () => {
    reportError(failure('queue_limit'));
    return false;
  };
  return {
    inputs: {},
    queues: {},
    pausedQueues: {},
    patchInput: (target, patch, edit) => {
      const key = targetKey(target),
        state = get(),
        old = state.inputs[key] ?? emptyInput();
      if (
        (patch.text === undefined ||
          (!edit && patch.text === old.text) ||
          (edit?.revision === old.revision &&
            patch.text.length === old.text.length &&
            old.text.slice(edit.start, edit.end) === edit.inserted)) &&
        (patch.paths === undefined || sameItems(patch.paths, old.paths)) &&
        (patch.files === undefined || sameItems(patch.files, old.files))
      )
        return true;
      revision = Math.max(revision, old.revision) + 1;
      const next = { ...old, ...patch, revision };
      const bytes =
        patch.text === undefined || (!edit && patch.text === old.text)
          ? inputBytes(old)
          : editedBytes(
              old.text,
              next.text,
              inputBytes(old),
              edit?.revision === old.revision ? edit : undefined
            );
      const total = inputTotal(state.inputs) - inputBytes(old) + bytes;
      if (total > textLimit) return reject();
      const nextInputs = { ...state.inputs };
      if (!next.text && !next.paths.length && !next.files.length) delete nextInputs[key];
      else nextInputs[key] = next;
      if (
        patch.files !== undefined &&
        attachmentBytes({ inputs: nextInputs, queues: state.queues }) > imageLimit
      )
        return reject();
      rememberInputBytes(next, bytes);
      inputTotals.set(nextInputs, total);
      set({ inputs: nextInputs });
      return true;
    },
    enqueue: (target, input) => {
      const key = targetKey(target),
        state = get(),
        usage = queueTotal(state.queues),
        bytes = inputBytes(input);
      if (usage.count >= 256 || usage.bytes + bytes > textLimit) return reject();
      const item = {
        ...input,
        paths: [...input.paths],
        files: [...input.files],
        id: crypto.randomUUID(),
      };
      rememberInputBytes(item, bytes);
      const nextQueues = { ...state.queues, [key]: [...(state.queues[key] ?? []), item] };
      const nextInputs = { ...state.inputs },
        old = state.inputs[key];
      if (old) {
        revision = Math.max(revision, old.revision) + 1;
        const cleared = { ...old, text: '', files: [], revision };
        rememberInputBytes(cleared, 0);
        if (cleared.paths.length) nextInputs[key] = cleared;
        else delete nextInputs[key];
      }
      if (attachmentBytes({ inputs: nextInputs, queues: nextQueues }) > imageLimit) return reject();
      inputTotals.set(nextInputs, inputTotal(state.inputs) - (old ? inputBytes(old) : 0));
      queueTotals.set(nextQueues, { bytes: usage.bytes + bytes, count: usage.count + 1 });
      set({ inputs: nextInputs, queues: nextQueues });
      return true;
    },
    removeQueue: (target, id) => {
      const key = targetKey(target),
        state = get(),
        list = state.queues[key];
      const item = list?.find((value) => value.id === id);
      if (!list || !item) return;
      const usage = queueTotal(state.queues);
      const next = list.filter((value) => value.id !== id),
        nextQueues = { ...state.queues };
      if (next.length) nextQueues[key] = next;
      else delete nextQueues[key];
      queueTotals.set(nextQueues, {
        bytes: usage.bytes - inputBytes(item),
        count: usage.count - 1,
      });
      set({ queues: nextQueues });
    },
    updateQueue: (target, id, text) => {
      const state = get(),
        key = targetKey(target),
        list = state.queues[key];
      const old = list?.find((value) => value.id === id);
      if (!old || !list || (!text.trim() && !old.files.length)) return false;
      if (text === old.text) return true;
      const usage = queueTotal(state.queues),
        bytes = editedBytes(old.text, text, inputBytes(old)),
        total = usage.bytes - inputBytes(old) + bytes;
      if (total > textLimit) return reject();
      const item = { ...old, text };
      rememberInputBytes(item, bytes);
      const nextQueues = {
        ...state.queues,
        [key]: list.map((value) => (value === old ? item : value)),
      };
      queueTotals.set(nextQueues, { bytes: total, count: usage.count });
      set({ queues: nextQueues });
      return true;
    },
  };
};
