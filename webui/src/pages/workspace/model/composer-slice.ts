// Purpose: Own incremental text budgets and atomic draft/follow-up mutations.
// Pattern: Zustand workspace slice with identity-bound budget reconciliation.
// Usage: The workspace store composes this state; external resets invalidate cached totals.
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
import { utf8Bytes } from '@/shared/lib/utf8';

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
  let inputs: WorkspaceState['inputs'] | undefined;
  let queues: WorkspaceState['queues'] | undefined;
  let inputTotal = 0,
    queueTotal = 0,
    queueCount = 0,
    revision = 0;
  const reconcile = (state: WorkspaceState) => {
    if (inputs !== state.inputs) {
      inputs = state.inputs;
      inputTotal = Object.values(inputs).reduce((n, value) => n + inputBytes(value), 0);
    }
    if (queues !== state.queues) {
      queues = state.queues;
      queueTotal = 0;
      queueCount = 0;
      for (const list of Object.values(queues)) {
        queueCount += list.length;
        for (const value of list) queueTotal += inputBytes(value);
      }
    }
  };
  const reject = () => {
    reportError(failure('queue_limit'));
    return false;
  };
  return {
    inputs: {},
    queues: {},
    pausedQueues: {},
    patchInput: (target, patch) => {
      const key = targetKey(target),
        state = get(),
        old = state.inputs[key] ?? emptyInput();
      if (
        (patch.text === undefined || patch.text === old.text) &&
        (patch.paths === undefined || sameItems(patch.paths, old.paths)) &&
        (patch.files === undefined || sameItems(patch.files, old.files))
      )
        return true;
      reconcile(state);
      revision = Math.max(revision, old.revision) + 1;
      const next = { ...old, ...patch, revision };
      const bytes =
        patch.text === undefined || patch.text === old.text
          ? inputBytes(old)
          : utf8Bytes(next.text);
      const total = inputTotal - inputBytes(old) + bytes;
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
      inputs = nextInputs;
      inputTotal = total;
      set({ inputs: nextInputs });
      return true;
    },
    enqueue: (target, input) => {
      const key = targetKey(target),
        state = get();
      reconcile(state);
      const bytes = inputBytes(input);
      if (queueCount >= 256 || queueTotal + bytes > textLimit) return reject();
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
      inputs = nextInputs;
      inputTotal -= old ? inputBytes(old) : 0;
      queues = nextQueues;
      queueTotal += bytes;
      queueCount++;
      set({ inputs: nextInputs, queues: nextQueues });
      return true;
    },
    removeQueue: (target, id) => {
      const key = targetKey(target),
        state = get(),
        list = state.queues[key];
      const item = list?.find((value) => value.id === id);
      if (!list || !item) return;
      reconcile(state);
      const next = list.filter((value) => value.id !== id),
        nextQueues = { ...state.queues };
      if (next.length) nextQueues[key] = next;
      else delete nextQueues[key];
      queues = nextQueues;
      queueTotal -= inputBytes(item);
      queueCount--;
      set({ queues: nextQueues });
    },
    updateQueue: (target, id, text) => {
      const state = get(),
        key = targetKey(target),
        list = state.queues[key];
      const old = list?.find((value) => value.id === id);
      if (!old || !list || !text.trim()) return false;
      if (text === old.text) return true;
      reconcile(state);
      const bytes = utf8Bytes(text),
        total = queueTotal - inputBytes(old) + bytes;
      if (total > textLimit) return reject();
      const item = { ...old, text };
      rememberInputBytes(item, bytes);
      const nextQueues = {
        ...state.queues,
        [key]: list.map((value) => (value === old ? item : value)),
      };
      queues = nextQueues;
      queueTotal = total;
      set({ queues: nextQueues });
      return true;
    },
  };
};
