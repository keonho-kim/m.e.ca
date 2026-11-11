// Purpose: Own one expanded project's bounded conversation pages and loading controls.
// Pattern: Mount-scoped query controller.
// Usage: WorkspaceGroup mounts this only while its project is expanded.
// Related: webui/src/pages/workspace/ui/navigation/session-list.tsx, webui/src/pages/workspace/api/queries.ts.
import { Button } from '@astryxdesign/core/Button';
import { useEntryAnimation } from '@astryxdesign/core/hooks';
import { VStack } from '@astryxdesign/core/Layout';
import { Skeleton } from '@astryxdesign/core/Skeleton';
import { useInfiniteQuery } from '@tanstack/react-query';
import { useMemo } from 'react';
import { type Target, useWorkspaceStore } from '@/pages/workspace/model/store';
import { workspaceTasks } from '@/pages/workspace/model/task-state';
import { SessionList } from '@/pages/workspace/ui/navigation/session-list';
import { call, queries, type SessionJson } from '@/shared/api';
import { useLocale } from '@/shared/i18n';

export function WorkspaceSessions({
  workspace,
  target,
  navigate,
  remove,
}: {
  workspace: string;
  target: Target;
  navigate: (target: Target) => void;
  remove: (session: SessionJson) => void;
}) {
  const entry = useEntryAnimation('slideDown');
  const t = useLocale(),
    tasks = useWorkspaceStore((state) => workspaceTasks(workspace, state.tasks));
  const sessions = useInfiniteQuery({
    queryKey: ['sessions', workspace],
    initialPageParam: '',
    maxPages: 4,
    gcTime: 15000,
    queryFn: async ({ pageParam, signal }) =>
      (await call(workspace, { listSessions: { cursor: pageParam, limit: 50 } }, signal)).sessions!,
    getNextPageParam: (page) => page.nextCursor || undefined,
  });
  const items = useMemo(
    () => sessions.data?.pages.flatMap((page) => page.items ?? []) ?? [],
    [sessions.data]
  );
  const running = useMemo(() => new Set(tasks.map((task) => task.session)), [tasks]);
  return (
    <VStack gap={1} xstyle={entry}>
      {sessions.isPending && (
        <VStack padding={3} gap={3}>
          <Skeleton height={14} />
          <Skeleton height={14} width="80%" index={1} />
        </VStack>
      )}
      {sessions.error && (
        <Button
          label={t('retry')}
          onClick={() => queries.resetQueries({ queryKey: ['sessions', workspace], exact: true })}
        />
      )}
      {!!sessions.data?.pageParams[0] && (
        <Button
          label={t('listStart')}
          size="sm"
          onClick={() => queries.resetQueries({ queryKey: ['sessions', workspace], exact: true })}
        />
      )}
      <SessionList
        items={items}
        running={running}
        workspace={workspace}
        target={target}
        navigate={navigate}
        remove={remove}
      />
      {sessions.hasNextPage && (
        <Button
          label={t('loadMore')}
          size="sm"
          isLoading={sessions.isFetchingNextPage}
          onClick={() => sessions.fetchNextPage()}
        />
      )}
    </VStack>
  );
}
