// Purpose: Define workspace query identities and invalidate related server projections.
// Pattern: TanStack Query ownership and cancellation.
// Usage: Sidebar, conversation and settings share these keys.
// Related: webui/src/shared/api/client.ts, webui/src/pages/workspace/model/store.ts.

import { invalidate } from '@/pages/workspace/api/cache-refresh';
import { readSnapshot, snapshotMetadata } from '@/pages/workspace/api/snapshots';
import type { Target } from '@/pages/workspace/model/store';
import { call } from '@/shared/api';
import { failure } from '@/shared/lib/errors';

export const capabilitiesOptions = {
  queryKey: ['capabilities'],
  queryFn: async ({ signal }: { signal: AbortSignal }) => {
    const reply = await call('', { capabilities: {} }, signal);
    if (!reply.capabilities) throw failure('invalid_response');
    return reply.capabilities;
  },
};

export const workspaceOptions = () => ({
  queryKey: ['workspaces'],
  queryFn: async ({ signal }: { signal: AbortSignal }) =>
    (await call('', { listWorkspaces: {} }, signal)).workspaces!,
});
export const stateOptions = (target: Target) => ({
  queryKey: ['state', target.workspace, target.session],
  queryFn: async ({ signal }: { signal: AbortSignal }) =>
    snapshotMetadata(await readSnapshot(target, signal)),
});
export const defaultModelOptions = {
  queryKey: ['state', 'default-model'],
  queryFn: async ({ signal }: { signal: AbortSignal }) =>
    (await call('', { state: { defaultModel: true } }, signal)).state!,
};
export function changed(target: Target) {
  invalidate(['state', target.workspace, target.session]);
  invalidate(['sessions', target.workspace]);
  invalidate(['workspaces']);
}
