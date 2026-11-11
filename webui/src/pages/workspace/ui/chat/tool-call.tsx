// Purpose: Show borderless native tool disclosures with inline results and a separate request action.
// Pattern: Astryx Collapsible trigger containing read-only ChatToolCalls metadata.
// Usage: Saved and live rows share stable execution identities and lazy result reads.
// Related: webui/src/pages/workspace/ui/chat/detail-content.tsx, webui/src/pages/workspace/model/inspection.ts.
import { ChatMessage, ChatMessageBubble, ChatToolCalls } from '@astryxdesign/core/Chat';
import { Collapsible } from '@astryxdesign/core/Collapsible';
import { Icon } from '@astryxdesign/core/Icon';
import { IconButton } from '@astryxdesign/core/IconButton';
import { VStack } from '@astryxdesign/core/Layout';
import { Text } from '@astryxdesign/core/Text';
import { Tooltip } from '@astryxdesign/core/Tooltip';
import { memo, useState } from 'react';
import { toolIdentity } from '@/pages/workspace/model/chat-events';
import type { InspectionActions } from '@/pages/workspace/model/inspection';
import type { Target } from '@/pages/workspace/model/store';
import { toolArguments } from '@/pages/workspace/model/tool-content';
import { toolStatus } from '@/pages/workspace/model/tool-status';
import { DetailContent } from '@/pages/workspace/ui/chat/detail-content';
import type { EventJson } from '@/shared/api';
import { useLocale } from '@/shared/i18n';
export const ToolCall = memo(function ToolCall({
  event,
  target,
  inspect,
}: {
  event: EventJson;
  target: Target;
  inspect: InspectionActions;
}) {
  const t = useLocale(),
    [open, setOpen] = useState(false);
  const reference = event.tool?.reference,
    name = event.tool?.name ?? t('toolResult');
  return (
    <ChatMessage
      sender="assistant"
      name={event.agent || undefined}
      className="tool-message min-w-0 w-full"
    >
      <ChatMessageBubble variant="ghost" width="100%">
        <VStack className="tool-entry relative min-w-0 w-full" gap={0}>
          <Collapsible
            className="tool-disclosure min-w-0 grow"
            data-tone="tool"
            isOpen={open}
            onOpenChange={setOpen}
            chevronPosition="start"
            trigger={
              <ChatToolCalls
                className="tool-calls pe-10"
                calls={[{ key: toolIdentity(event), name, status: toolStatus(event.kind) }]}
              />
            }
          >
            {open &&
              (reference ? (
                <DetailContent
                  key={JSON.stringify(reference)}
                  target={target}
                  source={{ tool: reference, view: 'TOOL_RESULT' }}
                />
              ) : (
                <Text type="supporting" color="secondary">
                  {event.text || t('running')}
                </Text>
              ))}
          </Collapsible>
          <Tooltip content={t('toolRequest')}>
            <IconButton
              className="tool-request-action shrink-0"
              size="md"
              label={`${t('toolRequest')} ${name}`}
              variant="tool-request"
              icon={<Icon icon="meca:request" />}
              onClick={() =>
                inspect.request({
                  name,
                  reference,
                  args: reference ? undefined : toolArguments(event.tool?.argumentsJson),
                })
              }
            />
          </Tooltip>
        </VStack>
      </ChatMessageBubble>
    </ChatMessage>
  );
});
