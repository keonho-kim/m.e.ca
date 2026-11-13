// Purpose: Render paged conversation history and live responses without replacing one with a loading flash.
// Pattern: Flow-positioned virtual history with independently subscribed live output.
// Usage: App mounts Chat for the selected conversation.
// Related: webui/src/pages/workspace/ui/chat/history-row.tsx, webui/src/pages/workspace/ui/chat/completion-row.tsx,
// webui/src/pages/workspace/ui/chat/live-output.tsx, webui/src/pages/workspace/ui/inspection/conversation-inspector.tsx.

import { Banner } from '@astryxdesign/core/Banner';
import { Button } from '@astryxdesign/core/Button';
import { Center } from '@astryxdesign/core/Center';
import { ChatMessageList } from '@astryxdesign/core/Chat';
import { Heading } from '@astryxdesign/core/Heading';
import { VStack } from '@astryxdesign/core/Layout';
import { Skeleton } from '@astryxdesign/core/Skeleton';
import { Text } from '@astryxdesign/core/Text';
import { useInfiniteQuery, useQuery } from '@tanstack/react-query';
import { type Range, useVirtualizer } from '@tanstack/react-virtual';
import { lazy, Suspense, useCallback, useLayoutEffect, useMemo, useRef, useState } from 'react';
import { useServerSession } from '@/entities/server-session';
import { stateOptions } from '@/pages/workspace/api/queries';
import { toolBatchIdentity, toolIdentity } from '@/pages/workspace/model/chat-events';
import { reached } from '@/pages/workspace/model/history-handoff';
import { type HistoryJump, historyRows } from '@/pages/workspace/model/history-rows';
import { useConversationInspection } from '@/pages/workspace/model/inspection';
import { useChatPresentationBlocked } from '@/pages/workspace/model/presentation';
import { type Target, targetKey, useWorkspaceStore } from '@/pages/workspace/model/store';
import { activeTask } from '@/pages/workspace/model/task-state';
import { CompletionRow } from '@/pages/workspace/ui/chat/completion-row';
import { HistoryWindow } from '@/pages/workspace/ui/chat/history-window';
import { chatWidth } from '@/pages/workspace/ui/chat/layout';
import { focusedRange } from '@/pages/workspace/ui/focused-range';
import { ConversationInspector } from '@/pages/workspace/ui/inspection/conversation-inspector';
import type { HistoryItemJson } from '@/shared/api';
import { call, queries } from '@/shared/api';
import { useLocale } from '@/shared/i18n';

const LiveOutput = lazy(() =>
  import('@/pages/workspace/ui/chat/live-output').then((module) => ({ default: module.LiveOutput }))
);
const HistorySearch = lazy(() =>
  import('@/pages/workspace/ui/chat/history-search').then((module) => ({
    default: module.HistorySearch,
  }))
);

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
  const blocked = useChatPresentationBlocked();
  const connected = useServerSession((state) => state.connected);
  const t = useLocale(),
    previousSize = useRef<number | null>(null),
    latestRequested = useRef(false);
  const taskId = useWorkspaceStore((s) => activeTask(target, s.tasks)?.id);
  const completionThrough = useWorkspaceStore((s) => s.completionThrough);
  const hasActivity = useWorkspaceStore((s) => s.events.length > 0);
  const hasOutput = useWorkspaceStore((s) =>
    Object.values(s.drafts).some((draft) => draft.text || draft.thinking)
  );
  const followedTask = useRef<string | undefined>(undefined);
  const state = useQuery(stateOptions(target));
  const [before, setBefore] = useState('0');
  const [focused, setFocused] = useState(-1);
  const [searchOpen, setSearchOpen] = useState(false),
    [match, setMatch] = useState<HistoryJump>();
  const searchRequested = useRef(false);
  const inspector = useConversationInspection();
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
      for (const item of pages[index]?.items ?? []) result.push(item);
    return result;
  }, [history.data]);
  const rows = useMemo(() => historyRows(items), [items]);
  const last = rows.at(-1);
  const completion = before === '0' && last && 'completion' in last ? last : undefined;
  const visibleRows = completion ? rows.slice(0, -1) : rows;
  const getItemKey = useCallback(
    (index: number) => {
      const row = visibleRows[index];
      return (
        (row &&
          ('item' in row
            ? (row.item.event &&
                (toolBatchIdentity(row.item.event) || toolIdentity(row.item.event))) ||
              row.item.sequence
            : row.sequence)) ??
        index
      );
    },
    [visibleRows]
  );
  const rangeExtractor = useCallback((range: Range) => focusedRange(range, focused), [focused]);
  const virtual = useVirtualizer({
    count: visibleRows.length,
    getScrollElement: () => scroller,
    estimateSize: () => 140,
    overscan: 5,
    getItemKey,
    rangeExtractor,
  });
  useLayoutEffect(() => {
    if (!history.data || before !== '0') return;
    const seen = history.data.pages[0]?.clientRevision;
    if (seen === undefined) return;
    useWorkspaceStore
      .getState()
      .acknowledgeHistory(target, seen, { through: history.data.pages[0]?.through, items });
  }, [history.data, before, target, items]);
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
  useLayoutEffect(() => {
    if (searchRequested.current && history.isSuccess && !history.isFetching) {
      searchRequested.current = false;
      virtual.scrollToIndex(Math.max(0, visibleRows.length - 1), { align: 'start' });
    }
  }, [history.isSuccess, history.isFetching, virtual, visibleRows.length]);
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
    <Center width="100%" className="meca-pane-enter min-w-0">
      <VStack width={chatWidth} paddingBlock={4} gap={5} className="chat-content min-w-0">
        <Button
          label={t('searchConversation')}
          variant="ghost"
          size="sm"
          className="self-end"
          onClick={() => setSearchOpen(true)}
        />
        {searchOpen && (
          <Suspense fallback={null}>
            <HistorySearch
              target={target}
              close={() => setSearchOpen(false)}
              select={(source) => {
                setMatch((previous) => ({ source, serial: (previous?.serial ?? 0) + 1 }));
                setBefore(String(BigInt(source.historySequence ?? '0') + 1n));
                searchRequested.current = true;
              }}
            />
          </Suspense>
        )}
        {empty && (
          <VStack gap={3} paddingBlock={6}>
            <Heading level={2} type="display-2">
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
          isStreaming={!!taskId}
          align="top"
          gap={0}
          style={{ width: '100%', minWidth: 0, maxWidth: '100%' }}
        >
          <HistoryWindow
            rows={visibleRows}
            window={virtual.getVirtualItems()}
            total={virtual.getTotalSize()}
            measure={virtual.measureElement}
            target={target}
            inspect={inspector.actions}
            match={match}
            setFocused={setFocused}
            blocked={blocked && connected}
          />
          {/* Astryx must observe live content growth as part of the message list. */}
          {before === '0' && (taskId || hasOutput || hasActivity) && (
            <Suspense fallback={<Skeleton height={80} />}>
              <LiveOutput target={target} history={items} inspect={inspector.actions} />
            </Suspense>
          )}
          {completion &&
            !taskId &&
            !hasOutput &&
            !hasActivity &&
            completionThrough !== null &&
            reached(history.data?.pages[0]?.through, completionThrough) && (
              <CompletionRow
                completion={completion.completion}
                sequence={completion.sequence}
                onDetails={inspector.actions.history}
              />
            )}
        </ChatMessageList>
        {state.data?.task?.error && <Banner status="error" title={state.data.task.error} />}
        <ConversationInspector value={inspector.value} target={target} close={inspector.close} />
      </VStack>
    </Center>
  );
}
