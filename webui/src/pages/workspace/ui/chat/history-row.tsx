// Purpose: Present durable messages, tool calls and authenticated attachments through Astryx Chat.
// Pattern: Memoized native-history presentation adapter.
// Usage: Chat renders only the virtualized durable history window.
// Related: webui/src/pages/workspace/ui/chat/chat.tsx, webui/src/pages/workspace/ui/inspection/original.tsx, webui/src/pages/workspace/model/tool-status.ts.

import { Button } from '@astryxdesign/core/Button';
import {
  ChatComposerTokenElement,
  ChatMessage,
  ChatMessageBubble,
  ChatMessageMetadata,
  ChatSystemMessage,
  ChatTokenizedText,
  ChatToolCalls,
} from '@astryxdesign/core/Chat';
import { HStack, VStack } from '@astryxdesign/core/Layout';
import { Lightbox } from '@astryxdesign/core/Lightbox';
import { StatusDot } from '@astryxdesign/core/StatusDot';
import { Text } from '@astryxdesign/core/Text';
import { memo, useState } from 'react';
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

export const HistoryRow = memo(function HistoryRow({
  item,
  target,
}: {
  item: HistoryItemJson;
  target: Target;
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
  const matches = user ? ((item.text ?? '').match(/(?:#[\w:.-]+|^![\w-]+)/g) ?? []) : [];
  const tokens = [...new Set(matches.length <= 128 ? matches : [])].slice(0, 32).map((value) => ({
    value,
    render: () => <ChatComposerTokenElement token={{ value, label: value, variant: 'neutral' }} />,
  }));
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
        <ChatSystemMessage>
          <HStack gap={2}>
            <StatusDot
              variant={
                outcome.status === 'completed'
                  ? 'success'
                  : outcome.status === 'failed'
                    ? 'error'
                    : 'warning'
              }
              label={outcomeLabel}
            />
            <Text type="supporting" color="secondary">
              {outcomeLabel}
            </Text>
          </HStack>
          {event?.statistics && (
            <Text type="supporting" color="secondary" className="completion-summary">
              {completionSummary(event.statistics, t)}
            </Text>
          )}
          {outcome.reason && <Text color="secondary">{outcome.reason}</Text>}
        </ChatSystemMessage>
      ) : event?.thinking ? (
        <Thinking text={event.thinking}>
          {event.truncated && <Button label={t('moreOriginal')} size="sm" onClick={detail} />}
        </Thinking>
      ) : event?.tool ? (
        <ChatToolCalls
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
          name={user ? undefined : event?.agent || 'MECA'}
        >
          <ChatMessageBubble
            variant={user ? 'filled' : 'ghost'}
            metadata={
              item.incomplete ? <ChatMessageMetadata footer={t('interrupted')} /> : undefined
            }
          >
            {user ? (
              <ChatTokenizedText tokens={tokens}>{item.text ?? ''}</ChatTokenizedText>
            ) : (
              <MarkdownText text={item.text ?? ''} showCompleteText={false} />
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
