// Purpose: Own workspace navigation, display controls and modal selection.
// Pattern: Zustand workspace view slice.
// Usage: Workspace presentation and commands change this state.
// Related: webui/src/pages/workspace/model/store.ts.

import type { StateCreator } from 'zustand';
import type { WorkspaceState } from '@/pages/workspace/model/types';

type ViewSlice = Pick<
  WorkspaceState,
  | 'selection'
  | 'mode'
  | 'folder'
  | 'picker'
  | 'tasksOpen'
  | 'paused'
  | 'streamStatus'
  | 'reviewTarget'
  | 'removedTarget'
  | 'settings'
  | 'fileDirty'
  | 'set'
>;
export const createViewSlice: StateCreator<WorkspaceState, [], [], ViewSlice> = (set) => ({
  selection: { workspace: '', session: '' },
  mode: 'chat',
  folder: '.',
  picker: false,
  tasksOpen: false,
  paused: false,
  streamStatus: 'connecting',
  reviewTarget: null,
  removedTarget: null,
  settings: null,
  fileDirty: false,
  set,
});
