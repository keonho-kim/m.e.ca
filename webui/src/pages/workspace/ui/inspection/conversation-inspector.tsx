// Purpose: Present one original or attachment viewer outside virtualized and streaming cells.
// Pattern: Conversation-owned controlled layer host.
// Usage: Chat keeps this host mounted while individual history and live rows change.
// Related: webui/src/pages/workspace/model/inspection.ts, webui/src/pages/workspace/ui/inspection/original.tsx.
import { lazy, memo, Suspense } from 'react';
import type { Inspection } from '@/pages/workspace/model/inspection';
import type { Target } from '@/pages/workspace/model/store';
import { useLocale } from '@/shared/i18n';

const Lightbox = lazy(() =>
  import('@astryxdesign/core/Lightbox').then((module) => ({ default: module.Lightbox }))
);
const Original = lazy(() =>
  import('@/pages/workspace/ui/inspection/original').then((module) => ({
    default: module.Original,
  }))
);
export const ConversationInspector = memo(function ConversationInspector({
  value,
  target,
  close,
}: {
  value: Inspection | null;
  target: Target;
  close: () => void;
}) {
  const t = useLocale();
  if (!value) return null;
  return (
    <Suspense fallback={null}>
      {value.kind === 'image' ? (
        <Lightbox
          isOpen
          onOpenChange={(open) => {
            if (!open) close();
          }}
          media={{ src: value.url, alt: t('attachment'), type: 'image' }}
        />
      ) : (
        <Original
          key={JSON.stringify(value.source)}
          target={target}
          source={value.source}
          onClose={close}
        />
      )}
    </Suspense>
  );
});
