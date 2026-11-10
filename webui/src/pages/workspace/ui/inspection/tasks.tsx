// Purpose: Inspect project task receipts and active commands without loading conversation bodies.
// Pattern: Lazy Astryx drawer with library-owned project disclosure.
// Usage: Header opens Tasks; navigation selects the task's owning conversation.
// Related: webui/src/pages/workspace/ui/inspection/command-runs.tsx, internal/commands/server/task_page.go.

import { Badge } from '@astryxdesign/core/Badge';
import { Button } from '@astryxdesign/core/Button';
import { Collapsible } from '@astryxdesign/core/Collapsible';
import { Drawer, DrawerHeader } from '@astryxdesign/core/Drawer';
import { HStack, Layout, LayoutContent, VStack } from '@astryxdesign/core/Layout';
import { List, ListItem } from '@astryxdesign/core/List';
import { Spinner } from '@astryxdesign/core/Spinner';
import { Text } from '@astryxdesign/core/Text';
import { Tooltip } from '@astryxdesign/core/Tooltip';
import { useQuery } from '@tanstack/react-query';
import { useState } from 'react';
import { reportError } from '@/entities/server-session';
import { type Target, useWorkspaceStore } from '@/pages/workspace/model/store';
import { CommandRuns } from '@/pages/workspace/ui/inspection/command-runs';
import type { TaskJson, WorkspaceJson } from '@/shared/api';
import { call, control } from '@/shared/api';
import { useLocale } from '@/shared/i18n';

const statusLabels = {
  accepted: 'pending',
  running: 'running',
  canceling: 'canceling',
  answered: 'responseCompleted',
  stopped: 'stopped',
  interrupted: 'interrupted',
} as const;
function taskStatus(
  task: TaskJson
):
  | (typeof statusLabels)[keyof typeof statusLabels]
  | 'failed'
  | 'resultNeedsReview'
  | 'unknownTaskState' {
  if (task.persistence === 'uncertain') return 'resultNeedsReview';
  if (task.outcome === 'failed') return 'failed';
  const value = task.status ?? '';
  return Object.hasOwn(statusLabels, value)
    ? statusLabels[value as keyof typeof statusLabels]
    : 'unknownTaskState';
}
export default function Tasks({
  workspaces,
  navigate,
}: {
  workspaces: WorkspaceJson[];
  navigate: (target: Target) => void;
}) {
  const t = useLocale(),
    close = () => useWorkspaceStore.getState().set({ tasksOpen: false });
  return (
    <Drawer
      isOpen
      label={t('tasks')}
      onOpenChange={(open) => {
        if (!open) close();
      }}
      side="end"
      width={480}
      hasScrim={false}
    >
      <Layout
        height="fill"
        header={<DrawerHeader title={t('tasks')} onOpenChange={close} />}
        content={
          <LayoutContent padding={4}>
            <VStack gap={5}>
              {workspaces.map((workspace) => (
                <ProjectTasks key={workspace.id} workspace={workspace} navigate={navigate} />
              ))}
            </VStack>
          </LayoutContent>
        }
      />
    </Drawer>
  );
}
function ProjectTasks({
  workspace,
  navigate,
}: {
  workspace: WorkspaceJson;
  navigate: (target: Target) => void;
}) {
  const t = useLocale(),
    [expanded, setExpanded] = useState(!!workspace.activeTasks),
    [cursor, setCursor] = useState(''),
    [details, setDetails] = useState('');
  const live = useWorkspaceStore((state) => state.streamStatus === 'connected' && !state.paused);
  const page = useQuery({
    queryKey: ['task-list', workspace.id, cursor],
    enabled: expanded,
    queryFn: async ({ signal }) =>
      (await call(workspace.id ?? '', { listTasks: { cursor, limit: 32 } }, signal)).tasks!,
    refetchInterval:
      workspace.activeTasks && !live
        ? 2000 + ((workspace.id?.charCodeAt(10) ?? 0) % 7) * 53
        : false,
  });
  return (
    <Collapsible
      isOpen={expanded}
      onOpenChange={setExpanded}
      trigger={
        <HStack gap={2}>
          <Text type="label" weight="semibold">
            {workspace.name}
          </Text>
          {!!workspace.activeTasks && <Badge label={workspace.activeTasks} />}
        </HStack>
      }
    >
      <VStack gap={3}>
        {page.isPending && <Spinner label={t('loading')} />}
        {page.error && <Button label={t('retry')} onClick={() => page.refetch()} />}
        {!page.isPending && !page.data?.items?.length && (
          <Text color="secondary">{t('noTasks')}</Text>
        )}
        <List>
          {page.data?.items?.map((task) => (
            <ListItem
              key={task.id}
              label={
                <VStack gap={1}>
                  <Tooltip content={task.sessionSummary || t('untitled')}>
                    <Button
                      label={task.sessionSummary || t('untitled')}
                      className="max-w-full"
                      variant="ghost"
                      onClick={() => {
                        navigate({ workspace: workspace.id ?? '', session: task.sessionId ?? '' });
                        useWorkspaceStore.getState().set({ tasksOpen: false });
                      }}
                    />
                  </Tooltip>
                  <Text type="supporting" color="secondary">
                    {t(taskStatus(task))}
                  </Text>
                  <Text type="supporting" color="secondary" maxLines={1} hasTruncateTooltip>
                    {task.id}
                  </Text>
                  {details === task.id && (
                    <CommandRuns
                      workspace={workspace.id ?? ''}
                      task={task.id ?? ''}
                      running={task.status === 'running'}
                    />
                  )}
                </VStack>
              }
              endContent={
                <HStack gap={1}>
                  <Button
                    label={t('details')}
                    size="sm"
                    variant="ghost"
                    onClick={() => setDetails(details === task.id ? '' : (task.id ?? ''))}
                  />
                  {task.status === 'running' && (
                    <Button
                      label={t('stop')}
                      size="sm"
                      onClick={() => {
                        void control(workspace.id ?? '', task.id ?? '', 'cancel').catch(
                          reportError
                        );
                      }}
                    />
                  )}
                </HStack>
              }
            />
          ))}
        </List>
        {page.data?.nextCursor && (
          <Button label={t('loadMore')} onClick={() => setCursor(page.data.nextCursor ?? '')} />
        )}
      </VStack>
    </Collapsible>
  );
}
