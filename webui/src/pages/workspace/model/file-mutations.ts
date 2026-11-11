// Purpose: Recover file-effect receipts after lost replies without replaying filesystem operations.
// Pattern: Exactly-once intent submission and authoritative result projection.
// Usage: File menus enqueue one keyed mutation before submissions capture selected references.
// Related: webui/src/pages/workspace/model/reference-queue.ts, internal/commands/server/file_mutation_receipt.go.

import { restoreReferences } from '@/pages/workspace/model/file-references';
import { forgetMutation, rememberMutation } from '@/pages/workspace/model/mutation-correlation';
import {
  blockReferences,
  enqueueReference,
  referenceActive,
  referenceEpoch,
} from '@/pages/workspace/model/reference-queue';
import { type Target, useWorkspaceStore } from '@/pages/workspace/model/store';
import { call, type FileMutationJson, type FileMutationResultJson } from '@/shared/api';
import { ApiError, failure } from '@/shared/lib/errors';

// The operation queue owns the overall deadline. Each receipt wait releases its
// timer and abort listener before the next read or when its target is deleted.
function waitForReceipt(signal: AbortSignal) {
  signal.throwIfAborted();
  return new Promise<void>((resolve, reject) => {
    const stop = () => {
      clearTimeout(timer);
      signal.removeEventListener('abort', stop);
      reject(signal.reason);
    };
    const timer = setTimeout(() => {
      signal.removeEventListener('abort', stop);
      resolve();
    }, 500);
    signal.addEventListener('abort', stop, { once: true });
  });
}
async function recover(target: Target, key: string, signal: AbortSignal) {
  for (;;) {
    const reply = await call(
      target.workspace,
      { lookupMutation: { sessionId: target.session, key } },
      signal
    );
    if (reply.fileMutation?.state !== 'pending') return reply.fileMutation;
    await waitForReceipt(signal);
  }
}
export function mutateFile(target: Target, change: FileMutationJson) {
  const version = referenceEpoch(),
    key = crypto.randomUUID();
  return enqueueReference(
    target,
    async (signal) => {
      rememberMutation(target, key);
      let ready = false;
      try {
        let result: FileMutationResultJson | undefined;
        try {
          const reply = await call(
            target.workspace,
            { mutateFile: { ...change, sessionId: target.session, key } },
            signal
          );
          result = reply.fileMutation;
        } catch (error) {
          if (signal.aborted || (error instanceof ApiError && error.rejected)) throw error;
          result = await recover(target, key, signal);
        }
        signal.throwIfAborted();
        if (
          !result ||
          result.key !== key ||
          !['applied', 'partial', 'rejected', 'pending', 'uncertain'].includes(result.state ?? '')
        )
          throw failure('invalid_response');
        if (result.state === 'rejected')
          throw new ApiError(
            result.errorCode ?? 'operation_failed',
            result.errorMessage ?? '',
            true
          );
        ready = !!result.referencesReady && ['applied', 'partial'].includes(result.state ?? '');
        if (!ready) throw failure('references_pending');
        if (version === referenceEpoch()) {
          if (target.session)
            useWorkspaceStore.getState().patchInput(target, { paths: result.paths ?? [] });
          blockReferences(target);
        }
        return result;
      } catch (error) {
        forgetMutation(key);
        if (version === referenceEpoch() && referenceActive(target) && !ready && !signal.aborted) {
          if (error instanceof ApiError && error.rejected) {
            try {
              await restoreReferences(target, signal, version);
            } catch {
              if (referenceActive(target)) blockReferences(target, failure('references_pending'));
            }
          } else blockReferences(target, failure('references_pending'));
        }
        throw error;
      }
    },
    120000
  );
}
