// Purpose: Inspect actual subprocess status only when the user expands a task's command details.
// Pattern: On-demand native status subscriber.
// Usage: Tasks renders this for an explicitly expanded task row.
// Related: webui/src/pages/workspace/ui/inspection/tasks.tsx, internal/serverapi/workspace.go.

import { Button } from '@astryxdesign/core/Button';
import { Code } from '@astryxdesign/core/Code';
import { VStack } from '@astryxdesign/core/Layout';
import { Text } from '@astryxdesign/core/Text';
import { useQuery } from '@tanstack/react-query';
import { call } from '@/shared/api';
import { useLocale } from '@/shared/i18n';
export function CommandRuns({
  workspace,
  task,
  running,
}: {
  workspace: string;
  task: string;
  running: boolean;
}) {
  const t = useLocale();
  const query = useQuery({
    queryKey: ['commands', workspace, task],
    queryFn: async ({ signal }) =>
      (await call(workspace, { commands: { taskId: task } }, signal)).commands!,
    refetchInterval: running ? 2000 : false,
    gcTime: 0,
  });
  return (
    <VStack className="command-runs" gap={4}>
      {query.isPending && <Text as="p">{t('loading')}</Text>}
      {query.error && (
        <Button
          type="button"
          className="text-button"
          onClick={() => query.refetch()}
          label={String(t('readError')) + ' ' + '\u00B7' + ' ' + String(t('retry'))}
          variant="ghost"
        >
          {t('readError')} · {t('retry')}
        </Button>
      )}
      {query.data?.items?.map((command) => (
        <VStack key={`${command.agent}/${command.id}`} gap={4}>
          <Code>{command.command}</Code>
          <Text>
            {command.agent} · {command.status} ·{' '}
            {(Number(command.elapsedMillis ?? 0) / 1000).toFixed(1)}s
            {command.exitCode !== undefined ? ` · exit ${command.exitCode}` : ''}
          </Text>
        </VStack>
      ))}
    </VStack>
  );
}
