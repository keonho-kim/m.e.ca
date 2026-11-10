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
  const boundaries = useMemo(() => {
    const result = [0];
    let lines = 0,
      start = 0;
    for (let i = 0; i < code.length; i++) {
      if (code.charCodeAt(i) === 10) lines++;
      if (lines < 160 && i + 1 - start < 8192) continue;
      if (
        code.charCodeAt(i) >= 0xd800 &&
        code.charCodeAt(i) <= 0xdbff &&
        code.charCodeAt(i + 1) >= 0xdc00 &&
        code.charCodeAt(i + 1) <= 0xdfff
      )
        i++;
      start = i + 1;
      result.push(start);
      lines = 0;
    }
    if (result.at(-1) !== code.length || result.length === 1) result.push(code.length);
    return result;
  }, [code]);
  const last = boundaries.length - 2,
    index = Math.min(page, last);
  return (
    <VStack gap={2} className="min-w-0 max-w-full">
      <CodeBlock
        code={code.slice(boundaries[index], boundaries[index + 1])}
        language={language}
        className={className}
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
