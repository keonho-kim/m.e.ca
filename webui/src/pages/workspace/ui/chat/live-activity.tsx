// Purpose: Present bounded native tool and skill activity using Astryx ChatToolCalls and notices.
// Pattern: Native activity renderer with explicit paged original lookup.
// Usage: LiveOutput supplies only the selected conversation's activity.
// Related: webui/src/pages/workspace/ui/inspection/original.tsx, webui/src/pages/workspace/model/tool-status.ts.
import { ChatSystemMessage } from '@astryxdesign/core/Chat';
import { VStack } from '@astryxdesign/core/Layout';
import { memo } from 'react';
import { toolIdentity } from '@/pages/workspace/model/chat-events';
import type { InspectionActions } from '@/pages/workspace/model/inspection';
import { type Target, useWorkspaceStore } from '@/pages/workspace/model/store';
import { isActiveTask } from '@/pages/workspace/model/task-state';
import { ToolCall } from '@/pages/workspace/ui/chat/tool-call';
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
  const tasks = useWorkspaceStore((state) => state.tasks);
  const entries = events
    .filter(
      (e) =>
        (!e.taskId || !tasks[e.taskId] || isActiveTask(tasks[e.taskId]?.status ?? 'running')) &&
        e.workspaceId === target.workspace &&
        e.sessionId === target.session &&
        (e.tool ||
          e.skill ||
          [
            'team_notice',
            'team_work',
            'retry',
            'compaction_started',
            'compaction_finished',
          ].includes(e.kind ?? ''))
    )
    .slice(-12);
  const tools = entries.filter((event) => event.tool);
  if (!entries.length) return null;
  return (
    <VStack gap={3}>
      {tools.map((event) => (
        <ToolCall
          key={toolIdentity(event) ?? `${event.taskId}/${event.sequence}`}
          target={target}
          event={event}
          inspect={inspect}
        />
      ))}
      {entries
        .filter((event) => !event.tool)
        .map((event) => (
          <ChatSystemMessage key={`${event.taskId}/${event.sequence}/${event.agent}`}>
            {event.agent || 'MECA'} · {event.skill?.name || event.kind} {event.text?.slice(0, 1024)}
          </ChatSystemMessage>
        ))}
    </VStack>
  );
});
