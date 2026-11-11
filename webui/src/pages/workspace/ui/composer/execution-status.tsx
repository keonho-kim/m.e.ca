// Purpose: Show a single line of conversation usage and current execution measurements.
// Pattern: Isolated status subscriber with terminal-event refresh of durable totals.
// Usage: The chat composer mounts this below the input.
// Related: internal/commands/server/session_usage.go, webui/src/pages/workspace/ui/chat/format-statistics.ts.
import { Text } from '@astryxdesign/core/Text';
import { useQuery } from '@tanstack/react-query';
import { useEffect, useState } from 'react';
import { stateOptions } from '@/pages/workspace/api/queries';
import { type Target, useWorkspaceStore } from '@/pages/workspace/model/store';
import { activeTask } from '@/pages/workspace/model/task-state';
import { duration, throughput, tokenSummary } from '@/pages/workspace/ui/chat/format-statistics';
import { call } from '@/shared/api';
import { useLocale } from '@/shared/i18n';

export function ExecutionStatus({ target }: { target: Target }) {
  const t = useLocale();
  const task = useWorkspaceStore((s) => activeTask(target, s.tasks));
  const latest = useWorkspaceStore((s) => s.statistics);
  const { data } = useQuery({ ...stateOptions(target), enabled: !!target.session });
  const usage = useQuery({
    queryKey: ['session-usage', target.workspace, target.session],
    enabled: !!target.session,
    staleTime: Infinity,
    queryFn: async ({ signal }) =>
      (await call(target.workspace, { sessionUsage: { sessionId: target.session } }, signal))
        .usage!,
  });
  const resources = useQuery({
    queryKey: ['resources'],
    enabled: !!target.workspace,
    staleTime: 30000,
    refetchInterval: task ? 30000 : false,
    queryFn: async ({ signal }) => (await call('', { resources: {} }, signal)).resources!,
  });
  const execution = data?.execution;
  const started =
    task && execution?.taskId === task.id ? Date.parse(execution.startedAt ?? '') : NaN;
  const [now, setNow] = useState(Date.now);
  useEffect(() => {
    if (!Number.isFinite(started)) return;
    setNow(Date.now());
    const timer = setInterval(() => {
      if (!document.hidden) setNow(Date.now());
    }, 1000);
    return () => clearInterval(timer);
  }, [started]);
  const stats = latest ?? execution?.statistics;
  const resident = Number(resources.data?.residentBytes ?? 0);
  const memory =
    resident >= 1073741824
      ? `${(resident / 1073741824).toFixed(2)} GiB`
      : `${(resident / 1048576).toFixed(1)} MiB`;
  const summary = [
    task
      ? `${t('elapsed')} ${duration(Number.isFinite(started) ? now - started : Number(stats?.elapsedMillis ?? 0))}`
      : '',
    resident > 0 ? `${t('memory')} ${memory}` : '',
    ...tokenSummary(usage.data, t, data?.model?.provider === 'openrouter'),
    throughput(stats),
  ]
    .filter(Boolean)
    .join(' · ');
  return (
    <Text
      as="p"
      type="supporting"
      color="secondary"
      role="group"
      aria-label={t('executionInfo')}
      className="execution-status min-w-0 whitespace-normal [overflow-wrap:anywhere]"
    >
      {summary}
    </Text>
  );
}
