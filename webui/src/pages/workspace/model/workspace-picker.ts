// Purpose: Own one native folder-selection action and share workspace registration with the remote modal.
// Pattern: Explicit action lifecycle with unmount cancellation.
// Usage: Sidebar triggers native selection; FolderPicker uses the same registration contract.
// Related: webui/src/pages/workspace/api/folder-picker.ts, webui/src/pages/workspace/ui/navigation/folder-picker.tsx.
import { useEffect, useRef, useState } from 'react';
import { reportError } from '@/entities/server-session';
import { nativeFolder } from '@/pages/workspace/api/folder-picker';
import { type Target, useWorkspaceStore } from '@/pages/workspace/model/store';
import { call, queries } from '@/shared/api';
import { ApiError, failure } from '@/shared/lib/errors';

export async function registerWorkspace(path: string) {
  const reply = await call('', { addWorkspace: { path } });
  const id = reply.workspaces?.selectedId;
  if (!id || !reply.workspaces?.items?.some((workspace) => workspace.id === id))
    throw failure('invalid_response');
  queries.setQueryData(['workspaces'], reply.workspaces);
  return { workspace: id, session: '' };
}
export function useWorkspacePicker(navigate: (target: Target) => void) {
  const request = useRef<AbortController | null>(null),
    [busy, setBusy] = useState(false);
  useEffect(() => () => request.current?.abort(), []);
  const open = async () => {
    if (request.current) return;
    const controller = new AbortController();
    request.current = controller;
    setBusy(true);
    try {
      const result = await nativeFolder(window.location.hostname, controller.signal);
      if (result.kind === 'fallback') useWorkspaceStore.getState().set({ picker: true });
      else if (result.kind === 'selected') {
        // Registration is a write: finish its observed result even if selection is subsequently closed.
        const target = await registerWorkspace(result.path);
        if (!controller.signal.aborted) navigate(target);
      }
    } catch (error) {
      if (!controller.signal.aborted) {
        if (
          error instanceof ApiError &&
          ['folder_picker', 'folder_picker_timeout', 'folder_picker_busy'].includes(error.code)
        )
          useWorkspaceStore.getState().set({ picker: true });
        reportError(error);
      }
    } finally {
      if (request.current === controller) {
        request.current = null;
        setBusy(false);
      }
    }
  };
  return { busy, open };
}
