// Purpose: Remove deleted conversation content from browser caches and transient request owners.
// Pattern: Explicit local deletion transition.
// Usage: Sidebar receipts and server deletion events share this idempotent cleanup.
// Related: webui/src/pages/workspace/api/queries.ts, webui/src/pages/workspace/model/store.ts, webui/src/pages/workspace/model/submission.ts.

import { clearFollowupsTarget } from '@/pages/workspace/model/followups';
import { cancelReferenceTarget } from '@/pages/workspace/model/reference-queue';
import { type Target, targetKey, useWorkspaceStore } from '@/pages/workspace/model/store';
import { clearPendingTarget } from '@/pages/workspace/model/submission';
import { queries } from '@/shared/api';
export function forgetConversation(target: Target) {
  cancelReferenceTarget(target);
  clearPendingTarget(target);
  clearFollowupsTarget(target);
  const state = useWorkspaceStore.getState(),
    key = targetKey(target);
  const inputs = { ...state.inputs },
    queues = { ...state.queues },
    pausedQueues = { ...state.pausedQueues };
  delete inputs[key];
  delete queues[key];
  delete pausedQueues[key];
  const selected =
    state.selection.workspace === target.workspace && state.selection.session === target.session;
  const reviewTarget =
    state.reviewTarget?.workspace === target.workspace &&
    state.reviewTarget.session === target.session
      ? null
      : state.reviewTarget;
  state.set({
    inputs,
    queues,
    pausedQueues,
    reviewTarget,
    removedTarget: target,
    ...(selected ? { drafts: {}, events: [], statistics: null } : {}),
    tasks: Object.fromEntries(
      Object.entries(state.tasks).filter(
        ([, task]) => task.workspace !== target.workspace || task.session !== target.session
      )
    ),
  });
  queries.removeQueries({
    predicate: (query) =>
      ['history', 'state', 'original', 'thumbnail', 'session-usage'].includes(
        String(query.queryKey[0])
      ) &&
      query.queryKey[1] === target.workspace &&
      query.queryKey[2] === target.session,
  });
}
