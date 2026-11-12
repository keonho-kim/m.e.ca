// Purpose: Apply native snapshots and ordered live events within the workspace.
// Pattern: Zustand stream projection slice.
// Usage: The workspace store composes this state with input and view slices.
// Related: webui/src/pages/workspace/model/stream-projection.ts.

import type { StateCreator } from 'zustand';
import { setServerLanguage } from '@/entities/server-session';
import { eventIdentities } from '@/pages/workspace/model/chat-events';
import { reached } from '@/pages/workspace/model/history-handoff';
import { applyEvents, snapshotDrafts } from '@/pages/workspace/model/stream-projection';
import { isActiveTask, trimTasks } from '@/pages/workspace/model/task-state';
import type { WorkspaceState } from '@/pages/workspace/model/types';

type StreamSlice = Pick<
  WorkspaceState,
  | 'replayCursor'
  | 'completionThrough'
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
  replayCursor: '',
  completionThrough: '0',
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
          historyThrough: task.historyThrough,
          ...(!isActiveTask(task.status ?? 'running') ? { doneRevision: revision } : {}),
        };
    }
    trimTasks(tasks);
    set({
      historyRevision: revision,
      replayCursor: snapshot.cursor ?? '',
      completionThrough:
        snapshot.task?.code === 'completion_not_saved'
          ? null
          : (snapshot.task?.historyThrough ?? '0'),
      drafts: target.session ? snapshotDrafts(snapshot, revision) : {},
      statistics: snapshot.execution?.statistics ?? null,
      tasks,
      events: snapshot.activity ?? [],
    });
  },
  receive: (batch) => set(applyEvents(get(), batch)),
  acknowledgeHistory: (target, revision, page) => {
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
            task.doneRevision <= revision &&
            reached(page.through, task.historyThrough)
        )
        .map((task) => task.id)
    );
    for (const [key, draft] of Object.entries(drafts)) {
      if (
        draft.doneRevision === undefined ||
        draft.doneRevision > revision ||
        !reached(page.through, draft.doneThrough)
      )
        continue;
      if (drafts === state.drafts) drafts = { ...drafts };
      delete drafts[key];
      const task = state.tasks[draft.task];
      if (task && !isActiveTask(task.status) && reached(page.through, task.historyThrough))
        finished.add(draft.task);
    }
    const represented = new Set(
      (page.items ?? []).flatMap((item) => (item.event ? eventIdentities(item.event) : []))
    );
    const retained = state.events.filter(
      (event) =>
        !finished.has(event.taskId ?? '') &&
        !(
          (event.historySequence &&
            event.historySequence !== '0' &&
            reached(page.through, event.historySequence)) ||
          eventIdentities(event).some((id) => represented.has(id))
        )
    );
    const events = retained.length === state.events.length ? state.events : retained;
    if (drafts !== state.drafts || events !== state.events) set({ drafts, events });
  },
});
