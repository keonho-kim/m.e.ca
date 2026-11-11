// Purpose: Own one resumable event connection and batch only visible text updates.
// Pattern: Native EventSource lifecycle with urgent application event delivery.
// Usage: App reconnects from a freshly scoped snapshot when selection or connectivity changes.
// Related: webui/src/pages/workspace/model/stream-projection.ts, webui/src/pages/workspace/model/execution.ts.

import { reportError } from '@/entities/server-session';
import { invalidate } from '@/pages/workspace/api/cache-refresh';
import { projectEventCache } from '@/pages/workspace/api/event-cache';
import { stateOptions } from '@/pages/workspace/api/queries';
import { readSnapshot, snapshotMetadata } from '@/pages/workspace/api/snapshots';
import { forgetConversation } from '@/pages/workspace/model/deletion';
import { refreshReferences } from '@/pages/workspace/model/file-references';
import { completed } from '@/pages/workspace/model/followups';
import {
  type Target,
  type TaskView,
  targetKey,
  useWorkspaceStore,
} from '@/pages/workspace/model/store';
import { recordTask, terminal } from '@/pages/workspace/model/submission';
import type { EventJson, NotificationJson, SnapshotJson } from '@/shared/api';
import { call, queries, subscribeEventStream } from '@/shared/api';
import { ApiError, failure } from '@/shared/lib/errors';

export function observe(target: Target) {
  let timer: ReturnType<typeof setTimeout> | undefined;
  let batch: EventJson[] = [];
  let initialized = false;
  const discardBatch = () => {
    if (timer) clearTimeout(timer);
    timer = undefined;
    batch = [];
  };
  const flush = (control?: EventJson) => {
    const values = batch;
    discardBatch();
    if (control) values.push(control);
    if (!values.length) return;
    useWorkspaceStore.getState().receive(values);
    if (!control) return;
    projectEventCache([control]);
    const owner = { workspace: control.workspaceId ?? '', session: control.sessionId ?? '' };
    if (control.kind === 'session_deleted') forgetConversation(owner);
    if (control.kind === 'request_finished')
      completed(
        owner,
        control.taskId ?? '',
        control.outcome?.status ?? '',
        control.errorCode ?? ''
      );
  };
  const acceptSnapshot = (snapshot: SnapshotJson) => {
    discardBatch();
    useWorkspaceStore.getState().bootstrap(target, snapshot);
    queries.setQueryData(stateOptions(target).queryKey, snapshotMetadata(snapshot));
    if (initialized && target.session) void refreshReferences(target).catch(reportError);
    initialized = true;
    if (target.session) {
      invalidate(['history', target.workspace, target.session, '0']);
      invalidate(['session-usage', target.workspace, target.session]);
    }
  };
  const acceptEvent = (raw: string) => {
    const event = (JSON.parse(raw) as NotificationJson).event;
    if (!event || event.kind === 'heartbeat') return;
    if (event.kind !== 'assistant_delta') {
      flush(event);
      return;
    }
    if (event.workspaceId !== target.workspace || event.sessionId !== target.session) return;
    batch.push(event);
    if (batch.length >= 128) flush();
    else if (!timer)
      timer = setTimeout(flush, typeof document !== 'undefined' && document.hidden ? 200 : 33);
  };
  const close = subscribeEventStream({
    connect: async (signal) => {
      const pending = queuedPredecessors();
      const snapshot = await readSnapshot(target, signal);
      signal.throwIfAborted();
      if (!snapshot.cursor || snapshot.workspaceId !== target.workspace)
        throw failure('invalid_response');
      acceptSnapshot(snapshot);
      void reconcileQueues(pending, signal);
      return {
        url: `/v1/events?cursor=${encodeURIComponent(snapshot.cursor)}&workspace_id=${encodeURIComponent(target.workspace)}&session_id=${encodeURIComponent(target.session)}`,
        events: {
          event: acceptEvent,
          snapshot: (raw: string) => {
            const next = (JSON.parse(raw) as NotificationJson).snapshot;
            if (!next?.cursor || next.workspaceId !== target.workspace)
              throw failure('invalid_response');
            acceptSnapshot(next);
            invalidate(['workspaces']);
          },
        },
      };
    },
    status: (streamStatus) => useWorkspaceStore.getState().set({ streamStatus }),
    failed: (error) => {
      reportError(error);
      return (
        !(error instanceof ApiError) ||
        ['network', 'unavailable', 'resource_exhausted', 'invalid_response'].includes(error.code)
      );
    },
  });
  return () => {
    close();
    discardBatch();
  };
}

function queuedPredecessors() {
  const state = useWorkspaceStore.getState(),
    tasks = new Map<string, TaskView>();
  for (const task of Object.values(state.tasks)) {
    const key = targetKey(task),
      previous = tasks.get(key);
    if (
      state.queues[key]?.length &&
      (!previous || terminal(previous.status) || !terminal(task.status))
    )
      tasks.set(key, task);
  }
  return [...tasks.values()];
}

async function reconcileQueues(tasks: TaskView[], signal: AbortSignal) {
  const stopped = () => signal.aborted;
  for (const previous of tasks) {
    if (stopped()) return;
    try {
      const reply = await call(previous.workspace, { taskStatus: { taskId: previous.id } }, signal);
      if (stopped() || !reply.task) continue;
      recordTask(previous, reply.task);
      if (terminal(reply.task.status ?? ''))
        completed(previous, previous.id, reply.task.outcome ?? '', reply.task.code ?? '');
    } catch (error) {
      if (!stopped()) reportError(error);
    }
  }
}
