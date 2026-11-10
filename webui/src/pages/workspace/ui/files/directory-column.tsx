// Purpose: Present workspace navigation, refresh and virtualized file reference selection.
// Pattern: Query-backed collection view.
// Usage: Files mounts one current folder and owns its navigation and refresh effects.
// Related: webui/src/pages/workspace/ui/files/files.tsx, webui/src/pages/workspace/model/file-references.ts.

import { Button } from '@astryxdesign/core/Button';
import { CheckboxInput } from '@astryxdesign/core/CheckboxInput';
import { Icon } from '@astryxdesign/core/Icon';
import { IconButton } from '@astryxdesign/core/IconButton';
import { HStack, Layout, LayoutContent, VStack } from '@astryxdesign/core/Layout';
import { Skeleton } from '@astryxdesign/core/Skeleton';
import { Text } from '@astryxdesign/core/Text';
import { Tooltip } from '@astryxdesign/core/Tooltip';
import { useInfiniteQuery } from '@tanstack/react-query';
import { useVirtualizer } from '@tanstack/react-virtual';
import { useMemo, useRef } from 'react';
import { reportError } from '@/entities/server-session';
import { selectReference } from '@/pages/workspace/model/file-references';
import { type Target, targetKey, useWorkspaceStore } from '@/pages/workspace/model/store';
import { call, queries } from '@/shared/api';
import { useLocale } from '@/shared/i18n';
export function DirectoryColumn({
  path,
  target,
  open,
  enabled,
  parent,
  refresh,
}: {
  path: string;
  target: Target;
  open: (path: string, directory: boolean) => void;
  enabled: boolean;
  parent?: () => void;
  refresh: () => void;
}) {
  const t = useLocale(),
    ref = useRef<HTMLDivElement>(null);
  const selected = useWorkspaceStore((s) => s.inputs[targetKey(target)]?.paths);
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
          {parent && (
            <HStack gap={2} className="min-w-0">
              <IconButton
                label={t('parent')}
                variant="ghost"
                icon={<Icon icon={'meca:back'} />}
                onClick={parent}
              />
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
                if (!entry) return null;
                const file = join(entry.name ?? ''),
                  reference = file + (entry.directory ? '/' : '');
                return (
                  <HStack
                    key={file}
                    className="directory-row"
                    style={{
                      position: 'absolute',
                      width: '100%',
                      transform: `translateY(${row.start}px)`,
                    }}
                    gap={2}
                  >
                    <CheckboxInput
                      label={`${t('references')}: ${entry.name}`}
                      isLabelHidden
                      value={selectedPaths.has(reference)}
                      isDisabled={!target.session}
                      onChange={(checked) => {
                        void selectReference(target, reference, checked).catch(reportError);
                      }}
                    />
                    <Tooltip content={entry.name}>
                      <Button
                        label={entry.name ?? ''}
                        variant="ghost"
                        icon={<Icon icon={entry.directory ? 'meca:folder' : 'meca:file'} />}
                        endContent={
                          entry.directory ? <Icon icon={'chevronRight'} size="xsm" /> : undefined
                        }
                        onClick={() => open(file, !!entry.directory)}
                      />
                    </Tooltip>
                  </HStack>
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
  );
}
