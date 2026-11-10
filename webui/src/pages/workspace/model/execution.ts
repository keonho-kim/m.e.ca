// Purpose: Distinguish direct steering from separately admitted browser follow-ups.
// Pattern: Explicit input dispatch.
// Usage: Composer uses send; queued requests use submission.startRequest directly.
// Related: webui/src/pages/workspace/model/submission.ts, webui/src/pages/workspace/model/followups.ts.

import { changed } from '@/pages/workspace/api/queries';
import { clearFollowups } from '@/pages/workspace/model/followups';
import {
  emptyInput,
  type Input,
  type Target,
  targetKey,
  useWorkspaceStore,
} from '@/pages/workspace/model/store';
import { clearPending, startRequest } from '@/pages/workspace/model/submission';
import { activeTask } from '@/pages/workspace/model/task-state';
import { call } from '@/shared/api';
import { failure } from '@/shared/lib/errors';
export async function send(target: Target, input: Input, reserve = false): Promise<boolean> {
  if (!target.session) throw failure('no_session');
  if (!input.text.trim() && !input.files.length) throw failure('input_empty');
  const task = activeTask(target, useWorkspaceStore.getState().tasks);
  if (task && (reserve || input.files.length))
    return useWorkspaceStore.getState().enqueue(target, input);
  if (task) {
    await call(target.workspace, {
      steer: { taskId: task.id, text: input.text, paths: input.paths },
    });
    const current = useWorkspaceStore.getState().inputs[targetKey(target)] ?? emptyInput();
    if (current.revision === input.revision)
      useWorkspaceStore.getState().patchInput(target, { text: '', files: [] });
    changed(target);
    return true;
  }
  return startRequest(target, input);
}
export function clearSubmissions() {
  clearPending();
  clearFollowups();
}
