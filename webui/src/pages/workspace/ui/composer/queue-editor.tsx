// Purpose: Own a queued request's temporary text only while its editing dialog exists.
// Pattern: Mount-scoped form.
// Usage: Queue mounts this for one current queue entry and discards its text on close or removal.
// Related: webui/src/pages/workspace/ui/composer/queue.tsx, webui/src/pages/workspace/model/followups.ts.
import { Button } from '@astryxdesign/core/Button';
import { Dialog, DialogHeader } from '@astryxdesign/core/Dialog';
import { HStack, Layout, LayoutContent, VStack } from '@astryxdesign/core/Layout';
import { TextArea } from '@astryxdesign/core/TextArea';
import { useState } from 'react';
import { type Queued, type Target, useWorkspaceStore } from '@/pages/workspace/model/store';
import { useLocale } from '@/shared/i18n';

export function QueueEditor({
  target,
  entry,
  close,
}: {
  target: Target;
  entry: Queued;
  close: () => void;
}) {
  const t = useLocale();
  const [text, setText] = useState(entry.text);
  return (
    <Dialog
      isOpen
      onOpenChange={(open) => {
        if (!open) close();
      }}
      purpose="form"
      width={760}
      className="queue-modal"
    >
      <Layout
        header={
          <DialogHeader
            title={t('queueTitle')}
            onOpenChange={(open) => {
              if (!open) close();
            }}
          />
        }
        content={
          <LayoutContent padding={4}>
            <VStack gap={4}>
              <TextArea label={t('input')} value={text} onChange={setText} />
              <HStack gap={2}>
                <Button label={t('cancel')} onClick={close} />
                <Button
                  label={t('save')}
                  variant="primary"
                  onClick={() => {
                    if (useWorkspaceStore.getState().updateQueue(target, entry.id, text)) close();
                  }}
                />
              </HStack>
            </VStack>
          </LayoutContent>
        }
      />
    </Dialog>
  );
}
