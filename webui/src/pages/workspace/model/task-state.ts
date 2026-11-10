// Purpose: Index immutable task sets and retain terminal history without evicting live work.
// Pattern: Weakly cached presentation projection.
// Usage: Composer, metrics and navigation select tasks without scanning on each token.
// Related: webui/src/pages/workspace/model/stream-projection.ts, webui/src/pages/workspace/model/types.ts.
import { type Target, type TaskView, targetKey } from '@/pages/workspace/model/types';

export function isActiveTask(status: string) {
  return status === 'running' || status === 'accepted' || status === 'canceling';
}
const indices = new WeakMap<
  Record<string, TaskView>,
  {
    targets: Map<string, TaskView>;
    workspaces: Map<string, TaskView[]>;
  }
>();
function index(tasks: Record<string, TaskView>) {
  let value = indices.get(tasks);
  if (!value) {
    value = { targets: new Map(), workspaces: new Map() };
    for (const task of Object.values(tasks)) {
      if (!isActiveTask(task.status)) continue;
      const key = targetKey(task);
      if (!value.targets.has(key)) value.targets.set(key, task);
      const list = value.workspaces.get(task.workspace) ?? [];
      list.push(task);
      value.workspaces.set(task.workspace, list);
    }
    indices.set(tasks, value);
  }
  return value;
}
const emptyTasks: TaskView[] = [];
export function activeTask(target: Target, tasks: Record<string, TaskView>) {
  return index(tasks).targets.get(targetKey(target));
}
export function workspaceTasks(workspace: string, tasks: Record<string, TaskView>) {
  return index(tasks).workspaces.get(workspace) ?? emptyTasks;
}
export function trimTasks(tasks: Record<string, TaskView>) {
  let remove = Object.keys(tasks).length - 1024;
  if (remove <= 0) return;
  for (const [id, task] of Object.entries(tasks)) {
    if (!isActiveTask(task.status)) {
      delete tasks[id];
      if (--remove === 0) break;
    }
  }
}
