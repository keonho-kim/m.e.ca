// Purpose: Present authored team contributions with their sender and structured work context.
// Pattern: Typed conversation presentation over Astryx Chat.
// Usage: Saved and live HistoryRow share this component.
// Related: webui/src/pages/workspace/ui/chat/history-row.tsx, api/meca/v1/events.proto.
import { ChatMessage, ChatMessageBubble } from '@astryxdesign/core/Chat';
import { HStack, VStack } from '@astryxdesign/core/Layout';
import { Text } from '@astryxdesign/core/Text';
import type { Target } from '@/pages/workspace/model/store';
import { DetailContent } from '@/pages/workspace/ui/chat/detail-content';
import type { TeamContributionJson } from '@/shared/api';
import { useLocale } from '@/shared/i18n';
import { MarkdownText } from '@/shared/ui/markdown';
import { ReferenceToken } from '@/shared/ui/reference-token';
export function TeamWorkRow({
  work,
  agent,
  target,
  sequence,
  truncated,
}: {
  work: TeamContributionJson;
  agent?: string;
  target: Target;
  sequence?: string;
  truncated?: boolean;
}) {
  const t = useLocale();
  return (
    <ChatMessage
      sender="assistant"
      name={agent || 'MECA'}
      className="team-contribution min-w-0 w-full"
    >
      <ChatMessageBubble variant="ghost" width="100%">
        <VStack gap={3} width="100%">
          <Text type="supporting" color="secondary">
            {t('teamUpdate')}
          </Text>
          {truncated && sequence ? (
            <DetailContent
              target={target}
              source={{ historySequence: sequence, view: 'TEXT' }}
              narrative
              fallback={work.summary ?? ''}
            />
          ) : (
            <MarkdownText text={work.summary ?? ''} />
          )}
          {!!work.details?.paths?.length && (
            <HStack gap={2} wrap="wrap">
              {work.details.paths.map((path) => (
                <ReferenceToken key={path} value={path} />
              ))}
            </HStack>
          )}
          {work.details?.contract && <MarkdownText text={work.details.contract} />}
        </VStack>
      </ChatMessageBubble>
    </ChatMessage>
  );
}
