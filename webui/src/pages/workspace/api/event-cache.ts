// Purpose: Apply authoritative task and approval facts before scheduling scoped metadata refreshes.
// Pattern: Event-to-query projection.
// Usage: The ordered event observer projects each batch after the workspace reducer.
// Related: webui/src/pages/workspace/api/events.ts, webui/src/pages/workspace/api/cache-refresh.ts.
import { invalidate } from '@/pages/workspace/api/cache-refresh';
import { stateOptions } from '@/pages/workspace/api/queries';
import type { EventJson, SnapshotJson, TaskPageJson } from '@/shared/api';
import { queries } from '@/shared/api';

export function projectEventCache(events: EventJson[]) {
  const sessions = new Set<string>(),
    histories = new Map<string, string[]>();
  let workspaces = false,
    settings = false;
  for (const event of events) {
    const workspace = event.workspaceId ?? '',
      session = event.sessionId ?? '';
    const target = { workspace, session },
      started = event.kind === 'request_started',
      finished = event.kind === 'request_finished';
    const approval = event.kind === 'approval_required' || event.kind === 'approval_closed';
    const lifecycle = ['session_opened', 'session_closed', 'session_deleted'].includes(
      event.kind ?? ''
    );
    if (started || finished || approval) {
      queries.setQueryData<SnapshotJson>(stateOptions(target).queryKey, (old) => {
        if (!old) return old;
        const task = {
          id: event.taskId,
          sessionId: session,
          status: event.status ?? (started ? 'running' : 'answered'),
          outcome: event.outcome?.status,
          code: event.errorCode,
        };
        const id = event.approvalId ?? event.approval?.id;
        const approvals = approval
          ? [
              ...(old.approvals ?? []).filter((value) => value.id !== id),
              ...(event.kind === 'approval_required' && event.approval ? [event.approval] : []),
            ]
          : old.approvals;
        return {
          ...old,
          approvals,
          ...(started || finished
            ? {
                task,
                tasks: [
                  ...(old.tasks ?? []).filter((value) => value.id !== task.id),
                  ...(started ? [task] : []),
                ],
                execution: started
                  ? {
                      taskId: task.id,
                      session: old.session,
                      model: old.model,
                      team: old.team,
                      startedAt: event.occurredAt,
                    }
                  : old.execution,
              }
            : {}),
        };
      });
    }
    if (started || finished) {
      queries.setQueriesData<TaskPageJson>({ queryKey: ['task-list', workspace] }, (page) =>
        page
          ? {
              ...page,
              items: page.items?.map((task) =>
                task.id === event.taskId
                  ? {
                      ...task,
                      status: event.status,
                      outcome: event.outcome?.status,
                      code: event.errorCode,
                    }
                  : task
              ),
            }
          : page
      );
      invalidate(['task-list', workspace]);
    }
    if (started || finished || approval || lifecycle) {
      sessions.add(workspace);
      workspaces = true;
    }
    if (started || finished || event.kind === 'draft_reset')
      histories.set(`${workspace}/${session}`, ['history', workspace, session]);
    if (event.kind === 'settings_changed') settings = true;
    if (event.kind === 'model_changed' || event.kind === 'model_unavailable') {
      invalidate(stateOptions(target).queryKey);
      invalidate(['capabilities']);
    }
  }
  for (const workspace of sessions) invalidate(['sessions', workspace]);
  for (const key of histories.values()) invalidate(key);
  if (workspaces) invalidate(['workspaces']);
  if (settings) {
    invalidate(['state']);
    invalidate(['capabilities']);
  }
}
