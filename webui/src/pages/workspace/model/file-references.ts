// Purpose: Serialize reference changes and let submissions wait for durable selection results.
// Pattern: Per-conversation cancellable mutation queue.
// Usage: File checkboxes and composer reference removal share selectReference.
// Related: webui/src/pages/workspace/ui/files/files.tsx, webui/src/pages/workspace/ui/composer/composer.tsx.
import { reportError } from '@/entities/server-session';
import {
  emptyInput,
  type Target,
  targetKey,
  useWorkspaceStore,
} from '@/pages/workspace/model/store';
import { call } from '@/shared/api';

const pending = new Map<string, Promise<void>>();
const controllers = new Set<AbortController>();

export function selectReference(target: Target, path: string, selected: boolean) {
  const key = targetKey(target),
    state = useWorkspaceStore.getState();
  const old = state.inputs[key] ?? emptyInput();
  state.patchInput(target, {
    paths: selected ? [...new Set([...old.paths, path])] : old.paths.filter((v) => v !== path),
  });
  const controller = new AbortController();
  controllers.add(controller);
  const previous = pending.get(key) ?? Promise.resolve();
  const operation = previous.then(async () => {
    await call(
      target.workspace,
      { selectFile: { sessionId: target.session, path, selected } },
      controller.signal
    );
  });
  // oxlint-disable-next-line typescript/no-misused-promises -- Promise.finally awaits async cleanup despite lib.d.ts's void callback type.
  const result = operation.finally(async () => {
    controllers.delete(controller);
    if (pending.get(key) !== result) return;
    pending.delete(key);
    if (controller.signal.aborted) return;
    try {
      const reply = await call(
        target.workspace,
        { fileSelections: { sessionId: target.session } },
        controller.signal
      );
      if (!pending.has(key))
        useWorkspaceStore
          .getState()
          .patchInput(target, { paths: reply.fileSelections?.paths ?? [] });
    } catch (error) {
      reportError(error);
    }
  });
  pending.set(key, result);
  return result;
}
export async function waitSelections(target: Target) {
  await pending.get(targetKey(target));
}
export function cancelSelections() {
  for (const controller of controllers) controller.abort();
  controllers.clear();
  pending.clear();
}
