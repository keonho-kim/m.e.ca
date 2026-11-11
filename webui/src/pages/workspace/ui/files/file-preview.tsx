// Purpose: Render read-only file pages and navigation in the document layout slots.
// Pattern: Astryx presentation over a page-owned query state.
// Usage: FileDocument supplies the revision-bound paging state.
// Related: webui/src/pages/workspace/model/file-preview.ts, webui/src/shared/ui/code-preview/index.tsx.
import { Banner } from '@astryxdesign/core/Banner';
import { Button } from '@astryxdesign/core/Button';
import { HStack } from '@astryxdesign/core/Layout';
import { Spinner } from '@astryxdesign/core/Spinner';
import { Text } from '@astryxdesign/core/Text';
import type { useFilePreview } from '@/pages/workspace/model/file-preview';
import { useLocale } from '@/shared/i18n';
import { codeLanguage } from '@/shared/lib/code-language';
import { describeError } from '@/shared/lib/errors';
import { CodePreview } from '@/shared/ui/code-preview';
import { MarkdownText } from '@/shared/ui/markdown';

type Preview = ReturnType<typeof useFilePreview>;
export function FilePreview({
  state,
  path,
  mode,
}: {
  state: Preview;
  path: string;
  mode: 'raw' | 'preview';
}) {
  const t = useLocale();
  return (
    <>
      {state.pending && <Spinner label={t('loading')} />}
      {state.error && (
        <Banner
          status="error"
          title={describeError(state.error)}
          endContent={<Button label={t('retry')} onClick={() => state.retry()} />}
        />
      )}
      {state.data &&
        (mode === 'preview' ? (
          <MarkdownText text={state.data.text ?? ''} />
        ) : (
          <CodePreview code={state.data.text ?? ''} language={codeLanguage(path)} />
        ))}
    </>
  );
}
export function FilePages({ state }: { state: Preview }) {
  const t = useLocale();
  return (
    <HStack gap={2} padding={3} justify="end" wrap="wrap">
      <Text type="supporting" color="secondary">
        {state.offset} {t('bytes')}
      </Text>
      <Button label={t('parent')} isDisabled={!state.hasPrevious} onClick={state.previous} />
      <Button label={t('moreOriginal')} isDisabled={!state.hasNext} onClick={state.next} />
    </HStack>
  );
}
