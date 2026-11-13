// Purpose: Show native tool disclosures with nested requests and inline execution results.
// Pattern: Astryx Collapsible trigger containing read-only ChatToolCalls metadata.
// Usage: Saved and live rows share stable execution identities and lazy result reads.
// Related: webui/src/pages/workspace/ui/chat/detail-content.tsx, webui/src/pages/workspace/ui/chat/tool-request.tsx.
import { ChatMessage, ChatMessageBubble, ChatToolCalls } from '@astryxdesign/core/Chat';
import { Collapsible } from '@astryxdesign/core/Collapsible';
import { VStack } from '@astryxdesign/core/Layout';
import { Text } from '@astryxdesign/core/Text';
import { memo, useMemo, useState } from 'react';
import {
  mergeToolEvents,
  toolBatchIdentity,
  toolIdentity,
} from '@/pages/workspace/model/chat-events';
import { type Target, useWorkspaceStore } from '@/pages/workspace/model/store';
import { toolStatus } from '@/pages/workspace/model/tool-status';
import { DetailContent } from '@/pages/workspace/ui/chat/detail-content';
import { ToolRequest } from '@/pages/workspace/ui/chat/tool-request';
import type { EventJson } from '@/shared/api';
import { useLocale } from '@/shared/i18n';

type ToolProps = { target: Target };
export const ToolCalls = memo(function ToolCalls({
  events,
  saved = false,
  target,
}: ToolProps & { events: EventJson[]; saved?: boolean }) {
  const batch = toolBatchIdentity(events[0]),
    identity = toolIdentity(events[0]);
  // Draft-only updates keep the event-array identity, so no per-token filtering runs.
  const source = useWorkspaceStore((state) => (saved ? state.events : null));
  const live = useMemo(
    () =>
      source?.filter(
        (event) =>
          event.workspaceId === target.workspace &&
          event.sessionId === target.session &&
          !!toolIdentity(event) &&
          event.kind !== 'tool_proposed' &&
          event.kind !== 'approved' &&
          (batch ? toolBatchIdentity(event) === batch : toolIdentity(event) === identity)
      ) ?? [],
    [source, target.workspace, target.session, batch, identity]
  );
  const entries = useMemo(
    () => (live.length ? mergeToolEvents(events, live) : events),
    [events, live]
  );
  return (
    <ChatMessage
      sender="assistant"
      name={events[0]?.agent || 'MECA'}
      className="tool-message min-w-0 w-full"
    >
      <ChatMessageBubble variant="ghost" width="100%">
        <VStack gap={4} className="min-w-0 w-full">
          {entries.map((event, index) => (
            <ToolEntry key={toolIdentity(event) ?? index} event={event} target={target} />
          ))}
        </VStack>
      </ChatMessageBubble>
    </ChatMessage>
  );
});
const ToolEntry = memo(function ToolEntry({ event, target }: ToolProps & { event: EventJson }) {
  const t = useLocale(),
    [open, setOpen] = useState(false);
  const reference = event.tool?.reference,
    name = event.tool?.name ?? t('toolResult');
  return (
    <VStack className="tool-entry min-w-0 w-full" gap={0}>
      <Collapsible
        className="tool-disclosure min-w-0 grow"
        data-tone="tool"
        isOpen={open}
        onOpenChange={setOpen}
        chevronPosition="start"
        trigger={
          <ChatToolCalls
            className="tool-calls"
            calls={[{ key: toolIdentity(event), name, status: toolStatus(event) }]}
          />
        }
      >
        {open && (
          <VStack gap={4} width="100%" className="tool-content min-w-0">
            {event.tool && <ToolRequest target={target} tool={event.tool} />}
            {reference ? (
              <DetailContent
                key={JSON.stringify(reference)}
                target={target}
                source={{ tool: reference, view: 'TOOL_RESULT' }}
              />
            ) : (
              <Text type="supporting" color="secondary">
                {event.tool?.summary || t('running')}
              </Text>
            )}
          </VStack>
        )}
      </Collapsible>
    </VStack>
  );
});
