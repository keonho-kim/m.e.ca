// Purpose: Define workspace query identities and invalidate related server projections.
// Pattern: TanStack Query ownership and cancellation.
// Usage: Sidebar, conversation and settings share these keys.
// Related: webui/src/shared/api/client.ts, webui/src/pages/workspace/model/store.ts.

import { invalidate } from '@/pages/workspace/api/cache-refresh';
import type { Target } from '@/pages/workspace/model/store';
import { call } from '@/shared/api';

export const workspaceOptions = () => ({
  queryKey: ['workspaces'],
  queryFn: async ({ signal }: { signal: AbortSignal }) =>
    (await call('', { listWorkspaces: {} }, signal)).workspaces!,
});
export const stateOptions = (target: Target) => ({
  queryKey: ['state', target.workspace, target.session],
  queryFn: async ({ signal }: { signal: AbortSignal }) =>
    (await call(target.workspace, { state: { sessionId: target.session } }, signal)).state!,
});
export function changed(target: Target) {
  invalidate(['state', target.workspace, target.session]);
  invalidate(['sessions', target.workspace]);
  invalidate(['workspaces']);
}
