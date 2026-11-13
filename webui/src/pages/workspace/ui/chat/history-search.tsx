// Purpose: Search server-owned narrative pages and navigate directly to a matching record.
// Pattern: Cancellable, bounded result dialog using Astryx controls.
// Usage: Chat keeps matching detail offsets when opening results in its own transcript.
// Related: internal/serverapi/history_search.go, webui/src/pages/workspace/ui/chat/chat.tsx.
import { Banner } from '@astryxdesign/core/Banner';
import { Button } from '@astryxdesign/core/Button';
import { Dialog, DialogHeader } from '@astryxdesign/core/Dialog';
import { HStack, VStack } from '@astryxdesign/core/Layout';
import { List, ListItem } from '@astryxdesign/core/List';
import { ScrollableArea } from '@astryxdesign/core/ScrollableArea';
import { Text } from '@astryxdesign/core/Text';
import { TextInput } from '@astryxdesign/core/TextInput';
import { useInfiniteQuery } from '@tanstack/react-query';
import { useState } from 'react';
import type { Target } from '@/pages/workspace/model/store';
import { call, type DetailRequestJson } from '@/shared/api';
import { useLocale } from '@/shared/i18n';
import { describeError } from '@/shared/lib/errors';

export function HistorySearch({
  target,
  select,
  close,
}: {
  target: Target;
  select: (source: DetailRequestJson) => void;
  close: () => void;
}) {
  const t = useLocale(),
    [input, setInput] = useState(''),
    [query, setQuery] = useState('');
  const results = useInfiniteQuery({
    queryKey: ['history-search', target.workspace, target.session, query],
    initialPageParam: '',
    enabled: !!query,
    maxPages: 8,
    gcTime: 0,
    queryFn: async ({ pageParam, signal }) =>
      (
        await call(
          target.workspace,
          { searchHistory: { sessionId: target.session, query, cursor: pageParam } },
          signal
        )
      ).historyMatches!,
    getNextPageParam: (page) => page.cursor || undefined,
  });
  const items = results.data?.pages.flatMap((page) => page.items ?? []) ?? [];
  return (
    <Dialog
      isOpen
      width={720}
      purpose="form"
      onOpenChange={(open) => {
        if (!open) close();
      }}
    >
      <DialogHeader title={t('searchConversation')} onOpenChange={close} />
      <VStack gap={4} padding={4}>
        <HStack
          as="form"
          gap={3}
          onSubmit={(event) => {
            event.preventDefault();
            setQuery(input.trim());
          }}
        >
          <TextInput
            type="search"
            label={t('searchConversation')}
            value={input}
            onChange={setInput}
            hasAutoFocus
            isRequired
            width="100%"
          />
          <Button type="submit" label={t('search')} isDisabled={!input.trim()} />
        </HStack>
        <Text type="supporting" color="secondary">
          {t('searchScope')}
        </Text>
        {results.error && <Banner status="error" title={describeError(results.error)} />}
        <ScrollableArea label={t('searchResults')} height="min(45dvh, 28rem)">
          <List>
            {items.map((item) => (
              <ListItem
                key={`${item.sequence}:${item.thinking}`}
                label={<Text className="whitespace-pre-wrap break-words">{item.excerpt}</Text>}
                onClick={() => {
                  select({
                    historySequence: item.sequence,
                    view: item.thinking ? 'THINKING' : 'TEXT',
                    offset: item.offset,
                    revision: item.revision,
                  });
                  close();
                }}
              />
            ))}
          </List>
        </ScrollableArea>
        {!!query && !results.isFetching && !items.length && (
          <Text>{t(results.hasNextPage ? 'searchContinue' : 'searchEmpty')}</Text>
        )}
        <HStack justify="end" gap={3}>
          {results.error && (
            <Button
              label={t('retry')}
              onClick={() => {
                setQuery('');
                queueMicrotask(() => setQuery(input.trim()));
              }}
            />
          )}
          <Button
            label={t('searchContinue')}
            isLoading={results.isFetching}
            isDisabled={!results.hasNextPage || results.isFetching}
            onClick={() => results.fetchNextPage()}
          />
        </HStack>
      </VStack>
    </Dialog>
  );
}
