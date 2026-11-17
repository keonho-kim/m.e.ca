// Purpose: Present the attachment viewer outside virtualized and streaming cells.
// Pattern: Conversation-owned controlled layer host.
// Usage: Chat keeps this host mounted while individual history and live rows change.
// Related: webui/src/pages/workspace/model/inspection.ts, webui/src/pages/workspace/ui/chat/history-row.tsx.
import { lazy, memo, Suspense } from 'react';
import type { Inspection } from '@/pages/workspace/model/inspection';
import { useLocale } from '@/shared/i18n';

const Lightbox = lazy(() =>
  import('@astryxdesign/core/Lightbox').then((module) => ({ default: module.Lightbox }))
);
export const ConversationInspector = memo(function ConversationInspector({
  value,
  close,
}: {
  value: Inspection | null;
  close: () => void;
}) {
  const t = useLocale();
  if (!value) return null;
  return (
    <Suspense fallback={null}>
      <Lightbox
        isOpen
        onOpenChange={(open) => {
          if (!open) close();
        }}
        media={{ src: value.url, alt: t('attachment'), type: 'image' }}
      />
    </Suspense>
  );
});
