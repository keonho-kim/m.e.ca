// Purpose: Render one generation without re-rendering unchanged team sources.
// Pattern: Memoized native Chat presentation.
// Usage: LiveOutput passes immutable draft values and the current stream status.
// Related: webui/src/pages/workspace/ui/chat/live-output.tsx.
import { ChatMessage, ChatMessageBubble, ChatMessageMetadata } from '@astryxdesign/core/Chat';
import { VStack } from '@astryxdesign/core/Layout';
import { StatusDot } from '@astryxdesign/core/StatusDot';
import { memo } from 'react';
import type { Draft } from '@/pages/workspace/model/types';
import { Thinking } from '@/pages/workspace/ui/chat/thinking';
import { useLocale } from '@/shared/i18n';
import { MarkdownText } from '@/shared/ui/markdown';
export const LiveSource = memo(function LiveSource({
  draft: d,
  streaming,
}: {
  draft: Draft;
  streaming: boolean;
}) {
  const t = useLocale();
  return (
    <ChatMessage
      sender="assistant"
      className="assistant-message"
      style={{ width: '100%', minWidth: 0 }}
      name={d.agent || 'MECA'}
    >
      <ChatMessageBubble
        variant="ghost"
        width="100%"
        metadata={
          <ChatMessageMetadata
            footer={
              d.doneRevision === undefined ? (
                <StatusDot variant="accent" isPulsing={streaming} label={t('running')} />
              ) : undefined
            }
          />
        }
      >
        <VStack gap={6} width="100%" className="response-content">
          {d.thinking && <Thinking text={d.thinking} streaming={streaming} />}
          {!!d.text && <MarkdownText text={d.text} streaming={streaming} />}
        </VStack>
      </ChatMessageBubble>
    </ChatMessage>
  );
});
