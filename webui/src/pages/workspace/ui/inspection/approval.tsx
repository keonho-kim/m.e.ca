// Purpose: Select and resolve actionable approvals without downloading dialog and code-viewer code while idle.
// Pattern: Ordered stream-owned approval state with a query fallback for another conversation.
// Usage: WorkspaceShell keeps this controller mounted for the selected approval target.
// Related: webui/src/pages/workspace/ui/inspection/approval-dialog.tsx, internal/commands/server/controls.go.
import { useQuery } from '@tanstack/react-query';
import { lazy, Suspense, useEffect, useState } from 'react';
import { reportError } from '@/entities/server-session';
import { changed, stateOptions } from '@/pages/workspace/api/queries';
import { type Target, useWorkspaceStore } from '@/pages/workspace/model/store';
import { control } from '@/shared/api';
import { ApiError } from '@/shared/lib/errors';

const ApprovalDialog = lazy(() => import('@/pages/workspace/ui/inspection/approval-dialog'));
export function Approval({ target, onResolved }: { target: Target; onResolved: () => void }) {
  const live = useWorkspaceStore((s) => s.approvals);
  const selected = useWorkspaceStore((s) => s.selection);
  const state = useQuery({
    ...stateOptions(target),
    enabled: !!target.session,
    refetchOnMount: 'always',
  });
  const [busy, setBusy] = useState(false),
    [resolved, setResolved] = useState<string[]>([]);
  const current = selected.workspace === target.workspace && selected.session === target.session;
  const approvals = current ? live : (state.data?.approvals ?? []);
  const approval = approvals.find((a) => a.id && !resolved.includes(a.id));
  useEffect(() => {
    if (!approval && !state.isFetching && state.isSuccess) onResolved();
  }, [approval, state.isFetching, state.isSuccess, onResolved]);
  const id = approval?.id,
    taskId = approval?.taskId;
  if (!id || !taskId) return null;
  const decide = async (allow: boolean) => {
    setBusy(true);
    try {
      await control(target.workspace, taskId, 'approve', allow, id);
      setResolved((values) => [...values.slice(-127), id]);
      changed(target);
      onResolved();
    } catch (error) {
      if (
        error instanceof ApiError &&
        ['not_found', 'conflict', 'failed_precondition'].includes(error.code)
      ) {
        setResolved((values) => [...values.slice(-127), id]);
        onResolved();
      }
      reportError(error);
      changed(target);
    } finally {
      setBusy(false);
    }
  };
  return (
    <Suspense fallback={null}>
      <ApprovalDialog
        key={id}
        target={target}
        approval={approval}
        displayRoot={state.data?.session?.displayRoot}
        busy={busy}
        decide={decide}
      />
    </Suspense>
  );
}
