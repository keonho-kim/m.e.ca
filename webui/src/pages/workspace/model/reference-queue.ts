// Purpose: Own cancellable reference admission and the submission barrier for each workspace.
// Pattern: Explicit serial I/O queue with bounded active ownership.
// Usage: Reference selections and file effects share this queue; submissions await it.
// Related: webui/src/pages/workspace/model/file-references.ts, webui/src/pages/workspace/model/file-mutations.ts.
import { clearMutationCorrelations } from '@/pages/workspace/model/mutation-correlation';
import { type Target, targetKey } from '@/pages/workspace/model/store';

const pending = new Map<string, Promise<void>>();
const latest = new Map<string, Promise<void>>();
const blocked = new Map<string, unknown>();
const controllers = new Map<AbortController, string>();
let epoch = 0;

export function enqueueReference<T>(
  target: Target,
  run: (signal: AbortSignal, isLast: () => boolean) => Promise<T>,
  timeout = 30000,
  external?: AbortSignal
) {
  const controller = new AbortController();
  const key = targetKey(target);
  controllers.set(controller, key);
  const signal = external ? AbortSignal.any([controller.signal, external]) : controller.signal;
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

export const referenceEpoch = () => epoch;
export const referenceActive = (target: Target) => latest.has(targetKey(target));
export function blockReferences(target: Target, error?: unknown) {
  if (error) blocked.set(targetKey(target), error);
  else blocked.delete(targetKey(target));
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
  clearMutationCorrelations(target);
  const key = targetKey(target);
  for (const [controller, owner] of controllers) if (owner === key) controller.abort();
  latest.delete(key);
  blocked.delete(key);
}

export function cancelSelections() {
  clearMutationCorrelations();
  epoch++;
  for (const controller of controllers.keys()) controller.abort();
  controllers.clear();
  pending.clear();
  latest.clear();
  blocked.clear();
}
