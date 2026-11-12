// Purpose: Present bounded native tool and skill activity using Astryx ChatToolCalls and notices.
// Pattern: Native activity renderer with explicit paged original lookup.
// Usage: LiveOutput supplies only the selected conversation's activity.
// Related: webui/src/pages/workspace/ui/inspection/original.tsx, webui/src/pages/workspace/model/tool-status.ts.
import { ChatSystemMessage } from '@astryxdesign/core/Chat';
import { VStack } from '@astryxdesign/core/Layout';
import { memo } from 'react';
import { toolBatchIdentity, toolGroups, toolIdentity } from '@/pages/workspace/model/chat-events';
import { eventNotice } from '@/pages/workspace/model/event-notice';
import type { InspectionActions } from '@/pages/workspace/model/inspection';
import type { Target } from '@/pages/workspace/model/store';
import { ToolCalls } from '@/pages/workspace/ui/chat/tool-call';
import type { EventJson } from '@/shared/api';
import { useLocale } from '@/shared/i18n';
export const LiveActivity = memo(function LiveActivity({
  target,
  events,
  inspect,
}: {
  inspect: InspectionActions;
  target: Target;
  events: EventJson[];
}) {
  const t = useLocale();
  const entries = events.filter(
    (e) =>
      e.workspaceId === target.workspace &&
      e.sessionId === target.session &&
      (e.tool ||
        e.skill ||
        ['team_work', 'retry', 'compaction_started', 'compaction_finished'].includes(e.kind ?? ''))
  );
  const tools = entries.filter((event) => event.tool);
  if (!entries.length) return null;
  return (
    <VStack gap={3}>
      {toolGroups(tools)
        .slice(-12)
        .map((events) => (
          <ToolCalls
            key={
              toolBatchIdentity(events[0]) ??
              toolIdentity(events[0]) ??
              `${events[0]?.taskId}/${events[0]?.sequence}`
            }
            target={target}
            events={events}
            inspect={inspect}
          />
        ))}
      {entries
        .filter((event) => !event.tool)
        .slice(-12)
        .map((event) => {
          const notice = eventNotice(event, t);
          return notice ? (
            <ChatSystemMessage key={`${event.taskId}/${event.sequence}/${event.agent}`}>
              {event.agent || 'MECA'} · {notice}
            </ChatSystemMessage>
          ) : null;
        })}
    </VStack>
  );
});
