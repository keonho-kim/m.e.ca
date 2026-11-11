// Purpose: Present a virtualized directory and route visible entry actions to the workspace.
// Pattern: Query-backed collection view.
// Usage: Files mounts one current folder and owns its navigation and refresh effects.
// Related: webui/src/pages/workspace/ui/files/files.tsx, webui/src/pages/workspace/model/file-references.ts.

import { Button } from '@astryxdesign/core/Button';
import { Icon } from '@astryxdesign/core/Icon';
import { IconButton } from '@astryxdesign/core/IconButton';
import { HStack, Layout, LayoutContent, VStack } from '@astryxdesign/core/Layout';
import { Skeleton } from '@astryxdesign/core/Skeleton';
import { Text } from '@astryxdesign/core/Text';
import { PointerActivationConstraints } from '@dnd-kit/dom';
import { DragDropProvider, KeyboardSensor, PointerSensor } from '@dnd-kit/react';
import { useInfiniteQuery } from '@tanstack/react-query';
import { useVirtualizer } from '@tanstack/react-virtual';
import { useMemo, useRef, useState } from 'react';
import { reportError } from '@/entities/server-session';
import { selectReference } from '@/pages/workspace/model/file-references';
import { type Target, targetKey, useWorkspaceStore } from '@/pages/workspace/model/store';
import { EntryDialog } from '@/pages/workspace/ui/files/entry-dialog';
import {
  type EntryAction,
  EntryRow,
  ParentFolderButton,
} from '@/pages/workspace/ui/files/entry-row';
import { call, type FileMutationJson, queries } from '@/shared/api';
import { useLocale } from '@/shared/i18n';

const entrySensors = [
  PointerSensor.configure({
    activationConstraints: (event) =>
      event.pointerType === 'touch'
        ? [new PointerActivationConstraints.Delay({ value: 250, tolerance: 5 })]
        : [new PointerActivationConstraints.Distance({ value: 6 })],
  }),
  KeyboardSensor.configure({
    preventActivation: (event) => !event.altKey || event.key !== ' ',
  }),
];
export function DirectoryColumn({
  path,
  target,
  open,
  enabled,
  parent,
  parentPath,
  refresh,
  busy,
  mutate,
}: {
  path: string;
  target: Target;
  open: (path: string, directory: boolean) => void;
  enabled: boolean;
  parent?: () => void;
  parentPath?: string;
  refresh: () => void;
  busy: boolean;
  mutate: (change: FileMutationJson) => void;
}) {
  const t = useLocale(),
    ref = useRef<HTMLDivElement>(null);
  const selected = useWorkspaceStore((s) => s.inputs[targetKey(target)]?.paths);
  const [action, setAction] = useState<EntryAction | null>(null);
  const dragged = useRef<{ path: string; revision: string } | null>(null);
  const page = useInfiniteQuery({
    queryKey: ['directory', target.workspace, path],
    initialPageParam: { offset: 0, revision: '' },
    enabled,
    maxPages: 4,
    gcTime: 15000,
    queryFn: async ({ pageParam, signal }) =>
      (await call(target.workspace, { directory: { path, ...pageParam, limit: 200 } }, signal))
        .directory!,
    getNextPageParam: (value) =>
      value.more ? { offset: value.nextOffset ?? 0, revision: value.revision ?? '' } : undefined,
    getPreviousPageParam: (first, _pages, param) =>
      param.offset > 0
        ? { offset: Math.max(0, param.offset - 200), revision: first.revision ?? '' }
        : undefined,
  });
  const entries = useMemo(() => page.data?.pages.flatMap((v) => v.items ?? []) ?? [], [page.data]);
  const selectedPaths = useMemo(() => new Set(selected), [selected]);
  const virtual = useVirtualizer({
    count: entries.length,
    getScrollElement: () => ref.current,
    estimateSize: () => 36,
    overscan: 8,
  });
  const base = page.data?.pages[0]?.path?.replace(/\/$/, '') || path.replace(/\/$/, '');
  const join = (name: string) => `${base}/${name}`.replace(/^\.\//, '');
  return (
    <DragDropProvider
      sensors={entrySensors}
      onDragStart={({ operation }) => {
        const source = operation.source?.data;
        dragged.current =
          typeof source?.path === 'string' && typeof source.revision === 'string'
            ? { path: source.path, revision: source.revision }
            : null;
      }}
      onDragEnd={({ operation, canceled }) => {
        const source = dragged.current;
        dragged.current = null;
        const folder = operation.target?.data.folder;
        if (canceled || busy || !source || typeof folder !== 'string' || folder === source.path)
          return;
        mutate({ path: source.path, revision: source.revision, folder });
      }}
    >
      <Layout
        className="directory-column"
        height="fill"
        padding={0}
        header={
          <VStack padding={3} gap={2} className="min-w-0">
            <HStack gap={2} align="center" className="min-w-0">
              <Text type="label" className="grow">
                {t('fileWorkspace')}
              </Text>
              <IconButton
                label={t('refresh')}
                variant="ghost"
                icon={<Icon icon={'meca:refresh'} />}
                onClick={refresh}
              />
            </HStack>
            {parent && parentPath && (
              <HStack gap={2} className="min-w-0">
                <ParentFolderButton path={parentPath} open={parent} busy={busy} />
                <Text type="supporting" maxLines={1} hasTruncateTooltip className="min-w-0">
                  {path}
                </Text>
              </HStack>
            )}
          </VStack>
        }
        content={
          <LayoutContent padding={0} ref={ref}>
            {page.error && (
              <Button
                type="button"
                className="inline-error"
                onClick={() =>
                  queries.resetQueries({
                    queryKey: ['directory', target.workspace, path],
                    exact: true,
                  })
                }
                label={String(t('readError')) + ' ' + '\u00B7' + ' ' + String(t('retry'))}
                variant="ghost"
              >
                {t('readError')} · {t('retry')}
              </Button>
            )}
            <VStack className="directory-scroll" gap={0}>
              {page.hasPreviousPage && (
                <Button
                  label={t('pagePrevious')}
                  onClick={() => page.fetchPreviousPage()}
                  isDisabled={page.isFetchingPreviousPage}
                />
              )}
              {page.isPending && (
                <VStack className="list-skeleton" role="status" aria-label={t('loading')} gap={4}>
                  <Skeleton height={14} />
                  <Skeleton height={14} />
                  <Skeleton height={14} />
                </VStack>
              )}
              {!page.isPending && !entries.length && (
                <Text className="empty-small" as="p">
                  {t('noFile')}
                </Text>
              )}
              <VStack style={{ height: virtual.getTotalSize(), position: 'relative' }} gap={4}>
                {virtual.getVirtualItems().map((row) => {
                  const entry = entries[row.index];
                  if (!entry?.name) return null;
                  const file = join(entry.name);
                  return (
                    <EntryRow
                      key={file}
                      entry={{
                        path: file,
                        name: entry.name,
                        directory: !!entry.directory,
                        revision: page.data?.pages[0]?.revision ?? '',
                      }}
                      top={row.start}
                      busy={busy}
                      referenced={selectedPaths.has(file + (entry.directory ? '/' : ''))}
                      canReference={!!target.session}
                      open={open}
                      reference={(path, enabled) => {
                        void selectReference(target, path, enabled).catch(reportError);
                      }}
                      selectAction={setAction}
                    />
                  );
                })}
              </VStack>
              {page.hasNextPage && (
                <Button
                  type="button"
                  className="text-button"
                  onClick={() => page.fetchNextPage()}
                  label={String(t('loadMore'))}
                  variant="ghost"
                  isDisabled={page.isFetchingNextPage}
                >
                  {t('loadMore')}
                </Button>
              )}
            </VStack>
          </LayoutContent>
        }
      />
      {action && <EntryDialog action={action} close={() => setAction(null)} mutate={mutate} />}
    </DragDropProvider>
  );
}
