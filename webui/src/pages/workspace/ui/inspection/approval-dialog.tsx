// Purpose: Present one pending approval and load original details only when explicitly opened.
// Pattern: Lazy Astryx dialog view over the approval controller.
// Usage: Approval mounts this after the selected conversation has an actionable approval.
// Related: webui/src/pages/workspace/ui/inspection/approval.tsx, webui/src/pages/workspace/ui/inspection/original.tsx.
import { Button } from '@astryxdesign/core/Button';
import { Code } from '@astryxdesign/core/Code';
import { Dialog, DialogHeader } from '@astryxdesign/core/Dialog';
import { Heading } from '@astryxdesign/core/Heading';
import { HStack, Layout, LayoutContent, VStack } from '@astryxdesign/core/Layout';
import { Text } from '@astryxdesign/core/Text';
import { lazy, Suspense, useState } from 'react';
import type { Target } from '@/pages/workspace/model/store';
import type { EventJson } from '@/shared/api';
import { useLocale } from '@/shared/i18n';
import { CodePreview } from '@/shared/ui/code-preview';

const Original = lazy(() =>
  import('@/pages/workspace/ui/inspection/original').then((module) => ({
    default: module.Original,
  }))
);
export default function ApprovalDialog({
  target,
  approval,
  displayRoot,
  busy,
  decide,
}: {
  target: Target;
  approval: NonNullable<EventJson['approval']>;
  displayRoot?: string;
  busy: boolean;
  decide: (allow: boolean) => Promise<void>;
}) {
  const t = useLocale(),
    [full, setFull] = useState(false);
  return (
    <Dialog
      isOpen={true}
      onOpenChange={(open) => {
        if (!open && !busy) void decide(false);
      }}
      purpose="form"
      width={760}
      className="approval-modal"
    >
      <Layout
        header={
          <DialogHeader
            title={t('approval')}
            subtitle={displayRoot}
            onOpenChange={(open) => {
              if (!open && !busy) void decide(false);
            }}
          />
        }
        content={
          <LayoutContent padding={4}>
            <VStack gap={4}>
              <VStack className="approval-content" gap={4}>
                <Heading level={3} type="display-3">
                  {approval.name}
                </Heading>
                <Code>{approval.target}</Code>
                <CodePreview code={approval.preview ?? ''} language="plaintext" isWrapped={true} />
                {approval.expiresAt && (
                  <Text className="muted" as="p">
                    {t('expires')}: {new Date(approval.expiresAt).toLocaleString()}
                  </Text>
                )}
                {approval.truncated && (
                  <Button label={t('moreOriginal')} onClick={() => setFull(true)} />
                )}
              </VStack>
              <HStack className="form-actions" gap={2}>
                <Button label={t('deny')} isDisabled={busy} onClick={() => decide(false)} />
                <Button
                  label={t('approve')}
                  variant="primary"
                  isLoading={busy}
                  onClick={() => decide(true)}
                />
              </HStack>
              {full && (
                <Suspense fallback={null}>
                  <Original
                    target={target}
                    source={{ approvalId: approval.id }}
                    onClose={() => setFull(false)}
                  />
                </Suspense>
              )}
            </VStack>
          </LayoutContent>
        }
      />
    </Dialog>
  );
}
