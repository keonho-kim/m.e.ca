// Purpose: Apply native snapshots and ordered live events within the workspace.
// Pattern: Zustand stream projection slice.
// Usage: The workspace store composes this state with input and view slices.
// Related: webui/src/pages/workspace/model/stream-projection.ts.

import type { StateCreator } from 'zustand';
import { setServerLanguage } from '@/entities/server-session';
import { applyEvents, snapshotDrafts } from '@/pages/workspace/model/stream-projection';
import { trimTasks } from '@/pages/workspace/model/task-state';
import type { WorkspaceState } from '@/pages/workspace/model/types';

type StreamSlice = Pick<
  WorkspaceState,
  'historyRevision' | 'statistics' | 'drafts' | 'tasks' | 'events' | 'bootstrap' | 'receive'
>;
export const createStreamSlice: StateCreator<WorkspaceState, [], [], StreamSlice> = (set, get) => ({
  historyRevision: 0,
  statistics: null,
  drafts: {},
  tasks: {},
  events: [],
  bootstrap: (target, snapshot) => {
    if (snapshot.language) setServerLanguage(snapshot.language);
    const tasks = { ...get().tasks };
    for (const [id, task] of Object.entries(tasks))
      if (target.session && task.workspace === target.workspace && task.session === target.session)
        delete tasks[id];
    for (const task of [...(snapshot.tasks ?? []), ...(snapshot.task ? [snapshot.task] : [])]) {
      if (task.id)
        tasks[task.id] = {
          id: task.id,
          workspace: target.workspace,
          session: task.sessionId ?? target.session,
          status: task.status ?? 'running',
        };
    }
    trimTasks(tasks);
    set({
      drafts: target.session ? snapshotDrafts(snapshot) : {},
      statistics: snapshot.execution?.statistics ?? null,
      tasks,
      events: snapshot.activity ?? [],
    });
  },
  receive: (batch) => set(applyEvents(get(), batch)),
});
