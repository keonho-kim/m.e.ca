// Purpose: Compose atomic workspace view, composer and live-stream slices.
// Pattern: Zustand slice composition root.
// Usage: Only workspace-page internals consume this store.
// Related: webui/src/pages/workspace/model/types.ts.

import { create } from 'zustand';
import { createComposerSlice } from '@/pages/workspace/model/composer-slice';
import { createStreamSlice } from '@/pages/workspace/model/stream-slice';
import type { WorkspaceState } from '@/pages/workspace/model/types';
import { createViewSlice } from '@/pages/workspace/model/view-slice';

export type {
  Input,
  Queued,
  Target,
  TaskView,
  WorkspaceState,
} from '@/pages/workspace/model/types';
export { emptyInput, targetKey } from '@/pages/workspace/model/types';
export const useWorkspaceStore = create<WorkspaceState>()((...args) => ({
  ...createViewSlice(...args),
  ...createComposerSlice(...args),
  ...createStreamSlice(...args),
}));
