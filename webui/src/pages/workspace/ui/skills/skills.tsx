// Purpose: Browse skill metadata and revision-bound references without activating or executing skills.
// Pattern: Read-only list/detail work area.
// Usage: !skills, the header and settings shortcut open this surface.
// Related: webui/src/shared/ui/markdown/index.tsx, api/meca/v1/skills.proto.

import { Banner } from '@astryxdesign/core/Banner';
import { Button } from '@astryxdesign/core/Button';
import { Code } from '@astryxdesign/core/Code';
import { EmptyState } from '@astryxdesign/core/EmptyState';
import { Icon } from '@astryxdesign/core/Icon';
import { IconButton } from '@astryxdesign/core/IconButton';
import { HStack, Layout, LayoutContent, LayoutPanel, VStack } from '@astryxdesign/core/Layout';
import { List, ListItem } from '@astryxdesign/core/List';
import { Spinner } from '@astryxdesign/core/Spinner';
import { Text } from '@astryxdesign/core/Text';
import { TextInput } from '@astryxdesign/core/TextInput';
import { Toolbar } from '@astryxdesign/core/Toolbar';
import { useInfiniteQuery, useQuery } from '@tanstack/react-query';
import { type Range, useVirtualizer } from '@tanstack/react-virtual';
import { useCallback, useMemo, useRef, useState } from 'react';
import { reportError } from '@/entities/server-session';
import {
  emptyInput,
  type Target,
  targetKey,
  useWorkspaceStore,
} from '@/pages/workspace/model/store';
import { focusedRange } from '@/pages/workspace/ui/focused-range';
import { call, queries } from '@/shared/api';
import { useLocale } from '@/shared/i18n';
import { MarkdownText } from '@/shared/ui/markdown';
export default function Skills({ target }: { target: Target }) {
  const t = useLocale(),
    [query, setQuery] = useState(''),
    [id, setId] = useState('');
  const [file, setFile] = useState(''),
    [offset, setOffset] = useState('0'),
    [revision, setRevision] = useState('');
  const list = useInfiniteQuery({
    queryKey: ['skills', target.workspace],
    initialPageParam: '',
    maxPages: 4,
    queryFn: async ({ pageParam, signal }) =>
      (await call(target.workspace, { listSkills: { cursor: pageParam, limit: 32 } }, signal))
        .skills!,
    getNextPageParam: (value) => value.nextCursor || undefined,
  });
  const document = useQuery({
    queryKey: ['skill', target.workspace, id],
    enabled: !!id,
    queryFn: async ({ signal }) =>
      (await call(target.workspace, { getSkill: { id } }, signal)).skill!,
  });
  const reference = useQuery({
    queryKey: ['skill-file', target.workspace, id, file, offset, revision],
    enabled: !!id && !!file,
    gcTime: 0,
    queryFn: async ({ signal }) =>
      (
        await call(
          target.workspace,
          {
            readSkillFile: {
              id,
              path: file,
              offset,
              limit: 65536,
              ...(offset === '0' ? {} : { revision }),
            },
          },
          signal
        )
      ).skillFile!,
  });
  const items = useMemo(() => list.data?.pages.flatMap((v) => v.items ?? []) ?? [], [list.data]);
  const visible = useMemo(() => {
    const needle = query.toLowerCase();
    return items.filter((v) => `${v.name} ${v.description}`.toLowerCase().includes(needle));
  }, [items, query]);
  const openReference = (href: string) => {
    try {
      const url = new URL(href, `https://skill.invalid/${file || 'SKILL.md'}`);
      if (url.origin !== 'https://skill.invalid') return;
      setFile(decodeURIComponent(url.pathname.slice(1)));
      setOffset('0');
      setRevision('');
    } catch (error) {
      reportError(error);
    }
  };
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
    <Layout
      className="skills-workspace"
      height="fill"
      padding={0}
      start={
        <LayoutPanel width={240} padding={3}>
          <VStack gap={4}>
            <TextInput label={t('searchSkills')} value={query} onChange={setQuery} />
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
                      label={skill.name}
                      description={skill.description}
                      className="skill-entry"
                      startContent={<Icon icon={'meca:book'} />}
                      isSelected={id === skill.id}
                      onClick={() => {
                        setId(skill.id ?? '');
                        setFile('');
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
        </LayoutPanel>
      }
      content={
        <LayoutContent className="skill-document" padding={5}>
          {document.data ? (
            <VStack gap={6}>
              <Toolbar
                className="document-toolbar"
                label={t('skills')}
                startContent={
                  <VStack gap={1}>
                    <Text type="large" weight="semibold">
                      {document.data.skill?.name}
                    </Text>
                    <Text type="supporting" color="secondary">
                      {t('source')}: {document.data.skill?.source}
                    </Text>
                  </VStack>
                }
                endContent={
                  <Button
                    label={t('insertSkill')}
                    isDisabled={!target.session}
                    onClick={() => {
                      const input =
                          useWorkspaceStore.getState().inputs[targetKey(target)] ?? emptyInput(),
                        mention = document.data.skill?.id || document.data.skill?.name;
                      useWorkspaceStore.getState().patchInput(target, {
                        text: `${input.text}${input.text ? ' ' : ''}#${mention} `,
                      });
                      useWorkspaceStore.getState().set({ mode: 'chat' });
                    }}
                  />
                }
              />
              {(document.error || reference.error) && (
                <Banner status="error" title={t('readError')} />
              )}
              {(document.isFetching || reference.isFetching) && <Spinner label={t('loading')} />}
              {file && (
                <HStack gap={3}>
                  <IconButton
                    label={t('parent')}
                    variant="ghost"
                    icon={<Icon icon={'meca:back'} />}
                    onClick={() => {
                      setFile('');
                      setOffset('0');
                    }}
                  />
                  <Code>{file}</Code>
                </HStack>
              )}
              {file ? (
                reference.data?.binary ? (
                  <Text>
                    {t('readonly')} · {reference.data.mediaType}
                  </Text>
                ) : (
                  <MarkdownText text={reference.data?.text ?? ''} onReference={openReference} />
                )
              ) : (
                <MarkdownText text={document.data.body ?? ''} onReference={openReference} />
              )}
              {file && (
                <HStack gap={3} justify="end">
                  <Text type="supporting">
                    {offset} {t('bytes')}
                  </Text>
                  <Button
                    label={t('parent')}
                    isDisabled={offset === '0'}
                    onClick={() => {
                      setOffset('0');
                      setRevision('');
                    }}
                  />
                  <Button
                    label={t('moreOriginal')}
                    isDisabled={
                      Number(reference.data?.nextOffset ?? 0) >=
                      Number(reference.data?.totalBytes ?? 0)
                    }
                    onClick={() => {
                      setOffset(reference.data?.nextOffset ?? '0');
                      setRevision(reference.data?.revision ?? '');
                    }}
                  />
                </HStack>
              )}
            </VStack>
          ) : (
            <EmptyState title={t('skills')} />
          )}
        </LayoutContent>
      }
    />
  );
}
