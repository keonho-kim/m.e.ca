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
import { useCallback, useLayoutEffect, useMemo, useRef, useState } from 'react';
import { stateOptions } from '@/pages/workspace/api/queries';
import { historyRows } from '@/pages/workspace/model/history-rows';
import { type Target, targetKey, useWorkspaceStore } from '@/pages/workspace/model/store';
import { activeTask } from '@/pages/workspace/model/task-state';
import { HistoryRow } from '@/pages/workspace/ui/chat/history-row';
import { chatWidth } from '@/pages/workspace/ui/chat/layout';
import { LiveOutput } from '@/pages/workspace/ui/chat/live-output';
import type { HistoryItemJson } from '@/shared/api';
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
  const taskId = useWorkspaceStore((s) => activeTask(target, s.tasks)?.id);
  const hasOutput = useWorkspaceStore((s) =>
    Object.values(s.drafts).some((draft) => draft.text || draft.thinking)
  );
  const followedTask = useRef<string | undefined>(undefined);
  const state = useQuery(stateOptions(target));
  const [before, setBefore] = useState('0');
  const history = useInfiniteQuery({
    queryKey: ['history', target.workspace, target.session, before],
    initialPageParam: before,
    staleTime: before === '0' ? undefined : Infinity,
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
  const items = useMemo(() => {
    const result: HistoryItemJson[] = [];
    const pages = history.data?.pages ?? [];
    // The server orders records within a page. Only the page order is reversed.
    for (let index = pages.length - 1; index >= 0; index--)
      for (const item of pages[index]?.items ?? [])
        if (item.event?.kind !== 'run_checkpoint' || item.outcome) result.push(item);
    return result;
  }, [history.data]);
  const rows = useMemo(() => historyRows(items), [items]);
  const last = rows.at(-1);
  const completion = !taskId && last?.item.outcome ? last : undefined;
  const visibleRows = completion ? rows.slice(0, -1) : rows;
  const getItemKey = useCallback(
    (index: number) => visibleRows[index]?.item.sequence ?? index,
    [visibleRows]
  );
  const virtual = useVirtualizer({
    count: visibleRows.length,
    getScrollElement: () => scroller,
    estimateSize: () => 140,
    overscan: 5,
    getItemKey,
  });
  useLayoutEffect(() => {
    if (!history.data || before !== '0') return;
    const seen = history.data.pages[0]?.clientRevision;
    if (seen === undefined) return;
    useWorkspaceStore.getState().acknowledgeHistory(target, seen);
  }, [history.data, before, target]);
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
  const empty = history.isSuccess && items.length === 0 && !taskId && !hasOutput;
  useLayoutEffect(() => onEmpty(empty), [empty, onEmpty]);
  useLayoutEffect(() => {
    if (!taskId || !scroller || before !== '0' || followedTask.current === taskId) return;
    followedTask.current = taskId;
    // A new turn returns to the newest content even after the prior turn was reviewed.
    scroller.scrollTop = scroller.scrollHeight;
  }, [taskId, scroller, before]);
  useLayoutEffect(() => {
    if (latestRequested.current && before === '0' && !history.isFetching && scroller) {
      scroller.scrollTop = scroller.scrollHeight;
      latestRequested.current = false;
    }
  }, [before, history.isFetching, scroller]);
  return (
    <Center width="100%" className="min-w-0">
      <VStack width={chatWidth} paddingBlock={4} gap={5} className="chat-content min-w-0">
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
        <ChatMessageList
          align="top"
          gap={0}
          style={{ width: '100%', minWidth: 0, maxWidth: '100%' }}
        >
          <VStack
            width="100%"
            className="history-rows"
            style={{ height: virtual.getTotalSize(), position: 'relative' }}
          >
            {virtual.getVirtualItems().map((row) => {
              const value = visibleRows[row.index];
              if (!value) return null;
              const item = value.item;
              return (
                <VStack
                  key={row.key}
                  className={`history-row ${item.role === 'user' ? 'user-message' : 'assistant-message'}`}
                  paddingBlockEnd={5}
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
                  <HistoryRow item={item} thinking={value.thinking} target={target} />
                </VStack>
              );
            })}
          </VStack>
          {/* Astryx must observe live content growth as part of the message list. */}
          {before === '0' && <LiveOutput target={target} history={items} />}
          {completion && <HistoryRow item={completion.item} target={target} />}
        </ChatMessageList>
        {state.data?.task?.error && <Banner status="error" title={state.data.task.error} />}
      </VStack>
    </Center>
  );
}
