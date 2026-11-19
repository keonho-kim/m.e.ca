// Purpose: Route parsed commands and visible controls through the same application actions.
// Pattern: Explicit browser action dispatch.
// Usage: Composer, header and settings shortcuts call these functions.
// Related: internal/commands/browser.go, webui/src/pages/workspace/model/store.ts.
import { reportError } from '@/entities/server-session';
import { changed } from '@/pages/workspace/api/queries';
import {
  emptyInput,
  type Target,
  targetKey,
  useWorkspaceStore,
} from '@/pages/workspace/model/store';
import { startRequest } from '@/pages/workspace/model/submission';
import type { PrepareSession } from '@/pages/workspace/model/use-composer-session';
import { call } from '@/shared/api';
import { failure } from '@/shared/lib/errors';

export function settings(tab: 'model' | 'execution', target: Target) {
  useWorkspaceStore.getState().set({ settings: { tab, target } });
}
export function view(mode: 'chat' | 'files' | 'skills', path?: string) {
  const state = useWorkspaceStore.getState();
  if (state.fileDirty) {
    reportError(failure('draft_changed'));
    return false;
  }
  state.set({ mode, ...(mode === 'files' ? { folder: path ?? state.folder } : {}) });
  return true;
}
export async function team(target: Target, value: boolean) {
  if (!target.session) throw failure('no_session');
  await call(target.workspace, { setSettings: { sessionId: target.session, team: value } });
  changed(target);
}
export async function command(
  target: Target,
  text: string,
  prepareSession: PrepareSession
): Promise<Target | false> {
  const reply = await call(target.workspace, { parseInput: { text } });
  const parsed = reply.parsedInput;
  let destination = target;
  switch (parsed?.action) {
    case 'model':
      settings('model', target);
      break;
    case 'language':
      useWorkspaceStore.getState().set({ languageMenu: true });
      break;
    case 'permission':
      settings('execution', target);
      break;
    case 'dir':
      return view('files', parsed.argument || '.') ? target : false;
    case 'skills':
      return view('skills') ? target : false;
    case 'compact': {
      if (!target.session) throw failure('no_session');
      const state = useWorkspaceStore.getState();
      if (state.fileDirty) throw failure('draft_changed');
      const draft = state.inputs[targetKey(target)];
      const compact = { ...emptyInput(), text: '!compact', revision: draft?.revision ?? 0 };
      if (!(await startRequest(target, compact))) return false;
      if (targetKey(useWorkspaceStore.getState().selection) === targetKey(target)) view('chat');
      return target;
    }
    case 'team': {
      destination = await prepareSession(async (next) => {
        const state = await call(next.workspace, { state: { sessionId: next.session } });
        await team(next, !state.state?.team);
      });
      break;
    }
    default:
      return false;
  }
  return destination;
}
