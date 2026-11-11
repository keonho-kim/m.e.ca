// Purpose: Present bounded native tool and skill activity using Astryx ChatToolCalls and notices.
// Pattern: Native activity renderer with explicit paged original lookup.
// Usage: LiveOutput supplies only the selected conversation's activity.
// Related: webui/src/pages/workspace/ui/inspection/original.tsx, webui/src/pages/workspace/model/tool-status.ts.
import { Button } from '@astryxdesign/core/Button';
import { ChatSystemMessage, ChatToolCalls } from '@astryxdesign/core/Chat';
import { VStack } from '@astryxdesign/core/Layout';
import { Text } from '@astryxdesign/core/Text';
import { memo, useState } from 'react';
import { toolIdentity } from '@/pages/workspace/model/chat-events';
import { type Target, useWorkspaceStore } from '@/pages/workspace/model/store';
import { isActiveTask } from '@/pages/workspace/model/task-state';
import { toolStatus } from '@/pages/workspace/model/tool-status';
import { Original } from '@/pages/workspace/ui/inspection/original';
import type { DetailRequestJson, EventJson } from '@/shared/api';
import { useLocale } from '@/shared/i18n';
export const LiveActivity = memo(function LiveActivity({
  target,
  events,
}: {
  target: Target;
  events: EventJson[];
}) {
  const t = useLocale(),
    [source, setSource] = useState<DetailRequestJson | null>(null);
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
  if (!entries.length && !source) return null;
  return (
    <VStack gap={3}>
      {!!tools.length && (
        <ChatToolCalls
          className="tool-calls"
          style={{ width: '100%', minWidth: 0, maxWidth: '100%' }}
          calls={tools.map((event) => ({
            key: toolIdentity(event) ?? `${event.taskId}/${event.agent}/${event.sequence}`,
            name: event.tool?.name ?? t('toolResult'),
            status: toolStatus(event.kind),
            node: event.agent,
            resultDetail: (
              <VStack gap={2}>
                <Text>{event.text?.slice(0, 1024)}</Text>
                {event.tool?.reference && (
                  <Button
                    label={t('details')}
                    size="sm"
                    onClick={() => setSource({ tool: event.tool?.reference })}
                  />
                )}
              </VStack>
            ),
          }))}
        />
      )}
      {entries
        .filter((event) => !event.tool)
        .map((event) => (
          <ChatSystemMessage key={`${event.taskId}/${event.sequence}/${event.agent}`}>
            {event.agent || 'MECA'} · {event.skill?.name || event.kind} {event.text?.slice(0, 1024)}
          </ChatSystemMessage>
        ))}
      {source && <Original target={target} source={source} onClose={() => setSource(null)} />}
    </VStack>
  );
});
