// Purpose: Present bounded live tools, authored team content and fixed notices in their original order.
// Pattern: Native activity renderer with explicit paged original lookup.
// Usage: LiveOutput supplies only the selected conversation's activity.
// Related: webui/src/pages/workspace/ui/inspection/original.tsx, webui/src/pages/workspace/model/tool-status.ts.
import { VStack } from '@astryxdesign/core/Layout';
import { memo } from 'react';
import { toolBatchIdentity, toolGroups, toolIdentity } from '@/pages/workspace/model/chat-events';
import { historyMessage } from '@/pages/workspace/model/history-rows';
import type { InspectionActions } from '@/pages/workspace/model/inspection';
import type { Target } from '@/pages/workspace/model/store';
import { HistoryRow } from '@/pages/workspace/ui/chat/history-row';
import { ToolCalls } from '@/pages/workspace/ui/chat/tool-call';
import type { EventJson } from '@/shared/api';
export const LiveActivity = memo(function LiveActivity({
  target,
  events,
  inspect,
}: {
  inspect: InspectionActions;
  target: Target;
  events: EventJson[];
}) {
  const entries = events.filter(
    (e) =>
      e.workspaceId === target.workspace &&
      e.sessionId === target.session &&
      (e.tool || e.work || e.notice || e.message)
  );
  if (!entries.length) return null;
  return (
    <VStack gap={6} width="100%" className="live-activity min-w-0">
      {toolGroups(entries)
        .slice(-24)
        .map((events) => {
          const event = events[0];
          if (!event) return null;
          const key =
            toolBatchIdentity(event) ??
            toolIdentity(event) ??
            event.historySequence ??
            event.cursor ??
            JSON.stringify([event.taskId, event.sequence, event.agent]);
          return event.tool ? (
            <ToolCalls key={key} target={target} events={events} />
          ) : (
            <HistoryRow
              key={key}
              target={target}
              item={historyMessage({
                sequence: event.historySequence,
                event,
                truncated: event.truncated,
              })}
              inspect={inspect}
            />
          );
        })}
    </VStack>
  );
});
