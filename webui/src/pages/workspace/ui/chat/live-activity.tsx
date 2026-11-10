// Purpose: Present bounded native tool and skill activity using Astryx ChatToolCalls and notices.
// Pattern: Latest-call projection with explicit paged original lookup.
// Usage: LiveOutput supplies only the selected conversation's activity.
// Related: webui/src/pages/workspace/ui/inspection/original.tsx, webui/src/pages/workspace/model/tool-status.ts.
import { Button } from '@astryxdesign/core/Button';
import { ChatSystemMessage, ChatToolCalls } from '@astryxdesign/core/Chat';
import { VStack } from '@astryxdesign/core/Layout';
import { Text } from '@astryxdesign/core/Text';
import { memo, useState } from 'react';
import type { Target } from '@/pages/workspace/model/store';
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
  const entries = events
    .filter(
      (e) =>
        e.workspaceId === target.workspace &&
        e.sessionId === target.session &&
        (e.tool ||
          e.skill ||
          [
            'team_notice',
            'team_work',
            'progress',
            'team_progress',
            'retry',
            'compaction_started',
            'compaction_finished',
          ].includes(e.kind ?? ''))
    )
    .slice(-12);
  const tools = new Map<string, EventJson>();
  for (const event of entries)
    if (event.tool) tools.set(`${event.taskId}/${event.agent}/${event.tool.callId}`, event);
  return (
    <VStack gap={3}>
      {!!tools.size && (
        <ChatToolCalls
          calls={[...tools].map(([key, event]) => ({
            key,
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
