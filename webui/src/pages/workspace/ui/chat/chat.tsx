// Purpose: Render paged conversation history and live responses without replacing one with a loading flash.
// Pattern: Virtualized history with independently subscribed live output.
// Usage: App mounts Chat for the selected conversation.
// Related: webui/src/pages/workspace/ui/chat/history-row.tsx, webui/src/pages/workspace/ui/chat/live-output.tsx, webui/src/pages/workspace/model/stream-projection.ts.

import { Banner } from '@astryxdesign/core/Banner';
import { Button } from '@astryxdesign/core/Button';
import { Center } from '@astryxdesign/core/Center';
import { ChatMessageList } from '@astryxdesign/core/Chat';
import { Heading } from '@astryxdesign/core/Heading';
import { VStack } from '@astryxdesign/core/Layout';
import { Skeleton } from '@astryxdesign/core/Skeleton';
import { Text } from '@astryxdesign/core/Text';
import { useInfiniteQuery, useQuery } from '@tanstack/react-query';
import { useVirtualizer } from '@tanstack/react-virtual';
import { useCallback, useEffect, useLayoutEffect, useMemo, useRef, useState } from 'react';
import { stateOptions } from '@/pages/workspace/api/queries';
import { type Target, targetKey, useWorkspaceStore } from '@/pages/workspace/model/store';
import { HistoryRow } from '@/pages/workspace/ui/chat/history-row';
import { LiveOutput } from '@/pages/workspace/ui/chat/live-output';
import { call, queries } from '@/shared/api';
import { useLocale } from '@/shared/i18n';

const positions = new Map<string, number>();
export function Chat({
  target,
  scroller,
  onEmpty,
}: {
  target: Target;
  scroller: HTMLDivElement | null;
  onEmpty: (empty: boolean) => void;
}) {
  const t = useLocale(),
    previousSize = useRef<number | null>(null),
    latestRequested = useRef(false);
  const hasLive = useWorkspaceStore(
    (s) =>
      Object.values(s.drafts).some((d) => d.text || d.thinking) ||
      Object.values(s.tasks).some(
        (task) =>
          task.workspace === target.workspace &&
          task.session === target.session &&
          task.status === 'running'
      )
  );
  const state = useQuery(stateOptions(target));
  const [before, setBefore] = useState('0');
  const history = useInfiniteQuery({
    queryKey: ['history', target.workspace, target.session, before],
    initialPageParam: before,
    queryFn: async ({ pageParam, signal }) => {
      const clientRevision = useWorkspaceStore.getState().historyRevision;
      const reply = await call(
        target.workspace,
        { history: { sessionId: target.session, before: pageParam, limit: 32 } },
        signal
      );
      return { ...reply.history!, clientRevision };
    },
    getNextPageParam: (last) => (last.before && BigInt(last.before) > 1n ? last.before : undefined),
  });
  const items = useMemo(
    () =>
      history.data?.pages
        .flatMap((p) => p.items ?? [])
        .filter((item) => item.event?.kind !== 'run_checkpoint' || !!item.outcome)
        .sort((a, b) => (BigInt(a.sequence ?? '0') < BigInt(b.sequence ?? '0') ? -1 : 1)) ?? [],
    [history.data]
  );
  const getItemKey = useCallback((index: number) => items[index]?.sequence ?? index, [items]);
  const virtual = useVirtualizer({
    count: items.length,
    getScrollElement: () => scroller,
    estimateSize: () => 140,
    overscan: 5,
    getItemKey,
  });
  useEffect(() => {
    if (!history.data || before !== '0') return;
    const seen = history.data.pages[0]?.clientRevision;
    if (seen === undefined) return;
    const state = useWorkspaceStore.getState(),
      drafts = { ...state.drafts };
    if (Object.values(drafts).some((d) => d.doneRevision !== undefined && d.doneRevision > seen))
      void history.refetch();
    const finished = new Set<string>();
    let changed = false;
    for (const [key, draft] of Object.entries(drafts)) {
      if (draft.doneRevision !== undefined && draft.doneRevision <= seen) {
        delete drafts[key];
        finished.add(draft.task);
        changed = true;
      }
    }
    if (changed)
      state.set({ drafts, events: state.events.filter((e) => !finished.has(e.taskId ?? '')) });
  }, [history.data, before, history.refetch]);
  useLayoutEffect(() => {
    const element = scroller;
    if (!element) return;
    const saved = positions.get(targetKey(target));
    if (saved !== undefined) {
      element.scrollTop = saved;
    }
    return () => {
      positions.set(targetKey(target), element.scrollTop);
      if (positions.size > 12) positions.delete(positions.keys().next().value!);
    };
  }, [target, scroller]);
  useLayoutEffect(() => {
    if (items.length > 0 && previousSize.current !== null && scroller) {
      scroller.scrollTop += virtual.getTotalSize() - previousSize.current;
      previousSize.current = null;
    }
  }, [items.length, virtual, scroller]);
  const empty = history.isSuccess && items.length === 0 && !hasLive;
  useLayoutEffect(() => onEmpty(empty), [empty, onEmpty]);
  useLayoutEffect(() => {
    if (latestRequested.current && before === '0' && !history.isFetching && scroller) {
      scroller.scrollTop = scroller.scrollHeight;
      latestRequested.current = false;
    }
  }, [before, history.isFetching, scroller]);
  return (
    <Center width="100%">
      <VStack width="100%" maxWidth={760} paddingInline={6} paddingBlock={4} gap={5}>
        {empty && (
          <VStack gap={3} paddingBlock={6}>
            <Heading level={1} type="display-2">
              {t('emptyTitle')}
            </Heading>
            <Text color="secondary">{t('emptyHelp')}</Text>
          </VStack>
        )}
        {before !== '0' && (
          <Button
            size="sm"
            label={t('latest')}
            onClick={() => {
              previousSize.current = null;
              latestRequested.current = true;
              setBefore('0');
              void queries.resetQueries({
                queryKey: ['history', target.workspace, target.session, '0'],
                exact: true,
              });
            }}
          />
        )}
        {history.hasNextPage && (
          <Button
            label={t('loadMore')}
            isLoading={history.isFetchingNextPage}
            onClick={() => {
              if ((history.data?.pages.length ?? 0) >= 4) {
                previousSize.current = null;
                setBefore(history.data?.pages.at(-1)?.before ?? before);
              } else {
                previousSize.current = virtual.getTotalSize();
                void history.fetchNextPage();
              }
            }}
          />
        )}
        {history.isPending && (
          <VStack className="message-skeleton" gap={3} aria-label={t('loading')}>
            <Skeleton height={14} />
            <Skeleton height={14} width="80%" index={1} />
          </VStack>
        )}
        {history.error && (
          <Button label={`${t('readError')} · ${t('retry')}`} onClick={() => history.refetch()} />
        )}
        <ChatMessageList align="top" gap={0}>
          <VStack
            className="history-rows"
            style={{ height: virtual.getTotalSize(), position: 'relative' }}
          >
            {virtual.getVirtualItems().map((row) => {
              const item = items[row.index];
              if (!item) return null;
              return (
                <VStack
                  key={row.key}
                  data-index={row.index}
                  ref={virtual.measureElement}
                  style={{
                    position: 'absolute',
                    top: 0,
                    left: 0,
                    width: '100%',
                    transform: `translateY(${row.start}px)`,
                  }}
                >
                  <HistoryRow item={item} target={target} />
                </VStack>
              );
            })}
          </VStack>
          {/* Astryx must observe live content growth as part of the message list. */}
          {before === '0' && (
            <VStack paddingBlockStart={5}>
              <LiveOutput target={target} />
            </VStack>
          )}
        </ChatMessageList>
        {state.data?.task?.error && <Banner status="error" title={state.data.task.error} />}
      </VStack>
    </Center>
  );
}
