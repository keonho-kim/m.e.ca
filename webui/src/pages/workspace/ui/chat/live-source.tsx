// Purpose: Render one generation without re-rendering unchanged team sources.
// Pattern: Memoized native Chat presentation.
// Usage: LiveOutput passes immutable draft values and one stable original-view action.
// Related: webui/src/pages/workspace/ui/chat/live-output.tsx.
import { Button } from '@astryxdesign/core/Button';
import { ChatMessage, ChatMessageBubble, ChatMessageMetadata } from '@astryxdesign/core/Chat';
import { StatusDot } from '@astryxdesign/core/StatusDot';
import { memo } from 'react';
import type { Draft } from '@/pages/workspace/model/types';
import { Thinking } from '@/pages/workspace/ui/chat/thinking';
import type { DraftRequestJson } from '@/shared/api';
import { useLocale } from '@/shared/i18n';
import { MarkdownText } from '@/shared/ui/markdown';
export const LiveSource = memo(function LiveSource({
  draft: d,
  streaming,
  onOriginal,
}: {
  draft: Draft;
  streaming: boolean;
  onOriginal: (request: DraftRequestJson) => void;
}) {
  const t = useLocale();
  return (
    <ChatMessage
      sender="assistant"
      className="assistant-message"
      key={`${d.task}/${d.agent}/${d.generation}`}
      name={d.agent || 'MECA'}
    >
      <ChatMessageBubble
        variant="ghost"
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
        {d.thinking && (
          <Thinking text={d.thinking} streaming={streaming}>
            {d.truncated && (
              <Button
                size="sm"
                label={t('moreOriginal')}
                onClick={() =>
                  onOriginal({
                    taskId: d.task,
                    agent: d.agent,
                    generation: d.generation,
                    thinking: true,
                  })
                }
              />
            )}
          </Thinking>
        )}
        {!!d.text && <MarkdownText text={d.text} streaming={streaming} />}
        {d.truncated && (
          <Button
            size="sm"
            label={t('moreOriginal')}
            onClick={() => onOriginal({ taskId: d.task, agent: d.agent, generation: d.generation })}
          />
        )}
      </ChatMessageBubble>
    </ChatMessage>
  );
});
