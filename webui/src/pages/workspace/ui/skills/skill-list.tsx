// Purpose: Browse bounded skill metadata and insert one skill from its list row.
// Pattern: Virtualized Astryx selection list.
// Usage: The shared skill browser uses this list in work areas and composer dialogs.
// Related: webui/src/pages/workspace/ui/skills/skills.tsx, webui/src/pages/workspace/ui/focused-range.ts.
import { Banner } from '@astryxdesign/core/Banner';
import { Button } from '@astryxdesign/core/Button';
import { Icon } from '@astryxdesign/core/Icon';
import { IconButton } from '@astryxdesign/core/IconButton';
import { VStack } from '@astryxdesign/core/Layout';
import { List, ListItem } from '@astryxdesign/core/List';
import { Spinner } from '@astryxdesign/core/Spinner';
import { Text } from '@astryxdesign/core/Text';
import { TextInput } from '@astryxdesign/core/TextInput';
import { Tooltip } from '@astryxdesign/core/Tooltip';
import { useInfiniteQuery } from '@tanstack/react-query';
import { type Range, useVirtualizer } from '@tanstack/react-virtual';
import { useCallback, useMemo, useRef, useState } from 'react';
import type { Target } from '@/pages/workspace/model/store';
import { focusedRange } from '@/pages/workspace/ui/focused-range';
import { call, queries } from '@/shared/api';
import { useLocale } from '@/shared/i18n';
export function SkillList({
  target,
  id,
  onSelect,
  onInsert,
  query,
  search,
}: {
  target: Target;
  id: string;
  onSelect: (id: string) => void;
  onInsert: (mention: string) => void;
  query: string;
  search: (query: string) => void;
}) {
  const t = useLocale();
  const list = useInfiniteQuery({
    queryKey: ['skills', target.workspace],
    initialPageParam: '',
    maxPages: 4,
    queryFn: async ({ pageParam, signal }) =>
      (await call(target.workspace, { listSkills: { cursor: pageParam, limit: 32 } }, signal))
        .skills!,
    getNextPageParam: (value) => value.nextCursor || undefined,
  });
  const items = useMemo(() => list.data?.pages.flatMap((v) => v.items ?? []) ?? [], [list.data]);
  const visible = useMemo(() => {
    const needle = query.toLowerCase();
    return items.filter((v) => `${v.name} ${v.description}`.toLowerCase().includes(needle));
  }, [items, query]);
  const scroller = useRef<HTMLDivElement>(null);
  const [focused, setFocused] = useState(-1);
  const rangeExtractor = useCallback((range: Range) => focusedRange(range, focused), [focused]);
  const getItemKey = useCallback((index: number) => visible[index]?.id ?? index, [visible]);
  const virtual = useVirtualizer({
    count: visible.length,
    getScrollElement: () => scroller.current,
    estimateSize: () => 72,
    overscan: 2,
    getItemKey,
    rangeExtractor,
  });
  return (
    <VStack gap={4} padding={3} className="min-w-0">
      <TextInput label={t('searchSkills')} value={query} onChange={search} hasAutoFocus />
      {!!list.data?.pageParams[0] && (
        <Button
          label={t('listStart')}
          onClick={() =>
            queries.resetQueries({ queryKey: ['skills', target.workspace], exact: true })
          }
        />
      )}
      {list.isPending && <Spinner label={t('loading')} />}
      {list.error && (
        <Button
          label={t('retry')}
          onClick={() =>
            queries.resetQueries({ queryKey: ['skills', target.workspace], exact: true })
          }
        />
      )}
      <VStack
        ref={scroller}
        style={{ height: `min(55vh, ${virtual.getTotalSize()}px)`, overflowY: 'auto' }}
      >
        <List
          style={{ height: virtual.getTotalSize(), position: 'relative' }}
          onBlurCapture={(event) => {
            if (!event.currentTarget.contains(event.relatedTarget)) setFocused(-1);
          }}
        >
          {virtual.getVirtualItems().map((row) => {
            const skill = visible[row.index];
            if (!skill) return null;
            return (
              <ListItem
                key={skill.id}
                ref={virtual.measureElement}
                data-index={row.index}
                onFocusCapture={() => setFocused(row.index)}
                style={{
                  position: 'absolute',
                  top: 0,
                  left: 0,
                  width: '100%',
                  transform: `translateY(${row.start}px)`,
                }}
                label={
                  <Text maxLines={1} hasTruncateTooltip>
                    {skill.name}
                  </Text>
                }
                description={skill.description}
                className="skill-entry"
                startContent={<Icon icon={'meca:book'} />}
                isSelected={id === skill.id}
                endContent={
                  <Tooltip content={t('insertSkill')}>
                    <IconButton
                      label={`${skill.name} ${t('insertSkill')}`}
                      variant="ghost"
                      icon={<Icon icon="meca:add" />}
                      isDisabled={!target.session}
                      onClick={(event) => {
                        event.stopPropagation();
                        onInsert(skill.id || skill.name || '');
                      }}
                    />
                  </Tooltip>
                }
                onClick={() => {
                  onSelect(skill.id ?? '');
                }}
              />
            );
          })}
        </List>
      </VStack>
      {!list.isPending && !visible.length && <Text color="secondary">{t('noSkills')}</Text>}
      {list.data?.pages
        .flatMap((v) => v.diagnostics ?? [])
        .map((value) => (
          <Banner key={value.id} status="warning" title={value.message} />
        ))}
      {list.hasNextPage && (
        <Button
          label={t('loadMore')}
          onClick={() => list.fetchNextPage()}
          isDisabled={list.isFetchingNextPage}
        />
      )}
    </VStack>
  );
}
