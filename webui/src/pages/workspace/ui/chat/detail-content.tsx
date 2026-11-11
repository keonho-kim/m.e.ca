// Purpose: Render a bounded detail page inline with native loading, failure and pagination controls.
// Pattern: Astryx view over the shared detail-query owner.
// Usage: Tool results, request dialogs and truncated saved narratives keep the same byte-page contract.
// Related: webui/src/pages/workspace/model/detail-pages.ts, webui/src/pages/workspace/ui/chat/tool-value.tsx.
import { Button } from '@astryxdesign/core/Button';
import { HStack, VStack } from '@astryxdesign/core/Layout';
import { Skeleton } from '@astryxdesign/core/Skeleton';
import { Text } from '@astryxdesign/core/Text';
import { useDetailPage } from '@/pages/workspace/model/detail-pages';
import type { Target } from '@/pages/workspace/model/store';
import { ToolValue } from '@/pages/workspace/ui/chat/tool-value';
import type { DetailRequestJson } from '@/shared/api';
import { useLocale } from '@/shared/i18n';
import { MarkdownText } from '@/shared/ui/markdown';
export function DetailContent({
  target,
  source,
  narrative = false,
  request = false,
  fallback,
}: {
  target: Target;
  source: DetailRequestJson;
  narrative?: boolean;
  request?: boolean;
  fallback?: string;
}) {
  const t = useLocale(),
    state = useDetailPage(target, source),
    data = state.page.data;
  return (
    <VStack gap={4} className="detail-content meca-content-reveal min-w-0 w-full">
      {state.page.isPending ? (
        fallback ? (
          <MarkdownText text={fallback} />
        ) : (
          <Skeleton height={40} />
        )
      ) : state.page.error ? (
        <Button label={t('retry')} onClick={state.retry} variant="ghost" />
      ) : data ? (
        narrative ? (
          <MarkdownText
            text={data.text ?? ''}
            tone={source.view === 'THINKING' ? 'thinking' : undefined}
          />
        ) : (
          <ToolValue text={data.text ?? ''} request={request} />
        )
      ) : null}
      {(state.more || state.hasPrevious) && (
        <HStack gap={3} align="center" wrap="wrap" role="group" aria-label={t('sourceRange')}>
          <Button
            size="sm"
            variant="ghost"
            label={t('pagePrevious')}
            isDisabled={!state.hasPrevious || state.page.isFetching}
            onClick={state.previous}
          />
          <Text type="supporting" color="secondary">
            {t('sourceRange')} {state.offset} / {data?.totalBytes ?? '0'} B
          </Text>
          <Button
            size="sm"
            variant="ghost"
            label={t('pageNext')}
            isDisabled={!state.more || state.page.isFetching}
            onClick={state.next}
          />
        </HStack>
      )}
    </VStack>
  );
}
