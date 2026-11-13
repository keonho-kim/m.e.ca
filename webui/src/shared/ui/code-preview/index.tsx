// Purpose: Wrap syntax previews and bound their DOM while exposing every source character in pages.
// Pattern: Astryx code viewer with exact UTF-16 page boundaries.
// Usage: Markdown fences, file previews and original-text dialogs share this view.
// Related: webui/src/shared/ui/markdown/index.tsx, webui/src/pages/workspace/ui/files/file-document.tsx.
import { Button } from '@astryxdesign/core/Button';
import { CodeBlock } from '@astryxdesign/core/CodeBlock';
import { HStack, VStack } from '@astryxdesign/core/Layout';
import { Text } from '@astryxdesign/core/Text';
import { memo, useMemo, useState } from 'react';
import { useLocale } from '@/shared/i18n';
import { codePages } from '@/shared/ui/code-preview/pages';

export const CodePreview = memo(function CodePreview({
  code,
  language,
  follow = false,
  className,
  isWrapped = true,
}: {
  code: string;
  language?: string;
  follow?: boolean;
  className?: string;
  isWrapped?: boolean;
}) {
  const t = useLocale(),
    [page, setPage] = useState(follow ? Number.MAX_SAFE_INTEGER : 0);
  const [indexPages] = useState(codePages);
  const boundaries = useMemo(() => indexPages(code), [code, indexPages]);
  const last = boundaries.length - 2,
    index = Math.min(page, last);
  return (
    <VStack gap={2} className="min-w-0 max-w-full">
      <CodeBlock
        width="100%"
        code={code.slice(boundaries[index], boundaries[index + 1])}
        language={language}
        className={`meca-code min-w-0 max-w-full ${className ?? ''}`}
        isWrapped={isWrapped}
      />
      {last > 0 && (
        <HStack gap={2} justify="end">
          <Button
            size="sm"
            label={t('pagePrevious')}
            isDisabled={index === 0}
            onClick={() => setPage(index - 1)}
          />
          <Text type="supporting">
            {index + 1} / {last + 1}
          </Text>
          <Button
            size="sm"
            label={t('pageNext')}
            isDisabled={index === last}
            onClick={() =>
              setPage(follow && index + 1 === last ? Number.MAX_SAFE_INTEGER : index + 1)
            }
          />
        </HStack>
      )}
    </VStack>
  );
});
