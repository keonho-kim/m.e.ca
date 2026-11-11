// Purpose: Persist selected file references and restore authoritative conversation paths.
// Pattern: Optimistic state projection over the workspace operation queue.
// Usage: Composer and file menus select references; file effects and events refresh them.
// Related: webui/src/pages/workspace/model/reference-queue.ts, webui/src/pages/workspace/model/file-mutations.ts.

import {
  blockReferences,
  enqueueReference,
  referenceEpoch,
} from '@/pages/workspace/model/reference-queue';
import {
  emptyInput,
  type Target,
  targetKey,
  useWorkspaceStore,
} from '@/pages/workspace/model/store';
import { call } from '@/shared/api';
import { failure } from '@/shared/lib/errors';
export async function restoreReferences(target: Target, signal: AbortSignal, version: number) {
  const reply = await call(
    target.workspace,
    { fileSelections: { sessionId: target.session } },
    signal
  );
  signal.throwIfAborted();
  if (!reply.fileSelections) throw failure('invalid_response');
  if (version === referenceEpoch())
    useWorkspaceStore.getState().patchInput(target, { paths: reply.fileSelections.paths ?? [] });
}

export function selectReference(target: Target, path: string, selected: boolean) {
  const version = referenceEpoch();
  const state = useWorkspaceStore.getState();
  const old = state.inputs[targetKey(target)] ?? emptyInput();
  state.patchInput(target, {
    paths: selected ? [...new Set([...old.paths, path])] : old.paths.filter((v) => v !== path),
  });
  return enqueueReference(target, async (signal, isLast) => {
    let failed: { error: unknown } | undefined;
    try {
      await call(
        target.workspace,
        { selectFile: { sessionId: target.session, path, selected } },
        signal
      );
    } catch (error) {
      failed = { error };
    }
    if (isLast()) {
      try {
        await restoreReferences(target, signal, version);
        if (!failed && version === referenceEpoch()) blockReferences(target);
      } catch (error) {
        if (version === referenceEpoch()) blockReferences(target, failure('references_pending'));
        failed ??= { error };
      }
    }
    if (failed) throw failed.error;
  });
}

export function refreshReferences(target: Target, external?: AbortSignal) {
  const version = referenceEpoch();
  return enqueueReference(
    target,
    async (signal, isLast) => {
      if (!isLast()) return;
      try {
        await restoreReferences(target, signal, version);
      } catch (error) {
        if (signal.aborted || version !== referenceEpoch()) return;
        blockReferences(target, failure('references_pending'));
        throw error;
      }
    },
    30000,
    external
  );
}
