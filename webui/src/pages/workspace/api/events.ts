// Purpose: Own one resumable event connection and batch only visible text updates.
// Pattern: Native EventSource lifecycle with urgent application event delivery.
// Usage: App reconnects from a freshly scoped snapshot when selection or pause state changes.
// Related: webui/src/pages/workspace/model/stream-projection.ts, webui/src/pages/workspace/model/execution.ts.

import { reportError } from '@/entities/server-session';
import { invalidate } from '@/pages/workspace/api/cache-refresh';
import { projectEventCache } from '@/pages/workspace/api/event-cache';
import { stateOptions } from '@/pages/workspace/api/queries';
import { forgetConversation } from '@/pages/workspace/model/deletion';
import { completed } from '@/pages/workspace/model/followups';
import {
  type Target,
  type TaskView,
  targetKey,
  useWorkspaceStore,
} from '@/pages/workspace/model/store';
import { recordTask, terminal } from '@/pages/workspace/model/submission';
import type { EventJson, NotificationJson } from '@/shared/api';
import { call, queries } from '@/shared/api';
import { ApiError, failure } from '@/shared/lib/errors';

export function observe(target: Target) {
  let source: EventSource | undefined;
  let timer: ReturnType<typeof setTimeout> | undefined;
  let batch: EventJson[] = [];
  let retry: ReturnType<typeof setTimeout> | undefined,
    probing = false,
    connecting = false;
  const controller = new AbortController();
  const isClosed = () => controller.signal.aborted;
  const flush = () => {
    if (timer) clearTimeout(timer);
    timer = undefined;
    if (!batch.length) return;
    const values = batch;
    batch = [];
    useWorkspaceStore.getState().receive(values);
    projectEventCache(values);
    for (const e of values) {
      if (e.kind === 'session_deleted')
        forgetConversation({ workspace: e.workspaceId ?? '', session: e.sessionId ?? '' });
      if (e.kind === 'request_finished')
        completed(
          { workspace: e.workspaceId ?? '', session: e.sessionId ?? '' },
          e.taskId ?? '',
          e.outcome?.status ?? '',
          e.errorCode ?? ''
        );
    }
  };
  async function connect() {
    if (isClosed() || connecting || (source && source.readyState !== EventSource.CLOSED)) return;
    if (retry) {
      clearTimeout(retry);
      retry = undefined;
    }
    connecting = true;
    probing = false;
    const state = useWorkspaceStore.getState(),
      queueTasks = new Map<string, TaskView>();
    for (const task of Object.values(state.tasks)) {
      const key = targetKey(task),
        previous = queueTasks.get(key);
      if (!state.queues[key]?.length) continue;
      // An interrupted reconciliation may already have observed a terminal snapshot.
      // Prefer an active task; otherwise confirm the newest known predecessor.
      if (!previous || terminal(previous.status) || !terminal(task.status))
        queueTasks.set(key, task);
    }
    try {
      useWorkspaceStore.getState().set({ streamStatus: 'connecting' });
      const snapshot = await queries.fetchQuery({ ...stateOptions(target), staleTime: 0 });
      if (isClosed()) return;
      if (!snapshot.cursor) throw failure('invalid_response');
      useWorkspaceStore.getState().bootstrap(target, snapshot);
      if (target.session) invalidate(['history', target.workspace, target.session]);
      const cursor = snapshot.cursor;
      const events = new EventSource(
        `/v1/events?cursor=${encodeURIComponent(cursor)}&workspace_id=${encodeURIComponent(target.workspace)}&session_id=${encodeURIComponent(target.session)}`
      );
      source = events;
      events.onopen = () => {
        if (isClosed() || source !== events) return;
        useWorkspaceStore.getState().set({ streamStatus: 'connected' });
      };
      events.onerror = () => {
        if (isClosed() || source !== events) return;
        useWorkspaceStore.getState().set({ streamStatus: 'reconnecting' });
        if (!probing) {
          probing = true;
          void call('', { capabilities: {} }, controller.signal)
            .then(() => {
              if (events.readyState === EventSource.CLOSED && !isClosed() && source === events)
                retry = setTimeout(() => void connect(), 3000);
            })
            .catch((error) => {
              if (
                !isClosed() &&
                source === events &&
                error instanceof ApiError &&
                error.code === 'unauthorized'
              )
                reportError(error);
            })
            .finally(() => {
              if (source === events) probing = false;
            });
        }
      };
      events.addEventListener('snapshot', (raw) => {
        if (isClosed()) return;
        try {
          const snapshot = (JSON.parse(raw.data) as NotificationJson).snapshot;
          if (!snapshot?.cursor || snapshot.workspaceId !== target.workspace)
            throw failure('invalid_response');
          useWorkspaceStore.getState().bootstrap(target, snapshot);
          queries.setQueryData(stateOptions(target).queryKey, snapshot);
          void queries.invalidateQueries({ queryKey: ['workspaces'] });
          void queries.invalidateQueries({
            queryKey: ['history', target.workspace, target.session],
          });
        } catch (error) {
          reportError(error);
          events.close();
        }
      });
      events.addEventListener('event', (raw) => {
        if (isClosed()) return;
        let n: NotificationJson;
        try {
          n = JSON.parse(raw.data) as NotificationJson;
        } catch (error) {
          reportError(error);
          events.close();
          return;
        }
        const e = n.event;
        if (!e || e.kind === 'heartbeat') return;
        if (
          e.kind === 'assistant_delta' &&
          (e.workspaceId !== target.workspace || e.sessionId !== target.session)
        )
          return;
        batch.push(e);
        if (e.kind !== 'assistant_delta' || batch.length >= 128) flush();
        else if (!timer)
          timer = setTimeout(flush, typeof document !== 'undefined' && document.hidden ? 200 : 33);
      });
      for (const previous of queueTasks.values()) {
        if (isClosed()) return;
        await call(previous.workspace, { taskStatus: { taskId: previous.id } }, controller.signal)
          .then((value) => {
            if (isClosed() || !value.task) return;
            recordTask(previous, value.task);
            if (terminal(value.task.status ?? ''))
              completed(previous, previous.id, value.task.outcome ?? '', value.task.code ?? '');
          })
          .catch((error) => {
            if (!isClosed()) reportError(error);
          });
      }
    } catch (error) {
      if (!isClosed()) {
        useWorkspaceStore.getState().set({ streamStatus: 'reconnecting' });
        reportError(error);
        if (
          !(error instanceof ApiError) ||
          ['network', 'unavailable', 'resource_exhausted'].includes(error.code)
        )
          retry = setTimeout(() => void connect(), 3000);
      }
    } finally {
      connecting = false;
    }
  }
  void connect();
  return () => {
    controller.abort();
    source?.close();
    if (retry) clearTimeout(retry);
    if (timer) clearTimeout(timer);
    batch = [];
  };
}
