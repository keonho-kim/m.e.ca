// Purpose: Describe workspace-scoped transient state and conversation identities.
// Pattern: Page model contracts.
// Usage: Workspace model slices and presentation use these types.
// Related: webui/src/pages/workspace/model/store.ts.

import type { EventJson, SnapshotJson, StatisticsJson } from '@/shared/api';
export type Target = { workspace: string; session: string };
export type Input = { text: string; paths: string[]; files: File[]; revision: number };
export type Queued = Input & { id: string };
export type Draft = {
  task: string;
  agent: string;
  generation: string;
  text: string;
  thinking: string;
  truncated: boolean;
  doneRevision?: number;
};
export type TaskView = { id: string; workspace: string; session: string; status: string };
export const targetKey = (t: Target) => `${t.workspace}/${t.session}`;
export const emptyInput = (): Input => ({ text: '', paths: [], files: [], revision: 0 });

export interface WorkspaceState {
  historyRevision: number;
  statistics: StatisticsJson | null;
  selection: Target;
  mode: 'chat' | 'files' | 'skills';
  folder: string;
  picker: boolean;
  tasksOpen: boolean;
  paused: boolean;
  streamStatus: string;
  reviewTarget: Target | null;
  removedTarget: Target | null;
  settings: null | { tab: 'model' | 'execution' | 'general'; target: Target };
  inputs: Record<string, Input>;
  queues: Record<string, Queued[]>;
  pausedQueues: Record<string, boolean>;
  drafts: Record<string, Draft>;
  tasks: Record<string, TaskView>;
  events: EventJson[];
  fileDirty: boolean;
  set: (value: Partial<WorkspaceState>) => void;
  patchInput: (target: Target, patch: Partial<Input>) => boolean;
  enqueue: (target: Target, input: Input) => boolean;
  removeQueue: (target: Target, id: string) => void;
  updateQueue: (target: Target, id: string, text: string) => boolean;
  bootstrap: (target: Target, state: SnapshotJson) => void;
  receive: (events: EventJson[]) => void;
}
