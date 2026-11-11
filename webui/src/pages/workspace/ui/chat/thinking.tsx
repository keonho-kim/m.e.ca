// Purpose: Present streamed and saved reasoning as subdued, expandable Markdown.
// Pattern: Shared Astryx content composition.
// Usage: LiveOutput and HistoryRow preserve the same reasoning hierarchy.
// Related: webui/src/shared/ui/markdown/index.tsx, webui/src/app/styles/theme.ts.
import { Collapsible } from '@astryxdesign/core/Collapsible';
import { Text } from '@astryxdesign/core/Text';
import { memo, type ReactNode, useState } from 'react';
import { useLocale } from '@/shared/i18n';
import { MarkdownText } from '@/shared/ui/markdown';

export const Thinking = memo(function Thinking({
  text,
  streaming = false,
  children,
}: {
  text: string;
  streaming?: boolean;
  children?: ReactNode;
}) {
  const t = useLocale();
  const [open, setOpen] = useState(true);
  return (
    <Collapsible
      className="thinking"
      data-tone="thinking"
      style={{ width: '100%', minWidth: 0 }}
      isOpen={open}
      onOpenChange={setOpen}
      trigger={
        <Text type="supporting" color="secondary">
          {t('thinkingLabel')}
        </Text>
      }
    >
      {open && (children ?? <MarkdownText text={text} streaming={streaming} tone="thinking" />)}
    </Collapsible>
  );
});
