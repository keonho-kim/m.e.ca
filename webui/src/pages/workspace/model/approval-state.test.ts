// Purpose: Keep actionable approvals independent of transcript paging and stale query metadata.
// Pattern: Ordered stream-state regression over the real store.
// Usage: Bun runs this with workspace model tests.
// Related: webui/src/pages/workspace/model/stream-slice.ts, webui/src/pages/workspace/ui/inspection/approval.tsx.
import { expect, test } from 'bun:test';
import { stateOptions } from '@/pages/workspace/api/queries';
import { useWorkspaceStore } from '@/pages/workspace/model/store';
import { queries } from '@/shared/api';

test('pending approval survives stale metadata, transcript churn and history acknowledgement', () => {
  const target = { workspace: 'approval-workspace', session: 'approval-session' };
  const previous = useWorkspaceStore.getState();
  try {
    previous.set({ selection: target, tasks: {}, drafts: {}, events: [], replayCursor: '' });
    previous.bootstrap(target, { approvals: [], tasks: [{ id: 'task', status: 'running' }] });
    previous.receive([
      {
        workspaceId: target.workspace,
        sessionId: target.session,
        taskId: 'task',
        kind: 'approval_required',
        approvalId: 'approve',
        approval: { id: 'approve', taskId: 'task', reasons: ['Outside workspace'] },
      },
    ]);
    queries.setQueryData(stateOptions(target).queryKey, { approvals: [] });
    previous.receive(
      Array.from({ length: 160 }, (_, index) => ({
        workspaceId: target.workspace,
        sessionId: target.session,
        taskId: 'task',
        kind: 'skill_loaded',
        sequence: String(index + 2),
        notice: { code: 'SKILL_READ' as const, skill: { name: 'fixture' } },
      }))
    );
    previous.acknowledgeHistory(target, 1000, { through: '1000', items: [] });
    expect(useWorkspaceStore.getState().approvals.map((approval) => approval.id)).toEqual([
      'approve',
    ]);
    previous.receive([
      {
        workspaceId: target.workspace,
        sessionId: target.session,
        taskId: 'task',
        kind: 'approval_closed',
        approvalId: 'approve',
      },
    ]);
    expect(useWorkspaceStore.getState().approvals).toEqual([]);
  } finally {
    useWorkspaceStore.setState(previous, true);
    queries.removeQueries({ queryKey: stateOptions(target).queryKey });
  }
});
