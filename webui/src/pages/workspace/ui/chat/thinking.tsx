// Purpose: Present streamed and saved reasoning as subdued, expandable Markdown.
// Pattern: Shared Astryx content composition.
// Usage: LiveOutput and HistoryRow preserve the same reasoning hierarchy.
// Related: webui/src/shared/ui/markdown/index.tsx, webui/src/app/styles/theme.ts.
import { Collapsible } from '@astryxdesign/core/Collapsible';
import { Text } from '@astryxdesign/core/Text';
import { memo, type ReactNode } from 'react';
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
  return (
    <Collapsible
      className="thinking"
      defaultIsOpen
      trigger={
        <Text type="supporting" color="secondary">
          {t('thinkingLabel')}
        </Text>
      }
    >
      <MarkdownText text={text} streaming={streaming} tone="thinking" />
      {children}
    </Collapsible>
  );
});
