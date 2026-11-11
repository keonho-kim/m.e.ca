// Purpose: Serialize reference and filesystem changes before conversation submissions capture paths.
// Pattern: Workspace-owned cancellable operation queue with explicit uncertain outcomes.
// Usage: File menus and mutations enqueue work; composer submissions await its durability barrier.
// Related: webui/src/pages/workspace/ui/files/files.tsx, webui/src/pages/workspace/ui/composer/composer.tsx.
import {
  emptyInput,
  type Target,
  targetKey,
  useWorkspaceStore,
} from '@/pages/workspace/model/store';
import { call, type FileMutationJson } from '@/shared/api';
import { ApiError, failure } from '@/shared/lib/errors';

const pending = new Map<string, Promise<void>>();
const latest = new Map<string, Promise<void>>();
const blocked = new Map<string, unknown>();
const controllers = new Map<AbortController, string>();
let epoch = 0;

function enqueue<T>(
  target: Target,
  run: (signal: AbortSignal, isLast: () => boolean) => Promise<T>,
  timeout = 30000
) {
  const controller = new AbortController();
  const key = targetKey(target);
  controllers.set(controller, key);
  const signal = controller.signal;
  let timer: ReturnType<typeof setTimeout> | undefined;
  let settled: Promise<void>;
  const operation = (pending.get(target.workspace) ?? Promise.resolve()).then(() => {
    signal.throwIfAborted();
    timer = setTimeout(() => controller.abort(), timeout);
    return run(signal, () => latest.get(key) === settled);
  });
  const result = operation.finally(() => {
    if (timer) clearTimeout(timer);
    controllers.delete(controller);
  });
  // Each caller receives its own error. Later deliberate changes still run, and
  // the submission barrier waits through the newest queued operation.
  settled = result.then(
    () => {},
    () => {}
  );
  pending.set(target.workspace, settled);
  latest.set(key, settled);
  void settled.then(() => {
    if (pending.get(target.workspace) === settled) pending.delete(target.workspace);
    if (latest.get(key) === settled) latest.delete(key);
  });
  return result;
}

async function restore(target: Target, signal: AbortSignal, version: number) {
  const reply = await call(
    target.workspace,
    { fileSelections: { sessionId: target.session } },
    signal
  );
  signal.throwIfAborted();
  if (!reply.fileSelections) throw failure('invalid_response');
  if (version === epoch)
    useWorkspaceStore.getState().patchInput(target, { paths: reply.fileSelections.paths ?? [] });
}

export function selectReference(target: Target, path: string, selected: boolean) {
  const version = epoch;
  const state = useWorkspaceStore.getState();
  const old = state.inputs[targetKey(target)] ?? emptyInput();
  state.patchInput(target, {
    paths: selected ? [...new Set([...old.paths, path])] : old.paths.filter((v) => v !== path),
  });
  return enqueue(target, async (signal, isLast) => {
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
        await restore(target, signal, version);
        if (!failed && version === epoch) blocked.delete(targetKey(target));
      } catch (error) {
        if (version === epoch) blocked.set(targetKey(target), failure('references_pending'));
        failed ??= { error };
      }
    }
    if (failed) throw failed.error;
  });
}

export function mutateFile(target: Target, change: FileMutationJson) {
  const version = epoch;
  return enqueue(
    target,
    async (signal) => {
      let ready = false;
      try {
        const reply = await call(
          target.workspace,
          {
            mutateFile: { ...change, sessionId: target.session },
          },
          signal
        );
        signal.throwIfAborted();
        const result = reply.fileMutation;
        if (!result) throw failure('invalid_response');
        ready = !!result.referencesReady;
        if (ready && version === epoch && target.session)
          useWorkspaceStore.getState().patchInput(target, { paths: result.paths ?? [] });
        if (!ready) throw failure('references_pending');
        if (result.errorCode) throw new ApiError(result.errorCode, result.errorMessage ?? '', true);
        return result.path ?? '';
      } catch (error) {
        if (version === epoch && latest.has(targetKey(target)) && !ready) {
          if (error instanceof ApiError && error.rejected) {
            try {
              await restore(target, signal, version);
            } catch {
              if (latest.has(targetKey(target)))
                blocked.set(targetKey(target), failure('references_pending'));
            }
          } else blocked.set(targetKey(target), failure('references_pending'));
        }
        throw error;
      }
    },
    120000
  );
}

export async function waitSelections(target: Target) {
  let operation = pending.get(target.workspace);
  while (operation) {
    await operation;
    operation = pending.get(target.workspace);
  }
  if (blocked.has(targetKey(target))) throw blocked.get(targetKey(target));
}

export function cancelReferenceTarget(target: Target) {
  const key = targetKey(target);
  for (const [controller, owner] of controllers) if (owner === key) controller.abort();
  latest.delete(key);
  blocked.delete(key);
}

export function cancelSelections() {
  epoch++;
  for (const controller of controllers.keys()) controller.abort();
  controllers.clear();
  pending.clear();
  latest.clear();
  blocked.clear();
}
