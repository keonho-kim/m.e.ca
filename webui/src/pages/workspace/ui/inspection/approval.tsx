// Purpose: Select and resolve actionable approvals without downloading dialog and code-viewer code while idle.
// Pattern: Query-backed approval controller with an explicit lazy presentation boundary.
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
  const events = useWorkspaceStore((s) => s.events);
  const state = useQuery({
    ...stateOptions(target),
    enabled: !!target.session,
    refetchOnMount: 'always',
  });
  const [busy, setBusy] = useState(false),
    [resolved, setResolved] = useState<string[]>([]);
  const relevant = events.filter(
    (e) => e.workspaceId === target.workspace && e.sessionId === target.session
  );
  const closed = new Set([
    ...resolved,
    ...relevant.filter((e) => e.kind === 'approval_closed').map((e) => e.approvalId),
  ]);
  const approvals = [
    ...(state.data?.approvals ?? []),
    ...relevant.flatMap((e) => (e.approval ? [e.approval] : [])),
  ];
  const approval = approvals.find((a) => a.id && !closed.has(a.id));
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
