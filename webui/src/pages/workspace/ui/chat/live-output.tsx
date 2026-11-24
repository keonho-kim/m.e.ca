// Purpose: Merge saved and streaming rows for active requests without changing their display positions.
// Pattern: Fine-grained live presentation subscriber.
// Usage: Chat mounts this for its selected conversation.
// Related: webui/src/pages/workspace/model/stream-projection.ts, webui/src/pages/workspace/model/inspection.ts, webui/src/pages/workspace/model/live-timeline.ts.

import { VStack } from '@astryxdesign/core/Layout';
import { Selector } from '@astryxdesign/core/Selector';
import { useMemo, useState, useSyncExternalStore } from 'react';
import { historyMessage, historyRows } from '@/pages/workspace/model/history-rows';
import type { InspectionActions } from '@/pages/workspace/model/inspection';
import { liveTimeline } from '@/pages/workspace/model/live-timeline';
import { useChatPresentationBlocked } from '@/pages/workspace/model/presentation';
import { type Target, useWorkspaceStore } from '@/pages/workspace/model/store';
import { HistoryRow } from '@/pages/workspace/ui/chat/history-row';
import type { HistoryItemJson } from '@/shared/api';
import { useLocale } from '@/shared/i18n';
export function LiveOutput({
  target,
  history,
  inspect,
}: {
  target: Target;
  history: HistoryItemJson[];
  inspect: InspectionActions;
}) {
  const t = useLocale(),
    visible = useVisibleOutput();
  const { drafts, events } = visible;
  const [agent, setAgent] = useState('');
  const selected = useWorkspaceStore((s) => s.selection);
  const blocked = useChatPresentationBlocked();
  const streaming = useWorkspaceStore((s) => s.streamStatus === 'connected');
  const sources =
    selected.workspace === target.workspace && selected.session === target.session
      ? Object.values(drafts)
      : [];
  const agents = [...new Set(sources.map((value) => value.agent).filter(Boolean))];
  if (agent && !agents.includes(agent)) setAgent('');
  const savedRows = useMemo(() => historyRows(history), [history]);
  const timeline = useMemo(
    () =>
      liveTimeline(
        sources,
        events.filter(
          (event) => event.workspaceId === target.workspace && event.sessionId === target.session
        ),
        history,
        Object.keys(visible.tasks),
        savedRows
      ),
    [sources, events, history, target, visible.tasks, savedRows]
  );
  if (!sources.length && !events.length) return null;
  return (
    <VStack width="100%" className="live-output" gap={8} paddingBlockStart={8}>
      {agents.length > 1 && (
        <Selector
          label={t('agent')}
          value={agent}
          onChange={setAgent}
          options={[
            { value: '', label: t('allAgents') },
            ...agents.map((value) => ({ value, label: value })),
          ]}
        />
      )}
      {timeline.map((entry) => {
        if (entry.kind === 'draft') {
          const d = entry.draft;
          if (agent && d.agent !== agent) return null;
          return (
            <HistoryRow
              key={entry.key}
              target={target}
              inspect={inspect}
              streaming={!blocked && streaming && d.doneRevision === undefined}
              item={{
                presentation: d.presentation,
                role: 'assistant',
                text: d.text,
                event: { kind: 'assistant_delta', agent: d.agent },
              }}
              thinking={
                d.thinking
                  ? {
                      event: {
                        kind: 'thinking',
                        agent: d.agent,
                        message: { role: 'assistant', thinking: d.thinking },
                      },
                    }
                  : undefined
              }
            />
          );
        }
        if (entry.kind === 'history') {
          const row = entry.row;
          if ('completion' in row) return null; // Completion is released by Chat after durable handoff.
          if (agent && row.item.event?.agent !== agent) return null;
          return <HistoryRow key={entry.key} target={target} inspect={inspect} {...row} />;
        }
        const event = entry.events[0];
        if (!event || (agent && event.agent !== agent)) return null;
        const values = entry.events.map((event) =>
          historyMessage({
            sequence: event.historySequence,
            presentation: event.presentation,
            event,
            truncated: event.truncated,
          })
        );
        return (
          <HistoryRow
            key={entry.key}
            target={target}
            item={values[0]!}
            tools={event.tool ? values : undefined}
            saved={false}
            inspect={inspect}
          />
        );
      })}
    </VStack>
  );
}

// Keep one immutable projection while covered. The canonical store continues
// processing approvals, completion and queues; no copied transcript is retained.
function useVisibleOutput() {
  const blocked = useChatPresentationBlocked();
  const [owner] = useState(() => {
    const initial = useWorkspaceStore.getState();
    let previous = { drafts: initial.drafts, events: initial.events, tasks: initial.tasks };
    return {
      read: (paused: boolean) => {
        const next = useWorkspaceStore.getState();
        if (
          (!paused || (!Object.keys(next.drafts).length && !next.events.length)) &&
          (next.drafts !== previous.drafts || next.events !== previous.events)
        )
          previous = { drafts: next.drafts, events: next.events, tasks: next.tasks };
        return previous;
      },
    };
  });
  return useSyncExternalStore(useWorkspaceStore.subscribe, () => owner.read(blocked));
}
