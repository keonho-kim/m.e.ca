// Purpose: Create a conversation on first use and transfer its workspace draft without copying payloads.
// Pattern: Composer-owned single-flight creation with stale-view rejection.
// Usage: Composer prepares a session before submitting work or changing team mode.
// Related: webui/src/pages/workspace/model/store.ts, webui/src/pages/workspace/api/queries.ts.
import { useEffect, useRef } from 'react';
import { useServerSession } from '@/entities/server-session';
import { changed } from '@/pages/workspace/api/queries';
import { type Target, targetKey, useWorkspaceStore } from '@/pages/workspace/model/store';
import { call } from '@/shared/api';
import { failure } from '@/shared/lib/errors';

export type PrepareSession = (initialize?: (next: Target) => Promise<void>) => Promise<Target>;

export function useComposerSession(target: Target, onCreated: (next: Target) => void) {
  const pending = useRef<Promise<Target> | null>(null);
  const alive = useRef(true);
  useEffect(() => {
    alive.current = true;
    return () => {
      alive.current = false;
    };
  }, []);
  const prepare: PrepareSession = async (initialize) => {
    const existing = target.session ? target : pending.current && (await pending.current);
    if (existing) {
      const next = existing;
      await initialize?.(next);
      return next;
    }
    if (!target.workspace) throw failure('no_session');
    const create = async () => {
      if (useWorkspaceStore.getState().fileDirty) throw failure('draft_changed');
      const reply = await call(target.workspace, { selectSession: { create: true } });
      const session = reply.state?.session?.id;
      if (!session) throw failure('invalid_response');
      const next = { workspace: target.workspace, session };
      changed(next);
      const current = () => {
        const state = useWorkspaceStore.getState();
        if (
          !alive.current ||
          !useServerSession.getState().connected ||
          targetKey(state.selection) !== targetKey(target)
        )
          throw new DOMException(failure('cancelled').message, 'AbortError');
        if (state.fileDirty) throw failure('draft_changed');
        return state;
      };
      current();
      try {
        // Apply initial settings before navigation starts snapshot subscriptions.
        await initialize?.(next);
      } finally {
        const state = current();
        // Transfer the latest draft atomically; never keep a second image/text owner.
        const inputs = { ...state.inputs };
        const input = inputs[targetKey(target)];
        delete inputs[targetKey(target)];
        if (input) inputs[targetKey(next)] = input;
        state.set({ inputs });
        onCreated(next);
      }
      return next;
    };
    pending.current = create();
    try {
      return await pending.current;
    } catch (error) {
      pending.current = null;
      throw error;
    }
  };
  return prepare;
}
