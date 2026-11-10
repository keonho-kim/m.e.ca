// Purpose: Present selected execution timing and usage alongside periodically sampled server memory.
// Pattern: Independent status subscriber with bounded background polling.
// Usage: ConnectionStatus mounts this only while execution details are open.
// Related: webui/src/pages/workspace/model/stream-projection.ts, internal/serverapi/workspace.go.

import { VStack } from '@astryxdesign/core/Layout';
import { Text } from '@astryxdesign/core/Text';
import { useQuery } from '@tanstack/react-query';
import { useEffect, useState } from 'react';
import { stateOptions } from '@/pages/workspace/api/queries';
import { type Target, useWorkspaceStore } from '@/pages/workspace/model/store';
import { activeTask } from '@/pages/workspace/model/task-state';
import { call } from '@/shared/api';
import { useLocale } from '@/shared/i18n';
export function Metrics({ target }: { target: Target }) {
  const t = useLocale(),
    latest = useWorkspaceStore((s) => s.statistics);
  const state = useQuery({ ...stateOptions(target), enabled: !!target.session });
  const resources = useQuery({
    queryKey: ['resources'],
    queryFn: async ({ signal }) => (await call('', { resources: {} }, signal)).resources!,
    refetchInterval: 10000,
  });
  const execution =
    state.data?.executions?.find((e) => e.session?.id === target.session) ?? state.data?.execution;
  const stats = latest ?? execution?.statistics;
  const task = useWorkspaceStore((s) => activeTask(target, s.tasks));
  const [now, setNow] = useState(Date.now());
  useEffect(() => {
    if (!task?.id) return;
    const timer = setInterval(() => setNow(Date.now()), 1000);
    return () => clearInterval(timer);
  }, [task?.id]);
  const elapsed =
    task && execution?.startedAt
      ? Math.max(0, now - Date.parse(execution.startedAt))
      : Number(stats?.elapsedMillis ?? 0);
  const tokens = Number(stats?.usage?.output ?? 0),
    generation = Number(stats?.generationMillis ?? 0);
  return (
    <VStack gap={2} className="execution-metrics">
      {elapsed > 0 && (
        <Text>
          {t('elapsed')} {(elapsed / 1000).toFixed(0)}s
        </Text>
      )}
      {!!stats?.usage && (
        <Text>
          {t('inputTokens')} {stats.usage.input ?? 0} · {t('outputTokens')} {tokens}
          {generation > 0 ? ` · ${((tokens * 1000) / generation).toFixed(1)} tok/s` : ''}
        </Text>
      )}
      {stats?.usage?.cost !== undefined && (
        <Text>
          {t('cost')} ${Number(stats.usage.cost).toFixed(4)}
        </Text>
      )}
      {resources.data?.residentBytes && (
        <Text>
          {t('memory')} {(Number(resources.data.residentBytes) / 1048576).toFixed(0)} MiB
        </Text>
      )}
    </VStack>
  );
}
