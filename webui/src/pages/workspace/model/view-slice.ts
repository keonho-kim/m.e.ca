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
  streamStatus: 'connecting',
  reviewTarget: null,
  removedTarget: null,
  settings: null,
  fileDirty: false,
  set: (patch) =>
    set((state) => {
      for (const key of Object.keys(patch) as (keyof WorkspaceState)[])
        if (!Object.is(state[key], patch[key])) return patch;
      return state;
    }),
});
