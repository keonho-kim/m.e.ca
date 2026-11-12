// Purpose: Apply native snapshots and ordered live events within the workspace.
// Pattern: Zustand stream projection slice.
// Usage: The workspace store composes this state with input and view slices.
// Related: webui/src/pages/workspace/model/stream-projection.ts.

import type { StateCreator } from 'zustand';
import { setServerLanguage } from '@/entities/server-session';
import { applyEvents, snapshotDrafts } from '@/pages/workspace/model/stream-projection';
import { isActiveTask, trimTasks } from '@/pages/workspace/model/task-state';
import type { WorkspaceState } from '@/pages/workspace/model/types';

type StreamSlice = Pick<
  WorkspaceState,
  | 'historyRevision'
  | 'statistics'
  | 'drafts'
  | 'tasks'
  | 'events'
  | 'bootstrap'
  | 'receive'
  | 'acknowledgeHistory'
>;
export const createStreamSlice: StateCreator<WorkspaceState, [], [], StreamSlice> = (set, get) => ({
  historyRevision: 0,
  statistics: null,
  drafts: {},
  tasks: {},
  events: [],
  bootstrap: (target, snapshot) => {
    if (snapshot.language) setServerLanguage(snapshot.language);
    const revision = get().historyRevision + 1;
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
          ...(!isActiveTask(task.status ?? 'running') ? { doneRevision: revision } : {}),
        };
    }
    trimTasks(tasks);
    set({
      historyRevision: revision,
      drafts: target.session ? snapshotDrafts(snapshot, revision) : {},
      statistics: snapshot.execution?.statistics ?? null,
      tasks,
      events: snapshot.activity ?? [],
    });
  },
  receive: (batch) => set(applyEvents(get(), batch)),
  acknowledgeHistory: (target, revision) => {
    const state = get();
    if (
      state.selection.workspace !== target.workspace ||
      state.selection.session !== target.session
    )
      return;
    let drafts = state.drafts;
    const finished = new Set(
      Object.values(state.tasks)
        .filter(
          (task) =>
            task.workspace === target.workspace &&
            task.session === target.session &&
            !isActiveTask(task.status) &&
            task.doneRevision !== undefined &&
            task.doneRevision <= revision
        )
        .map((task) => task.id)
    );
    for (const [key, draft] of Object.entries(drafts)) {
      if (draft.doneRevision === undefined || draft.doneRevision > revision) continue;
      if (drafts === state.drafts) drafts = { ...drafts };
      delete drafts[key];
      const task = state.tasks[draft.task];
      if (task && !isActiveTask(task.status)) finished.add(draft.task);
    }
    const events =
      finished.size && state.events.some((event) => finished.has(event.taskId ?? ''))
        ? state.events.filter((event) => !finished.has(event.taskId ?? ''))
        : state.events;
    if (drafts !== state.drafts || events !== state.events) set({ drafts, events });
  },
});
