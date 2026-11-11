// Purpose: Present durable messages, tool calls and authenticated attachments through Astryx Chat.
// Pattern: Memoized native-history presentation adapter.
// Usage: Chat renders only the virtualized durable history window.
// Related: webui/src/pages/workspace/ui/chat/chat.tsx, webui/src/pages/workspace/ui/inspection/original.tsx, webui/src/pages/workspace/model/tool-status.ts.

import { Button } from '@astryxdesign/core/Button';
import {
  ChatMessage,
  ChatMessageBubble,
  ChatMessageMetadata,
  ChatSystemMessage,
  ChatToolCalls,
} from '@astryxdesign/core/Chat';
import { HStack, VStack } from '@astryxdesign/core/Layout';
import { Lightbox } from '@astryxdesign/core/Lightbox';
import { StatusDot } from '@astryxdesign/core/StatusDot';
import { Text } from '@astryxdesign/core/Text';
import { memo, useState } from 'react';
import { messageContent } from '@/pages/workspace/model/message-content';
import type { Target } from '@/pages/workspace/model/store';
import { toolStatus } from '@/pages/workspace/model/tool-status';
import { AttachmentImage } from '@/pages/workspace/ui/chat/attachment-image';
import { completionSummary } from '@/pages/workspace/ui/chat/format-statistics';
import { Thinking } from '@/pages/workspace/ui/chat/thinking';
import { Original } from '@/pages/workspace/ui/inspection/original';
import type { DetailRequestJson, HistoryItemJson } from '@/shared/api';
import { imageURL } from '@/shared/api';
import { useLocale } from '@/shared/i18n';
import { MarkdownText } from '@/shared/ui/markdown';
import { ReferenceText, ReferenceToken } from '@/shared/ui/reference-token';

export const HistoryRow = memo(function HistoryRow({
  item,
  target,
  thinking,
}: {
  item: HistoryItemJson;
  target: Target;
  thinking?: HistoryItemJson;
}) {
  const t = useLocale(),
    [source, setSource] = useState<DetailRequestJson | null>(null),
    [image, setImage] = useState('');
  const user = item.role === 'user',
    event = item.event;
  const detail = () =>
    setSource(
      event?.tool?.reference ? { tool: event.tool.reference } : { historySequence: item.sequence }
    );
  const content = user ? messageContent(item.text ?? '') : undefined;
  const outcome = item.outcome;
  const outcomeLabel =
    outcome?.status === 'completed'
      ? t('completed')
      : outcome?.status === 'failed'
        ? t('failed')
        : t('interrupted');
  return (
    <>
      {outcome ? (
        <ChatSystemMessage className="completion-row min-w-0">
          <HStack gap={2}>
            {outcome.status !== 'completed' && (
              <StatusDot
                variant={outcome.status === 'failed' ? 'error' : 'warning'}
                label={outcomeLabel}
              />
            )}
            <Text type="supporting" color="secondary">
              {outcomeLabel}
            </Text>
          </HStack>
          {event?.statistics && (
            <Text type="supporting" color="secondary" className="completion-summary">
              {completionSummary(event.statistics, t)}
            </Text>
          )}
          {outcome.status !== 'completed' && outcome.reason && (
            <Text color="secondary">{outcome.reason}</Text>
          )}
        </ChatSystemMessage>
      ) : event?.thinking ? (
        <ChatMessage sender="assistant" name={event.agent || 'MECA'} className="min-w-0 w-full">
          <ChatMessageBubble variant="ghost" width="100%">
            <Thinking text={event.thinking}>
              {event.truncated && <Button label={t('moreOriginal')} size="sm" onClick={detail} />}
            </Thinking>
          </ChatMessageBubble>
        </ChatMessage>
      ) : event?.tool ? (
        <ChatToolCalls
          className="tool-calls"
          style={{ width: '100%', minWidth: 0, maxWidth: '100%' }}
          calls={[
            {
              key: event.tool.callId || item.sequence,
              name: event.tool.name ?? t('toolResult'),
              status: toolStatus(event.kind),
              node: event.agent,
              resultDetail: (
                <VStack gap={3}>
                  {item.text && <MarkdownText text={item.text} showCompleteText={false} />}
                  <Button label={t('details')} size="sm" onClick={detail} />
                </VStack>
              ),
            },
          ]}
        />
      ) : item.text || item.images?.length ? (
        <ChatMessage
          sender={user ? 'user' : 'assistant'}
          style={{ width: '100%', minWidth: 0 }}
          name={user ? undefined : event?.agent || 'MECA'}
        >
          <ChatMessageBubble
            width={user ? undefined : '100%'}
            variant={user ? 'filled' : 'ghost'}
            metadata={
              item.incomplete ? <ChatMessageMetadata footer={t('interrupted')} /> : undefined
            }
          >
            {user ? (
              <VStack className="min-w-0 max-w-full" gap={2}>
                <ReferenceText text={content?.text ?? ''} />
                {!!content?.paths.length && (
                  <HStack gap={2} wrap="wrap" className="min-w-0">
                    {content.paths.map((path) => (
                      <ReferenceToken key={path} value={path} />
                    ))}
                  </HStack>
                )}
              </VStack>
            ) : (
              <>
                {thinking?.event?.thinking && (
                  <Thinking text={thinking.event.thinking}>
                    {thinking.truncated && (
                      <Button
                        label={t('moreOriginal')}
                        size="sm"
                        onClick={() => setSource({ historySequence: thinking.sequence })}
                      />
                    )}
                  </Thinking>
                )}
                <MarkdownText text={item.text ?? ''} showCompleteText={false} />
              </>
            )}
            {!!item.images?.length && (
              <HStack gap={2} wrap="wrap">
                {item.images.map((value, index) => {
                  const url = imageURL(
                    target.workspace,
                    target.session,
                    item.sequence ?? '0',
                    value.index ?? index
                  );
                  return (
                    <AttachmentImage
                      key={value.index ?? index}
                      target={target}
                      sequence={item.sequence ?? '0'}
                      index={value.index ?? index}
                      onOpen={() => setImage(url)}
                    />
                  );
                })}
              </HStack>
            )}
          </ChatMessageBubble>
        </ChatMessage>
      ) : event ? (
        <ChatSystemMessage>{event.skill?.name || event.kind}</ChatSystemMessage>
      ) : null}
      {item.truncated && <Button label={t('moreOriginal')} size="sm" onClick={detail} />}
      {source && <Original target={target} source={source} onClose={() => setSource(null)} />}
      {image && (
        <Lightbox
          isOpen
          onOpenChange={(open) => {
            if (!open) setImage('');
          }}
          media={{ src: image, alt: t('attachment'), type: 'image' }}
        />
      )}
    </>
  );
});
