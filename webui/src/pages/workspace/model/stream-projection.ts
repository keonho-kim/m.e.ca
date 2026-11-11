// Purpose: Project ordered events into bounded drafts, durable-history handoffs and task indicators.
// Pattern: Pure presentation reducer.
// Usage: Zustand applies batches without subscribing the whole application to each token.
// Related: webui/src/pages/workspace/model/store.ts, webui/src/pages/workspace/api/events.ts.

import { isHistoryBoundary, isResponseEvent } from '@/pages/workspace/model/chat-events';
import { compactDeltas } from '@/pages/workspace/model/delta-batch';
import { trimTasks } from '@/pages/workspace/model/task-state';
import type { Draft, WorkspaceState } from '@/pages/workspace/model/types';
import type { EventJson, SnapshotJson } from '@/shared/api';
import { textTail } from '@/shared/lib/utf8';

const windowChars = 128 * 1024;
const totalChars = 4 * 1024 * 1024;

export function snapshotDrafts(snapshot: SnapshotJson, revision: number) {
  const values: Record<string, Draft> = {};
  for (const d of snapshot.drafts ?? []) {
    if (d.taskId)
      values[`${d.taskId}/${d.agent ?? ''}`] = {
        task: d.taskId,
        agent: d.agent ?? '',
        generation: d.generation ?? '0',
        text: d.text ?? '',
        thinking: d.thinking ?? '',
        truncated: !!d.truncated,
        ...(d.committed ? { doneRevision: revision } : {}),
      };
  }
  return values;
}

export function applyEvents(state: WorkspaceState, batch: EventJson[]): Partial<WorkspaceState> {
  let { drafts, tasks, events, historyRevision, statistics } = state;
  for (const e of compactDeltas(batch)) {
    if (
      state.removedTarget &&
      state.removedTarget.workspace === e.workspaceId &&
      state.removedTarget.session === e.sessionId
    )
      continue;
    if (!e.taskId) continue;
    if (e.kind === 'request_started' || e.kind === 'request_finished') {
      if (tasks === state.tasks) tasks = { ...tasks };
      tasks[e.taskId] = {
        id: e.taskId,
        workspace: e.workspaceId ?? '',
        session: e.sessionId ?? '',
        status: e.status ?? 'running',
      };
    }
    if (e.workspaceId !== state.selection.workspace || e.sessionId !== state.selection.session)
      continue;
    if (e.kind === 'request_started') statistics = null;
    if (e.statistics) statistics = e.statistics;
    if (isHistoryBoundary(e)) {
      historyRevision++;
      const key = `${e.taskId}/${e.agent ?? ''}`,
        draft = drafts[key];
      if (draft && draft.doneRevision === undefined) {
        if (drafts === state.drafts) drafts = { ...drafts };
        drafts[key] = { ...draft, doneRevision: historyRevision };
      }
    }
    if (e.kind === 'draft_reset') {
      historyRevision++;
      if (drafts === state.drafts) drafts = { ...drafts };
      const key = `${e.taskId}/${e.agent ?? ''}`,
        old = drafts[key];
      if (old) {
        drafts[`${key}/${old.generation}`] = { ...old, doneRevision: historyRevision };
        delete drafts[key];
      }
    } else if (e.kind === 'request_finished') {
      historyRevision++;
      if (drafts === state.drafts) drafts = { ...drafts };
      for (const [key, draft] of Object.entries(drafts))
        if (draft.task === e.taskId) drafts[key] = { ...draft, doneRevision: historyRevision };
    } else if (e.kind === 'assistant_delta') {
      if (drafts === state.drafts) drafts = { ...drafts };
      const key = `${e.taskId}/${e.agent ?? ''}`,
        generation = e.generation ?? '0';
      const old = drafts[key];
      const value =
        old && old.generation === generation && !e.reset
          ? old
          : {
              task: e.taskId,
              agent: e.agent ?? '',
              generation,
              text: '',
              thinking: '',
              truncated: false,
            };
      const text = value.text + (e.text ?? ''),
        thinking = value.thinking + (e.thinking ?? '');
      drafts[key] = {
        ...value,
        text: textTail(text, windowChars),
        thinking: textTail(thinking, windowChars),
        truncated: value.truncated || text.length > windowChars || thinking.length > windowChars,
      };
    } else if (!isResponseEvent(e)) {
      if (events === state.events) events = [...events];
      events.push(e);
    }
  }
  if (drafts !== state.drafts) {
    for (const [key, draft] of Object.entries(drafts).slice(0, -128))
      if (draft.doneRevision !== undefined) delete drafts[key];
    const all = Object.entries(drafts);
    if (all.reduce((n, [, d]) => n + d.text.length + d.thinking.length, 0) > totalChars) {
      const limit = Math.floor(totalChars / Math.max(1, all.length * 2));
      for (const [key, d] of all)
        drafts[key] = {
          ...d,
          text: textTail(d.text, limit),
          thinking: textTail(d.thinking, limit),
          truncated: true,
        };
    }
  }
  if (tasks !== state.tasks) trimTasks(tasks);
  if (events.length > 128) events = events.slice(-128);
  if (
    drafts === state.drafts &&
    tasks === state.tasks &&
    events === state.events &&
    historyRevision === state.historyRevision &&
    statistics === state.statistics
  )
    return state;
  return { drafts, tasks, events, historyRevision, statistics };
}
