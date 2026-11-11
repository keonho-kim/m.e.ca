// Purpose: Read skill documents and references, and insert selected mentions into the current draft.
// Pattern: List/detail work area over a bounded metadata list.
// Usage: !skills, navigation and the composer's skill dialog share this browser.
// Related: webui/src/pages/workspace/ui/skills/skill-list.tsx, webui/src/pages/workspace/ui/skills/skill-picker.tsx, api/meca/v1/skills.proto.

import { Banner } from '@astryxdesign/core/Banner';
import { Button } from '@astryxdesign/core/Button';
import { Code } from '@astryxdesign/core/Code';
import { EmptyState } from '@astryxdesign/core/EmptyState';
import { useMediaQuery } from '@astryxdesign/core/hooks';
import { Icon } from '@astryxdesign/core/Icon';
import { IconButton } from '@astryxdesign/core/IconButton';
import { HStack, Layout, LayoutContent, LayoutPanel, VStack } from '@astryxdesign/core/Layout';
import { Spinner } from '@astryxdesign/core/Spinner';
import { Text } from '@astryxdesign/core/Text';
import { Toolbar } from '@astryxdesign/core/Toolbar';
import { useQuery } from '@tanstack/react-query';
import { useState } from 'react';
import { reportError } from '@/entities/server-session';
import type { Target } from '@/pages/workspace/model/store';
import { SkillList } from '@/pages/workspace/ui/skills/skill-list';
import { call } from '@/shared/api';
import { useLocale } from '@/shared/i18n';
import { MarkdownText } from '@/shared/ui/markdown';
export default function Skills({
  target,
  onInsert,
}: {
  target: Target;
  onInsert: (mention: string) => void;
}) {
  const t = useLocale(),
    [id, setId] = useState('');
  const compact = useMediaQuery('(max-width: 1024px)');
  const [details, setDetails] = useState(false),
    [query, search] = useState('');
  const insert = (mention: string) => {
    if (!mention || !target.session) return;
    onInsert(mention);
  };
  const [file, setFile] = useState(''),
    [offset, setOffset] = useState('0'),
    [revision, setRevision] = useState('');
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
  const list = (
    <SkillList
      target={target}
      id={id}
      query={query}
      search={search}
      onSelect={(id) => {
        setId(id);
        setFile('');
        setDetails(true);
      }}
      onInsert={insert}
    />
  );
  return (
    <Layout
      className="skills-workspace meca-pane-enter min-w-0"
      height="fill"
      padding={0}
      start={
        !compact ? (
          <LayoutPanel width={320} padding={0}>
            {list}
          </LayoutPanel>
        ) : undefined
      }
      header={
        compact && details ? (
          <Toolbar
            label={t('skills')}
            startContent={
              <Button
                label={t('backToSkills')}
                variant="ghost"
                icon={<Icon icon="meca:back" />}
                onClick={() => setDetails(false)}
              />
            }
          />
        ) : undefined
      }
      content={
        compact && !details ? (
          <LayoutContent padding={0}>{list}</LayoutContent>
        ) : (
          <LayoutContent className="skill-document min-w-0" padding={5}>
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
        )
      }
    />
  );
}
