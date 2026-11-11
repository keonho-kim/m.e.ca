// Purpose: Read revision-bound original history or tool text in bounded display pages.
// Pattern: Explicit paged detail viewer.
// Usage: Failure and approval inspection opens this bounded original-data drawer.
// Related: webui/src/pages/workspace/ui/chat/chat.tsx, internal/serverapi/detail.go.

import { Button } from '@astryxdesign/core/Button';
import { Drawer, DrawerHeader } from '@astryxdesign/core/Drawer';
import { HStack, Layout, LayoutContent, VStack } from '@astryxdesign/core/Layout';
import { Text } from '@astryxdesign/core/Text';
import { useQuery } from '@tanstack/react-query';
import { useState } from 'react';
import type { Target } from '@/pages/workspace/model/store';
import type { DetailRequestJson } from '@/shared/api';
import { call } from '@/shared/api';
import { useLocale } from '@/shared/i18n';
import { CodePreview } from '@/shared/ui/code-preview';
export function Original({
  target,
  source,
  onClose,
}: {
  target: Target;
  source?: DetailRequestJson;
  onClose: () => void;
}) {
  const t = useLocale(),
    [offset, setOffset] = useState('0'),
    [revision, setRevision] = useState('');
  const page = useQuery({
    queryKey: ['original', target.workspace, target.session, source, offset, revision],
    gcTime: 0,
    queryFn: async ({ signal }) => {
      return (
        await call(
          target.workspace,
          {
            detail: {
              ...source,
              sessionId: target.session,
              offset,
              limit: 65536,
              ...(offset === '0' ? {} : { revision }),
            },
          },
          signal
        )
      ).detail!;
    },
  });
  const text = page.data?.text ?? '';
  const nextOffset = page.data?.nextOffset ?? offset;
  return (
    <Drawer
      isOpen={true}
      onOpenChange={(open) => {
        if (!open) onClose();
      }}
      label={t('details')}
      side="end"
      width={760}
      hasScrim={!!source?.approvalId}
      className="original-modal"
    >
      <Layout
        header={
          <DrawerHeader
            title={t('details')}
            onOpenChange={(open) => {
              if (!open) onClose();
            }}
          />
        }
        content={
          <LayoutContent padding={4}>
            <VStack gap={4}>
              {page.isPending && <Text as="p">{t('loading')}</Text>}
              {page.error && (
                <Button
                  type="button"
                  onClick={() => {
                    setOffset('0');
                    setRevision('');
                    void page.refetch();
                  }}
                  label={String(t('readError')) + ' ' + '\u00B7' + ' ' + String(t('retry'))}
                  variant="ghost"
                >
                  {t('readError')} · {t('retry')}
                </Button>
              )}
              <CodePreview
                className="original-text"
                code={text}
                language="plaintext"
                isWrapped={true}
              />
              <HStack className="page-controls" gap={2}>
                <Text>
                  {offset} {t('bytes')}
                </Text>
                <Button
                  label={t('parent')}
                  onClick={() => {
                    setOffset('0');
                    setRevision('');
                  }}
                  isDisabled={offset === '0'}
                />
                <Button
                  label={t('pageNext')}
                  isDisabled={!page.data || Number(nextOffset) >= Number(page.data.totalBytes ?? 0)}
                  onClick={() => {
                    setRevision(page.data?.revision ?? '');
                    setOffset(nextOffset);
                  }}
                />
              </HStack>
            </VStack>
          </LayoutContent>
        }
      />
    </Drawer>
  );
}
