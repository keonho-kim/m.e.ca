// Purpose: Present durable messages, tool calls and authenticated attachments through Astryx Chat.
// Pattern: Memoized native-history presentation adapter.
// Usage: Chat renders only the virtualized durable history window.
// Related: webui/src/pages/workspace/ui/chat/chat.tsx, webui/src/pages/workspace/ui/chat/tool-call.tsx, webui/src/pages/workspace/model/inspection.ts.

import { ChatMessage, ChatMessageBubble, ChatMessageMetadata } from '@astryxdesign/core/Chat';
import { HStack, VStack } from '@astryxdesign/core/Layout';
import { memo } from 'react';
import type { HistoryJump, HistoryMessageItem } from '@/pages/workspace/model/history-rows';
import type { InspectionActions } from '@/pages/workspace/model/inspection';
import { messageContent } from '@/pages/workspace/model/message-content';
import type { Target } from '@/pages/workspace/model/store';
import { AttachmentImage } from '@/pages/workspace/ui/chat/attachment-image';
import { DetailContent } from '@/pages/workspace/ui/chat/detail-content';
import { NoticeRow } from '@/pages/workspace/ui/chat/notice-row';
import { TeamWorkRow } from '@/pages/workspace/ui/chat/team-work-row';
import { Thinking } from '@/pages/workspace/ui/chat/thinking';
import { ToolCalls } from '@/pages/workspace/ui/chat/tool-call';
import { type DetailRequestJson, imageURL } from '@/shared/api';
import { useLocale } from '@/shared/i18n';
import { MarkdownText } from '@/shared/ui/markdown';
import { ReferenceText, ReferenceToken } from '@/shared/ui/reference-token';

export const HistoryRow = memo(function HistoryRow({
  item,
  target,
  thinking,
  inspect,
  tools,
  match,
}: {
  inspect: InspectionActions;
  item: HistoryMessageItem;
  target: Target;
  thinking?: HistoryMessageItem;
  tools?: HistoryMessageItem[];
  match?: HistoryJump;
}) {
  const t = useLocale();
  const user = item.role === 'user',
    event = item.event;
  const content = user ? messageContent(item.text ?? '') : undefined;
  const request = match?.source;
  const detailKey = (sequence: string | undefined, view: 'TEXT' | 'THINKING') =>
    request?.historySequence === sequence && request?.view === view ? match?.serial : undefined;
  const source = (sequence: string | undefined, view: 'TEXT' | 'THINKING'): DetailRequestJson => ({
    historySequence: sequence,
    view,
    ...(request && request.historySequence === sequence && request.view === view
      ? { offset: request.offset, revision: request.revision }
      : {}),
  });
  return (
    <>
      {event?.message?.thinking ? (
        <ChatMessage sender="assistant" name={event.agent || 'MECA'} className="min-w-0 w-full">
          <ChatMessageBubble variant="ghost" width="100%">
            <Thinking text={event.message.thinking}>
              {item.truncated ? (
                <DetailContent
                  target={target}
                  source={source(item.sequence, 'THINKING')}
                  key={detailKey(item.sequence, 'THINKING')}
                  narrative
                  fallback={event.message.thinking}
                />
              ) : undefined}
            </Thinking>
          </ChatMessageBubble>
        </ChatMessage>
      ) : event?.work ? (
        <TeamWorkRow
          work={event.work}
          agent={event.agent}
          target={target}
          sequence={item.sequence}
          truncated={item.truncated}
        />
      ) : event?.tool ? (
        <ToolCalls
          target={target}
          events={tools?.flatMap((item) => (item.event ? [item.event] : [])) ?? [event]}
          saved
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
                {item.truncated ? (
                  <DetailContent
                    target={target}
                    source={source(item.sequence, 'TEXT')}
                    key={detailKey(item.sequence, 'TEXT')}
                    narrative
                    fallback={content?.text ?? ''}
                  />
                ) : (
                  <ReferenceText text={content?.text ?? ''} />
                )}
                {!!content?.paths.length && (
                  <HStack gap={2} wrap="wrap" className="min-w-0">
                    {content.paths.map((path) => (
                      <ReferenceToken key={path} value={path} />
                    ))}
                  </HStack>
                )}
              </VStack>
            ) : (
              <VStack gap={6} width="100%" className="response-content">
                {thinking?.event?.message?.thinking && (
                  <Thinking text={thinking.event.message.thinking}>
                    {thinking.truncated ? (
                      <DetailContent
                        target={target}
                        source={source(thinking.sequence, 'THINKING')}
                        key={detailKey(thinking.sequence, 'THINKING')}
                        narrative
                        fallback={thinking.event.message.thinking}
                      />
                    ) : undefined}
                  </Thinking>
                )}
                {item.truncated ? (
                  <DetailContent
                    target={target}
                    source={source(item.sequence, 'TEXT')}
                    key={detailKey(item.sequence, 'TEXT')}
                    narrative
                    fallback={item.text ?? ''}
                  />
                ) : (
                  <MarkdownText text={item.text ?? ''} />
                )}
              </VStack>
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
                      onOpen={() => inspect.image(url)}
                    />
                  );
                })}
              </HStack>
            )}
          </ChatMessageBubble>
        </ChatMessage>
      ) : event?.notice ? (
        <NoticeRow notice={event.notice} />
      ) : null}
    </>
  );
});
