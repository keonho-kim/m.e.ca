// Purpose: Restore selected references once per conversation navigation, independently of composer placement.
// Pattern: Conversation-owned cancellable read lifecycle.
// Usage: WorkspacePage owns this hook; file and chat tabs reuse its authoritative input state.
// Related: webui/src/pages/workspace/model/file-references.ts, webui/src/pages/workspace/model/reference-queue.ts.
import { useEffect } from 'react';
import { reportError } from '@/entities/server-session';
import { refreshReferences } from '@/pages/workspace/model/file-references';
import type { Target } from '@/pages/workspace/model/store';
export function useFileReferences(target: Target) {
  useEffect(() => {
    if (!target.session) return;
    const controller = new AbortController();
    void refreshReferences(target, controller.signal).catch(reportError);
    return () => controller.abort();
  }, [target]);
}
